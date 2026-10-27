package lint

import (
	"fmt"
	"strings"

	"github.com/nazimaraz/cronwhy/internal/cronexpr"
)

func init() {
	register(unevenStep{meta{"uneven-step", Warning,
		"Steps such as */7 that do not divide the field evenly and leave a short gap at the wrap-around"}})
}

type unevenStep struct{ meta }

var stepUnits = [5]string{"minute", "hour", "day", "month", "day"}

func (r unevenStep) Check(ctx *Context) []Finding {
	var out []Finding
	for _, e := range ctx.Valid() {
		if e.Schedule.Reboot() {
			continue
		}
		for _, f := range e.Schedule.Fields {
			for _, t := range f.Terms {
				if msg, fix, ok := unevenTerm(f.Kind, t); ok {
					out = append(out, r.finding(e.Line, fix, "%s", msg))
				}
			}
		}
	}
	return out
}

func unevenTerm(kind cronexpr.FieldKind, t cronexpr.Term) (msg, fix string, ok bool) {
	lo, hi := kind.Min(), kind.Max()
	if kind == cronexpr.DayOfWeek {
		hi = 6
	}
	// Only steps over the whole field wrap around into the next cycle.
	if t.Step <= 1 || t.Start != lo || (t.End != hi && !(kind == cronexpr.DayOfWeek && t.End == 7)) {
		return "", "", false
	}
	cycle := hi - lo + 1
	if cycle%t.Step == 0 {
		return "", "", false
	}
	var points []int
	for v := lo; v <= hi; v += t.Step {
		points = append(points, v)
	}
	last := points[len(points)-1]
	gap := lo + cycle - last
	unit := stepUnits[kind]
	msg = fmt.Sprintf("%s in the %s field fires at %s; after %d it wraps to %d, a gap of %d %s(s) instead of %d",
		t.Raw, kind, joinInts(points, ","), last, lo, gap, unit, t.Step)
	if kind == cronexpr.DayOfMonth {
		msg += " (and months have 28-31 days)"
	}
	var divs []string
	for d := 2; d < cycle; d++ {
		if cycle%d == 0 {
			divs = append(divs, fmt.Sprintf("*/%d", d))
		}
	}
	if len(divs) > 0 {
		fix = fmt.Sprintf("use a step that divides %d evenly (%s) or list the exact values", cycle, strings.Join(divs, ", "))
	} else {
		fix = "list the exact values you want instead of a step"
	}
	return msg, fix, true
}
