package cronexpr

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	tests := []struct {
		expr  string
		field FieldKind
		want  []int
	}{
		{"*/15 * * * *", Minute, []int{0, 15, 30, 45}},
		{"10-50/10 * * * *", Minute, []int{10, 20, 30, 40, 50}},
		{"1,5,9 * * * *", Minute, []int{1, 5, 9}},
		{"0 9-12 * * *", Hour, []int{9, 10, 11, 12}},
		{"0 0 * JAN,jul *", Month, []int{1, 7}},
		{"0 0 * * MON-FRI", DayOfWeek, []int{1, 2, 3, 4, 5}},
		{"0 0 * * 7", DayOfWeek, []int{0}},
		{"0 0 * * 5-7", DayOfWeek, []int{0, 5, 6}},
		{"0 0 * * 0-7", DayOfWeek, []int{0, 1, 2, 3, 4, 5, 6}},
		{"0 0 * * sun", DayOfWeek, []int{0}},
		{"0 0 1-31/10 * *", DayOfMonth, []int{1, 11, 21, 31}},
		{"@hourly", Minute, []int{0}},
		{"@weekly", DayOfWeek, []int{0}},
		{"@YEARLY", Month, []int{1}},
	}
	for _, tt := range tests {
		s, err := Parse(tt.expr)
		if err != nil {
			t.Errorf("Parse(%q): %v", tt.expr, err)
			continue
		}
		if got := s.Fields[tt.field].Values(); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Parse(%q) %s = %v, want %v", tt.expr, tt.field, got, tt.want)
		}
	}
}

func TestParseReboot(t *testing.T) {
	s, err := Parse("@reboot")
	if err != nil || !s.Reboot() {
		t.Fatalf("Parse(@reboot) = %v, %v", s, err)
	}
	if _, err := s.Next(refTime); !errors.Is(err, ErrReboot) {
		t.Errorf("Next on @reboot: err = %v", err)
	}
}

func TestParseInvalid(t *testing.T) {
	tests := []struct {
		expr  string
		field FieldKind
		frag  string
	}{
		{"", NoField, "empty"},
		{"* * * *", NoField, "expected 5 fields"},
		{"0 * * * * *", NoField, "not supported by standard cron"},
		{"60 * * * *", Minute, "out of range 0-59"},
		{"0 24 * * *", Hour, "out of range 0-23"},
		{"0 0 0 * *", DayOfMonth, "out of range 1-31"},
		{"0 0 * 13 *", Month, "out of range 1-12"},
		{"0 0 * * 8", DayOfWeek, "out of range 0-7"},
		{"0 5-1 * * *", Hour, "reversed range"},
		{"*/0 * * * *", Minute, "positive integer"},
		{"*/x * * * *", Minute, "positive integer"},
		{"*/61 * * * *", Minute, "larger than"},
		{"5/15 * * * *", Minute, "single value"},
		{"0 0 * FOO *", Month, "unknown month"},
		{"0 0 * * MOO", DayOfWeek, "unknown day"},
		{"abc * * * *", Minute, "not a number"},
		{"0 0 L * *", DayOfMonth, "not supported by standard cron"},
		{"0 0 15W * *", DayOfMonth, "not supported by standard cron"},
		{"0 0 * * 5L", DayOfWeek, "not supported by standard cron"},
		{"0 0 * * MON#2", DayOfWeek, "not supported by standard cron"},
		{"0 0 ? * MON", DayOfMonth, "not supported by standard cron"},
		{"1,,2 * * * *", Minute, "empty element"},
		{"1-2-3 * * * *", Minute, "not a number"},
		{"99999999999 * * * *", Minute, "out of range"},
		{"@fortnightly", NoField, "unknown macro"},
	}
	for _, tt := range tests {
		_, err := Parse(tt.expr)
		var pe *ParseError
		if !errors.As(err, &pe) {
			t.Errorf("Parse(%q): err = %v, want *ParseError", tt.expr, err)
			continue
		}
		if pe.Field != tt.field || !strings.Contains(pe.Error(), tt.frag) {
			t.Errorf("Parse(%q) = field %v %q, want field %v containing %q", tt.expr, pe.Field, pe.Error(), tt.field, tt.frag)
		}
	}
}

func TestStarFlag(t *testing.T) {
	s, _ := Parse("0 0 */2 * 1")
	if !s.Fields[DayOfMonth].Star || s.Fields[DayOfWeek].Star {
		t.Errorf("star flags wrong: dom=%v dow=%v", s.Fields[DayOfMonth].Star, s.Fields[DayOfWeek].Star)
	}
}

func TestPossible(t *testing.T) {
	tests := map[string]bool{
		"0 0 31 2 *":      false,
		"0 0 30 2 *":      false,
		"0 0 31 4,6,9 *":  false,
		"0 0 29 2 *":      true,
		"0 0 31 2 1":      true, // OR: any Monday in February
		"0 0 31 2 */1":    false,
		"0 0 31 1-2 *":    true,
		"* * * * *":       true,
		"0 0 29 2 */2":    true,
		"0 0 1 * MON-FRI": true,
	}
	for expr, want := range tests {
		s, err := Parse(expr)
		if err != nil {
			t.Fatal(err)
		}
		if got := s.Possible(); got != want {
			t.Errorf("Possible(%q) = %v, want %v", expr, got, want)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"* * * * *", "*/15 9-17 * * 1-5", "0 0 31 2 *", "@daily", "5/15 * * * *",
		"0 0 L * *", "1-2-3,4/5 */0 * * SUN-SAT", "0 0 * * 7", "@reboot", "1,,,", "-1 * * * *",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, expr string) {
		s, err := Parse(expr)
		if err != nil {
			return
		}
		_ = s.String()
		if s.Possible() && !s.Reboot() {
			next, err := s.Next(refTime)
			if err == nil && !s.Match(next) {
				t.Fatalf("Next(%q) = %v does not match", expr, next)
			}
		}
	})
}
