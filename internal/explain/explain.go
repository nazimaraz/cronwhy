// Package explain turns parsed cron schedules into plain English.
package explain

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nazimaraz/cronwhy/internal/cronexpr"
)

var monthNames = []string{"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

// Describe returns a one-line English description of s, e.g.
// "At every 15th minute, from 09:00 through 17:59, Monday through Friday".
func Describe(s *cronexpr.Schedule) string {
	if s.Reboot() {
		return "At system startup"
	}
	f := s.Fields
	var parts []string
	parts = append(parts, describeTime(f[cronexpr.Minute], f[cronexpr.Hour])...)

	dom, dow := f[cronexpr.DayOfMonth], f[cronexpr.DayOfWeek]
	domDesc, dowDesc := describeDOM(dom), describeDOW(dow)
	switch {
	case domDesc != "" && dowDesc != "" && !dom.Star && !dow.Star:
		parts = append(parts, domDesc+" or "+strings.TrimPrefix(dowDesc, "on "))
	default:
		if domDesc != "" {
			parts = append(parts, domDesc)
		}
		if dowDesc != "" {
			parts = append(parts, dowDesc)
		}
	}
	if m := describeMonth(f[cronexpr.Month]); m != "" {
		parts = append(parts, m)
	}
	return strings.Join(parts, ", ")
}

// Ordinal formats n as "1st", "2nd", "11th", ...
func Ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(n) + suffix
}

// JoinList joins items as "a", "a and b" or "a, b, and c".
func JoinList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
}

func allSingles(f cronexpr.Field) bool {
	for _, t := range f.Terms {
		if t.IsRange || t.HasStep {
			return false
		}
	}
	return true
}

// plainRange reports whether f is a single "a-b" term without a step.
func plainRange(f cronexpr.Field) (cronexpr.Term, bool) {
	if len(f.Terms) == 1 && f.Terms[0].IsRange && !f.Terms[0].Star && f.Terms[0].Step == 1 {
		return f.Terms[0], true
	}
	return cronexpr.Term{}, false
}

func unrestricted(f cronexpr.Field) bool {
	return f.Full() && f.Star
}

// describeField describes the terms of f generically. noun is e.g. "minute";
// named fields (month, weekday) pass named=true to drop the noun on values.
func describeField(f cronexpr.Field, noun string, val func(int) string, named bool) string {
	if allSingles(f) {
		var vals []string
		for _, t := range f.Terms {
			vals = append(vals, val(t.Start))
		}
		if named {
			return JoinList(vals)
		}
		return noun + " " + JoinList(vals)
	}
	var terms []string
	for _, t := range f.Terms {
		every := "every " + noun
		if t.Step > 1 {
			every = "every " + Ordinal(t.Step) + " " + noun
		}
		switch {
		case t.Star:
			terms = append(terms, every)
		case t.IsRange || t.HasStep:
			terms = append(terms, fmt.Sprintf("%s from %s through %s", every, val(t.Start), val(t.End)))
		case named:
			terms = append(terms, val(t.Start))
		default:
			terms = append(terms, noun+" "+val(t.Start))
		}
	}
	return JoinList(terms)
}

func describeTime(min, hour cronexpr.Field) []string {
	if allSingles(min) && allSingles(hour) && len(min.Terms)*len(hour.Terms) <= 8 {
		var times []string
		for _, h := range hour.Values() {
			for _, m := range min.Values() {
				times = append(times, fmt.Sprintf("%02d:%02d", h, m))
			}
		}
		return []string{"At " + JoinList(times)}
	}
	minDesc := "At " + describeField(min, "minute", strconv.Itoa, false)
	if unrestricted(min) {
		minDesc = "At every minute"
	}
	if unrestricted(hour) {
		return []string{minDesc}
	}
	if t, ok := plainRange(hour); ok {
		return []string{minDesc, fmt.Sprintf("from %02d:00 through %02d:59", t.Start, t.End)}
	}
	return []string{minDesc, "past " + describeField(hour, "hour", strconv.Itoa, false)}
}

func describeDOM(f cronexpr.Field) string {
	if unrestricted(f) {
		return ""
	}
	return "on " + describeField(f, "day-of-month", strconv.Itoa, false)
}

func weekday(v int) string { return time.Weekday(v % 7).String() }

func describeDOW(f cronexpr.Field) string {
	if unrestricted(f) {
		return ""
	}
	if t, ok := plainRange(f); ok {
		return weekday(t.Start) + " through " + weekday(t.End)
	}
	return "on " + describeField(f, "day-of-week", weekday, true)
}

func describeMonth(f cronexpr.Field) string {
	if unrestricted(f) {
		return ""
	}
	if t, ok := plainRange(f); ok {
		return "from " + monthNames[t.Start] + " through " + monthNames[t.End]
	}
	return "in " + describeField(f, "month", func(v int) string { return monthNames[v] }, true)
}

// CompactValues renders the selected values of f with runs collapsed,
// e.g. "0,15,30,45" or "9-17".
func CompactValues(f cronexpr.Field) string {
	vals := f.Values()
	var out []string
	for i := 0; i < len(vals); {
		j := i
		for j+1 < len(vals) && vals[j+1] == vals[j]+1 {
			j++
		}
		switch {
		case j-i >= 2:
			out = append(out, fmt.Sprintf("%d-%d", vals[i], vals[j]))
		case j > i:
			out = append(out, strconv.Itoa(vals[i]), strconv.Itoa(vals[j]))
		default:
			out = append(out, strconv.Itoa(vals[i]))
		}
		i = j + 1
	}
	return strings.Join(out, ",")
}

// Relative formats d as "in 2h 13m", "in 3d 4h" or "in 45m".
func Relative(d time.Duration) string {
	if d < time.Minute {
		return "in <1m"
	}
	mins := int(d / time.Minute)
	days, hours, m := mins/1440, mins%1440/60, mins%60
	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("in %dd %dh", days, hours)
	case days > 0:
		return fmt.Sprintf("in %dd", days)
	case hours > 0:
		return fmt.Sprintf("in %dh %dm", hours, m)
	}
	return fmt.Sprintf("in %dm", m)
}
