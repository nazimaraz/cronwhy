package explain

import (
	"testing"
	"time"

	"github.com/nazimaraz/cronwhy/internal/cronexpr"
)

func TestDescribe(t *testing.T) {
	tests := []struct{ expr, want string }{
		{"*/15 9-17 * * 1-5", "At every 15th minute, from 09:00 through 17:59, Monday through Friday"},
		{"* * * * *", "At every minute"},
		{"*/5 * * * *", "At every 5th minute"},
		{"0 0 * * *", "At 00:00"},
		{"@daily", "At 00:00"},
		{"@reboot", "At system startup"},
		{"30 4 1,15 * *", "At 04:30, on day-of-month 1 and 15"},
		{"0 9,12,18 * * *", "At 09:00, 12:00, and 18:00"},
		{"5 * * * *", "At minute 5"},
		{"10-50/10 */2 * * *", "At every 10th minute from 10 through 50, past every 2nd hour"},
		{"0 0 1 * 1", "At 00:00, on day-of-month 1 or Monday"},
		{"0 0 * JAN,JUL SUN", "At 00:00, on Sunday, in January and July"},
		{"0 8 * 3-5 *", "At 08:00, from March through May"},
		{"0 0 */2 * *", "At 00:00, on every 2nd day-of-month"},
		{"0,30-40 * * * *", "At minute 0 and every minute from 30 through 40"},
		{"* 3 * * *", "At every minute, past hour 3"},
	}
	for _, tt := range tests {
		s, err := cronexpr.Parse(tt.expr)
		if err != nil {
			t.Fatal(err)
		}
		if got := Describe(s); got != tt.want {
			t.Errorf("Describe(%q)\n got  %q\n want %q", tt.expr, got, tt.want)
		}
	}
}

func TestOrdinal(t *testing.T) {
	for n, want := range map[int]string{1: "1st", 2: "2nd", 3: "3rd", 4: "4th", 11: "11th", 12: "12th", 13: "13th", 21: "21st", 22: "22nd"} {
		if got := Ordinal(n); got != want {
			t.Errorf("Ordinal(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestRelative(t *testing.T) {
	tests := map[time.Duration]string{
		30 * time.Second:           "in <1m",
		45 * time.Minute:           "in 45m",
		133 * time.Minute:          "in 2h 13m",
		(3*24 + 4) * time.Hour:     "in 3d 4h",
		2 * 24 * time.Hour:         "in 2d",
		(24*time.Hour + time.Hour): "in 1d 1h",
	}
	for d, want := range tests {
		if got := Relative(d); got != want {
			t.Errorf("Relative(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestCompactValues(t *testing.T) {
	s, _ := cronexpr.Parse("0,15,30,45 9-17 1,2 * 1-5")
	tests := map[cronexpr.FieldKind]string{
		cronexpr.Minute: "0,15,30,45", cronexpr.Hour: "9-17", cronexpr.DayOfMonth: "1,2", cronexpr.DayOfWeek: "1-5",
	}
	for k, want := range tests {
		if got := CompactValues(s.Fields[k]); got != want {
			t.Errorf("%s: %q, want %q", k, got, want)
		}
	}
}
