package lint

import (
	"errors"

	"github.com/nazimaraz/cronwhy/internal/cronexpr"
)

func init() {
	register(invalidField{meta{"invalid-field", Error,
		"Out-of-range values, reversed ranges, bad steps, unknown names or malformed lines"}})
}

type invalidField struct{ meta }

func (r invalidField) Check(ctx *Context) []Finding {
	var out []Finding
	for _, e := range ctx.File.Entries {
		if e.Err == nil {
			continue
		}
		fix := "a job line is: minute hour day-of-month month day-of-week [user] command"
		var pe *cronexpr.ParseError
		if errors.As(e.Err, &pe) {
			switch {
			case pe.Hint != "":
				fix = pe.Hint
			case pe.Field != cronexpr.NoField:
				fix = "valid ranges: minute 0-59, hour 0-23, day-of-month 1-31, month 1-12 or JAN-DEC, day-of-week 0-7 or SUN-SAT"
			}
		}
		out = append(out, r.finding(e.Line, fix, "%v", e.Err))
	}
	return out
}
