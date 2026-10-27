package lint

import (
	"fmt"
	"strings"

	"github.com/nazimaraz/cronwhy/internal/crontab"
)

func init() {
	register(duplicateJob{meta{"duplicate-job", Warning,
		"The same schedule and command appearing more than once"}})
}

type duplicateJob struct{ meta }

// jobKey identifies a job by what it does, not how it is spelled:
// "0 0 * * *" and "@daily" are the same schedule.
func jobKey(e *crontab.Entry) string {
	var b strings.Builder
	s := e.Schedule
	if s.Reboot() {
		b.WriteString("@reboot")
	} else {
		for _, f := range s.Fields {
			fmt.Fprintf(&b, "%x/%t ", f.Bits, f.Star)
		}
	}
	b.WriteString("|" + e.User + "|" + strings.Join(strings.Fields(e.Command), " "))
	return b.String()
}

func (r duplicateJob) Check(ctx *Context) []Finding {
	var out []Finding
	first := map[string]int{}
	for _, e := range ctx.Valid() {
		k := jobKey(e)
		if line, ok := first[k]; ok {
			out = append(out, r.finding(e.Line, "remove one of the two lines",
				"same schedule and command as line %d; the job runs twice every time", line))
			continue
		}
		first[k] = e.Line
	}
	return out
}
