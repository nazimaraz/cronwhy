package cronexpr

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

var refTime = time.Date(2026, 10, 27, 20, 0, 0, 0, time.UTC)

func mustLoad(t testing.TB, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return loc
}

func TestNext(t *testing.T) {
	ist := mustLoad(t, "Europe/Istanbul")
	ny := mustLoad(t, "America/New_York")
	lon := mustLoad(t, "Europe/London")
	tests := []struct {
		expr string
		from time.Time
		want []string // formatted "2006-01-02 15:04 MST"
	}{
		{"*/15 9-17 * * 1-5", time.Date(2026, 10, 27, 20, 0, 0, 0, ist),
			[]string{"2026-10-28 09:00 +03", "2026-10-28 09:15 +03", "2026-10-28 09:30 +03"}},
		{"0 0 1 * *", time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC),
			[]string{"2027-01-01 00:00 UTC", "2027-02-01 00:00 UTC"}},
		{"0 0 29 2 *", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			[]string{"2028-02-29 00:00 UTC", "2032-02-29 00:00 UTC"}},
		// Vixie OR: the 13th or any Friday.
		{"0 12 13 * 5", time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
			[]string{"2026-11-06 12:00 UTC", "2026-11-13 12:00 UTC", "2026-11-20 12:00 UTC"}},
		// Spring forward: 02:30 does not exist on 2027-03-14 in New York.
		{"30 2 * * *", time.Date(2027, 3, 13, 12, 0, 0, 0, ny),
			[]string{"2027-03-15 02:30 EDT"}},
		// Fall back: 01:30 happens twice on 2026-11-01 in New York.
		{"30 1 * * *", time.Date(2026, 11, 1, 0, 0, 0, 0, ny),
			[]string{"2026-11-01 01:30 EDT", "2026-11-01 01:30 EST", "2026-11-02 01:30 EST"}},
		// London falls back at 02:00 BST -> 01:00 GMT on 2026-10-25.
		{"15 1 25 10 *", time.Date(2026, 10, 24, 0, 0, 0, 0, lon),
			[]string{"2026-10-25 01:15 BST", "2026-10-25 01:15 GMT"}},
		{"@yearly", time.Date(2026, 10, 27, 0, 0, 0, 0, ist),
			[]string{"2027-01-01 00:00 +03"}},
	}
	for _, tt := range tests {
		s, err := Parse(tt.expr)
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.NextN(tt.from, len(tt.want))
		if err != nil {
			t.Errorf("%q: %v", tt.expr, err)
			continue
		}
		var gs []string
		for _, g := range got {
			gs = append(gs, g.Format("2006-01-02 15:04 MST"))
		}
		if strings.Join(gs, "|") != strings.Join(tt.want, "|") {
			t.Errorf("%q from %v:\n got  %v\n want %v", tt.expr, tt.from, gs, tt.want)
		}
	}
}

func TestNextImpossible(t *testing.T) {
	s, _ := Parse("0 0 31 2 *")
	start := time.Now()
	if _, err := s.Next(refTime); !errors.Is(err, ErrNeverFires) {
		t.Fatalf("err = %v, want ErrNeverFires", err)
	}
	if _, err := s.Prev(refTime); !errors.Is(err, ErrNeverFires) {
		t.Fatalf("Prev err = %v, want ErrNeverFires", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("impossible expression took too long")
	}
}

func TestPrev(t *testing.T) {
	s, _ := Parse("*/15 9-17 * * 1-5")
	ist := mustLoad(t, "Europe/Istanbul")
	got, err := s.Prev(time.Date(2026, 10, 26, 9, 0, 0, 0, ist)) // Monday 09:00
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 10, 23, 17, 45, 0, 0, ist); !got.Equal(want) {
		t.Errorf("Prev = %v, want %v", got, want)
	}
}

// slowNext is the minute-by-minute reference implementation.
func slowNext(s *Schedule, from time.Time, horizon time.Duration) (time.Time, bool) {
	t := from.Truncate(time.Minute).Add(time.Minute)
	end := from.Add(horizon)
	for ; !t.After(end); t = t.Add(time.Minute) {
		if s.Match(t) {
			return t, true
		}
	}
	return time.Time{}, false
}

func randField(r *rand.Rand, k FieldKind) string {
	lo, hi := k.Min(), k.Max()
	pick := func() int { return lo + r.Intn(hi-lo+1) }
	switch r.Intn(6) {
	case 0:
		return "*"
	case 1:
		return fmt.Sprintf("*/%d", 1+r.Intn(hi-lo+1))
	case 2:
		return fmt.Sprint(pick())
	case 3:
		a, b := pick(), pick()
		if a > b {
			a, b = b, a
		}
		return fmt.Sprintf("%d-%d", a, b)
	case 4:
		a, b := pick(), pick()
		if a > b {
			a, b = b, a
		}
		return fmt.Sprintf("%d-%d/%d", a, b, 1+r.Intn(5))
	default:
		return fmt.Sprintf("%d,%d,%d", pick(), pick(), pick())
	}
}

func TestNextProperty(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	locs := []*time.Location{
		mustLoad(t, "Europe/Istanbul"), mustLoad(t, "America/New_York"),
		mustLoad(t, "Europe/London"), time.UTC,
	}
	// Start points around DST transitions plus random ones.
	starts := []time.Time{
		time.Date(2027, 3, 14, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC),
		time.Date(2027, 3, 28, 0, 0, 0, 0, time.UTC),
	}
	const horizon = 45 * 24 * time.Hour
	iterations := 300
	if testing.Short() {
		iterations = 50
	}
	for i := 0; i < iterations; i++ {
		parts := make([]string, 5)
		for k := range parts {
			parts[k] = randField(r, FieldKind(k))
		}
		expr := strings.Join(parts, " ")
		s, err := Parse(expr)
		if err != nil {
			t.Fatalf("generated invalid %q: %v", expr, err)
		}
		loc := locs[r.Intn(len(locs))]
		from := starts[r.Intn(len(starts))].Add(time.Duration(r.Intn(60*24*90)) * time.Minute).In(loc)
		got, err := s.Next(from)
		want, found := slowNext(s, from, horizon)
		switch {
		case !s.Possible():
			if found {
				t.Fatalf("%q: Possible()=false but reference found %v", expr, want)
			}
		case err != nil:
			t.Fatalf("%q from %v: unexpected error %v", expr, from, err)
		case found && !got.Equal(want):
			t.Fatalf("%q from %v (%s): Next=%v, reference=%v", expr, from, loc, got, want)
		case !found && !got.After(from.Add(horizon)):
			t.Fatalf("%q from %v: Next=%v but reference found nothing in horizon", expr, from, got)
		case !s.Match(got):
			t.Fatalf("%q: Next=%v does not match", expr, got)
		}
	}
}
