package cronexpr

import (
	"errors"
	"math/bits"
	"time"
)

var (
	// ErrReboot is returned by Next/Prev for @reboot, which has no calendar times.
	ErrReboot = errors.New("@reboot runs only at startup and has no calendar times")
	// ErrNeverFires is returned when no calendar date satisfies the expression.
	ErrNeverFires = errors.New("expression can never fire: no calendar date matches it")
	// ErrNoMatch is returned when no match was found within the search bound.
	ErrNoMatch = errors.New("no matching time found within the search bound")
)

// searchYears bounds Next/Prev. Leap-day schedules restricted further by a
// "*/n" day-of-week can need well over a decade to recur.
const searchYears = 40

// DayMatches reports whether the calendar day of t satisfies the month,
// day-of-month and day-of-week fields, using the Vixie OR rule.
func (s *Schedule) DayMatches(t time.Time) bool {
	return s.dayMatch(t.Month(), t.Day(), t.Weekday())
}

func (s *Schedule) dayMatch(m time.Month, d int, wd time.Weekday) bool {
	if !s.Fields[Month].Has(int(m)) {
		return false
	}
	dom, dow := s.Fields[DayOfMonth], s.Fields[DayOfWeek]
	if dom.Star || dow.Star {
		return dom.Has(d) && dow.Has(int(wd))
	}
	return dom.Has(d) || dow.Has(int(wd))
}

// Match reports whether the wall-clock minute of t (in t's location) fires.
func (s *Schedule) Match(t time.Time) bool {
	if s.Reboot() {
		return false
	}
	return s.Fields[Minute].Has(t.Minute()) && s.Fields[Hour].Has(t.Hour()) && s.DayMatches(t)
}

// Possible reports whether at least one calendar date satisfies the
// expression.
func (s *Schedule) Possible() bool { return !s.never }

// possible checks a full 28-year weekday/leap-year cycle.
func (s *Schedule) possible() bool {
	if s.Reboot() {
		return true
	}
	d := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 28*366; i++ {
		if s.DayMatches(d) {
			return true
		}
		d = d.AddDate(0, 0, 1)
	}
	return false
}

// Next returns the first firing time strictly after from, in from's
// location. It jumps month-, day-, hour- and minute-wise instead of
// scanning every minute, and gives up after a bounded number of years.
func (s *Schedule) Next(from time.Time) (time.Time, error) {
	if s.Reboot() {
		return time.Time{}, ErrReboot
	}
	if !s.Possible() {
		return time.Time{}, ErrNeverFires
	}
	loc := from.Location()
	t := from.Truncate(time.Minute).Add(time.Minute)
	limit := from.AddDate(searchYears, 0, 0)
	minutes := s.Fields[Minute].Bits
	for t.Before(limit) {
		y, mo, d := t.Date()
		if !s.Fields[Month].Has(int(mo)) {
			t = time.Date(y, mo+1, 1, 0, 0, 0, 0, loc)
			continue
		}
		if !s.DayMatches(t) {
			t = time.Date(y, mo, d+1, 0, 0, 0, 0, loc)
			continue
		}
		m := t.Minute()
		if !s.Fields[Hour].Has(t.Hour()) {
			t = t.Add(time.Duration(60-m) * time.Minute)
			continue
		}
		if rest := minutes >> uint(m); rest != 0 {
			nm := m + bits.TrailingZeros64(rest)
			if nm == m {
				return t, nil
			}
			// Re-check every field: an offset change may happen mid-hour.
			t = t.Add(time.Duration(nm-m) * time.Minute)
			continue
		}
		t = t.Add(time.Duration(60-m) * time.Minute)
	}
	return time.Time{}, ErrNoMatch
}

// Prev returns the last firing time strictly before from.
func (s *Schedule) Prev(from time.Time) (time.Time, error) {
	if s.Reboot() {
		return time.Time{}, ErrReboot
	}
	if !s.Possible() {
		return time.Time{}, ErrNeverFires
	}
	loc := from.Location()
	t := from.Truncate(time.Minute)
	if t.Equal(from) {
		t = t.Add(-time.Minute)
	}
	limit := from.AddDate(-searchYears, 0, 0)
	minutes := s.Fields[Minute].Bits
	for t.After(limit) {
		y, mo, d := t.Date()
		if !s.Fields[Month].Has(int(mo)) {
			t = time.Date(y, mo, 1, 0, 0, 0, 0, loc).Add(-time.Minute)
			continue
		}
		if !s.DayMatches(t) {
			t = time.Date(y, mo, d, 0, 0, 0, 0, loc).Add(-time.Minute)
			continue
		}
		m := t.Minute()
		if !s.Fields[Hour].Has(t.Hour()) {
			t = t.Add(-time.Duration(m+1) * time.Minute)
			continue
		}
		if low := minutes & (1<<uint(m+1) - 1); low != 0 {
			pm := 63 - bits.LeadingZeros64(low)
			if pm == m {
				return t, nil
			}
			t = t.Add(-time.Duration(m-pm) * time.Minute)
			continue
		}
		t = t.Add(-time.Duration(m+1) * time.Minute)
	}
	return time.Time{}, ErrNoMatch
}

// NextN returns up to n consecutive firing times after from.
func (s *Schedule) NextN(from time.Time, n int) ([]time.Time, error) {
	out := make([]time.Time, 0, n)
	t := from
	for len(out) < n {
		next, err := s.Next(t)
		if err != nil {
			return out, err
		}
		out = append(out, next)
		t = next
	}
	return out, nil
}
