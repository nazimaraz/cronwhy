// Package cronexpr parses five-field cron expressions with Vixie cron
// semantics and computes their firing times.
package cronexpr

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

// FieldKind identifies one of the five cron fields.
type FieldKind int

// The five cron fields, in expression order.
const (
	Minute FieldKind = iota
	Hour
	DayOfMonth
	Month
	DayOfWeek

	// NoField marks errors that are not tied to a single field.
	NoField FieldKind = -1
)

type fieldSpec struct {
	name     string
	min, max int
	names    map[string]int
}

var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

var dayNames = map[string]int{
	"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6,
}

var specs = [5]fieldSpec{
	{"minute", 0, 59, nil},
	{"hour", 0, 23, nil},
	{"day-of-month", 1, 31, nil},
	{"month", 1, 12, monthNames},
	{"day-of-week", 0, 7, dayNames},
}

// String returns the human name of the field, e.g. "day-of-month".
func (k FieldKind) String() string {
	if k < Minute || k > DayOfWeek {
		return "expression"
	}
	return specs[k].name
}

// Min returns the smallest value accepted by the field.
func (k FieldKind) Min() int { return specs[k].min }

// Max returns the largest value accepted by the field (7 for day-of-week).
func (k FieldKind) Max() int { return specs[k].max }

// Macros maps the supported @-macros to their five-field equivalent.
// @reboot has no schedule and maps to "".
var Macros = map[string]string{
	"@reboot":   "",
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// ParseError describes why an expression could not be parsed.
type ParseError struct {
	Field  FieldKind // NoField when the problem is not field-specific
	Value  string    // offending field text (or the whole expression)
	Reason string
	Hint   string // optional suggested fix
}

func (e *ParseError) Error() string {
	if e.Field == NoField {
		return e.Reason
	}
	return fmt.Sprintf("%s field %q: %s", e.Field, e.Value, e.Reason)
}

// Term is one comma-separated element of a field, e.g. "10-50/10".
type Term struct {
	Raw     string
	Start   int  // first value (names resolved)
	End     int  // last value, inclusive
	Step    int  // 1 when no step was given
	Star    bool // the base was "*"
	HasStep bool
	IsRange bool // "*" or "a-b"
}

// Field is a parsed cron field.
type Field struct {
	Kind  FieldKind
	Raw   string
	Terms []Term
	Bits  uint64 // bit v set when value v matches; day-of-week 7 is folded into 0
	// Star reports whether the field text starts with "*". Vixie cron uses
	// this (not the resulting bit set) to decide between AND and OR when
	// combining day-of-month and day-of-week.
	Star bool
}

// Has reports whether value v is selected by the field.
func (f Field) Has(v int) bool { return v >= 0 && v < 64 && f.Bits&(1<<uint(v)) != 0 }

// Values returns the selected values in ascending order.
func (f Field) Values() []int {
	var out []int
	for b := f.Bits; b != 0; b &= b - 1 {
		out = append(out, bits.TrailingZeros64(b))
	}
	return out
}

// Full reports whether every possible value of the field is selected.
func (f Field) Full() bool {
	lo, hi := f.Kind.Min(), f.Kind.Max()
	if f.Kind == DayOfWeek {
		hi = 6
	}
	return f.Bits == rangeBits(lo, hi, 1)
}

// Schedule is a parsed cron expression.
type Schedule struct {
	Raw    string   // expression as written
	Macro  string   // "@daily" etc., empty for five-field expressions
	Fields [5]Field // zero for @reboot

	never bool // no calendar date matches; computed by Parse
}

// Reboot reports whether the schedule is @reboot (no calendar times).
func (s *Schedule) Reboot() bool { return s.Macro == "@reboot" }

// Field returns the parsed field of the given kind.
func (s *Schedule) Field(k FieldKind) Field { return s.Fields[k] }

// String returns the canonical five-field form (or the macro).
func (s *Schedule) String() string {
	if s.Reboot() {
		return s.Macro
	}
	parts := make([]string, 5)
	for i, f := range s.Fields {
		parts[i] = f.Raw
	}
	return strings.Join(parts, " ")
}

// Parse parses a five-field expression or an @-macro.
func Parse(expr string) (*Schedule, error) {
	raw := strings.TrimSpace(expr)
	if raw == "" {
		return nil, &ParseError{Field: NoField, Value: expr, Reason: "empty expression"}
	}
	if strings.HasPrefix(raw, "@") {
		name := strings.ToLower(raw)
		five, ok := Macros[name]
		if !ok {
			return nil, &ParseError{Field: NoField, Value: raw,
				Reason: fmt.Sprintf("unknown macro %q", raw),
				Hint:   "use one of @reboot, @yearly, @annually, @monthly, @weekly, @daily, @midnight, @hourly"}
		}
		if five == "" {
			return &Schedule{Raw: raw, Macro: name}, nil
		}
		s, err := Parse(five)
		if err != nil {
			panic(err) // built-in macro definitions are valid
		}
		s.Raw, s.Macro = raw, name
		return s, nil
	}

	parts := strings.Fields(raw)
	if len(parts) != 5 {
		pe := &ParseError{Field: NoField, Value: raw,
			Reason: fmt.Sprintf("expected 5 fields (minute hour day-of-month month day-of-week), got %d", len(parts))}
		if len(parts) == 6 || len(parts) == 7 {
			pe.Reason += "; seconds and year fields are a Quartz/AWS extension, this is not supported by standard cron"
		}
		return nil, pe
	}
	s := &Schedule{Raw: raw}
	for i, p := range parts {
		f, err := parseField(FieldKind(i), p)
		if err != nil {
			return nil, err
		}
		s.Fields[i] = f
	}
	s.never = !s.possible()
	return s, nil
}

func parseField(kind FieldKind, raw string) (Field, error) {
	f := Field{Kind: kind, Raw: raw, Star: strings.HasPrefix(raw, "*")}
	fail := func(reason, hint string) (Field, error) {
		return Field{}, &ParseError{Field: kind, Value: raw, Reason: reason, Hint: hint}
	}
	if strings.ContainsAny(raw, "#?") {
		return fail("'#' and '?' are Quartz/AWS extensions; this is not supported by standard cron", "")
	}
	for _, part := range strings.Split(raw, ",") {
		if part == "" {
			return fail("empty element in list", "remove the extra comma")
		}
		t, reason, hint := parseTerm(kind, part)
		if reason != "" {
			return fail(reason, hint)
		}
		f.Terms = append(f.Terms, t)
		f.Bits |= termBits(kind, t)
	}
	return f, nil
}

func parseTerm(kind FieldKind, part string) (t Term, reason, hint string) {
	t = Term{Raw: part, Step: 1}
	base := part
	if i := strings.IndexByte(part, '/'); i >= 0 {
		base = part[:i]
		stepStr := part[i+1:]
		if strings.Contains(stepStr, "/") {
			return t, fmt.Sprintf("multiple '/' in %q", part), ""
		}
		n, ok := atoi(stepStr)
		if !ok || n <= 0 {
			return t, fmt.Sprintf("step %q must be a positive integer", stepStr), ""
		}
		span := kind.Max() - kind.Min() + 1
		if n > span {
			return t, fmt.Sprintf("step %d is larger than the field's range (%d values)", n, span), ""
		}
		t.Step, t.HasStep = n, true
	}

	switch {
	case base == "*":
		t.Star, t.IsRange = true, true
		t.Start, t.End = kind.Min(), kind.Max()
		if kind == DayOfWeek {
			t.End = 6
		}
	case strings.Contains(base, "-"):
		a, b, _ := strings.Cut(base, "-")
		lo, r, h := parseValue(kind, a)
		if r != "" {
			return t, r, h
		}
		hi, r, h := parseValue(kind, b)
		if r != "" {
			return t, r, h
		}
		if lo > hi {
			return t, fmt.Sprintf("reversed range %s; cron does not wrap ranges", base),
				fmt.Sprintf("split it into two ranges, e.g. %d-%d,%d-%d", lo, kind.Max(), kind.Min(), hi)
		}
		t.Start, t.End, t.IsRange = lo, hi, true
	default:
		v, r, h := parseValue(kind, base)
		if r != "" {
			return t, r, h
		}
		if t.HasStep {
			return t, fmt.Sprintf("a step on a single value (%s) is not supported by Vixie cron", part),
				fmt.Sprintf("use a range: %d-%d/%d", v, kind.Max(), t.Step)
		}
		t.Start, t.End = v, v
	}
	return t, "", ""
}

func parseValue(kind FieldKind, s string) (int, string, string) {
	if s == "" {
		return 0, "missing value", ""
	}
	if n, ok := atoi(s); ok {
		if n < kind.Min() || n > kind.Max() {
			return 0, fmt.Sprintf("value %d out of range %d-%d", n, kind.Min(), kind.Max()), ""
		}
		return n, "", ""
	}
	if isDigits(s) {
		return 0, fmt.Sprintf("value %s out of range %d-%d", s, kind.Min(), kind.Max()), ""
	}
	up := strings.ToUpper(s)
	if v, ok := specs[kind].names[up]; ok {
		return v, "", ""
	}
	if isQuartzToken(up) {
		hint := ""
		if kind == DayOfMonth && strings.HasPrefix(up, "L") {
			hint = `for the last day of the month use 28-31 and test it in the command: [ "$(date -d tomorrow +\%d)" = 01 ] && cmd`
		}
		return 0, fmt.Sprintf("%q is a Quartz/AWS extension (L/W); this is not supported by standard cron", s), hint
	}
	switch kind {
	case Month:
		return 0, fmt.Sprintf("unknown month name %q", s), "use JAN-DEC or 1-12"
	case DayOfWeek:
		return 0, fmt.Sprintf("unknown day name %q", s), "use SUN-SAT or 0-7"
	}
	return 0, fmt.Sprintf("%q is not a number", s), ""
}

// isQuartzToken recognises L, LW, 15W, 5L, L-3 and similar.
func isQuartzToken(s string) bool {
	if s == "L" || s == "LW" || strings.HasPrefix(s, "L-") {
		return true
	}
	if len(s) >= 2 && (s[len(s)-1] == 'W' || s[len(s)-1] == 'L') && isDigits(s[:len(s)-1]) {
		return true
	}
	return false
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func atoi(s string) (int, bool) {
	if !isDigits(s) || len(s) > 6 {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func rangeBits(lo, hi, step int) uint64 {
	var b uint64
	for v := lo; v <= hi; v += step {
		b |= 1 << uint(v)
	}
	return b
}

func termBits(kind FieldKind, t Term) uint64 {
	b := rangeBits(t.Start, t.End, t.Step)
	if kind == DayOfWeek && b&(1<<7) != 0 {
		b = b&^(1<<7) | 1
	}
	return b
}
