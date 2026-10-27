package lint

import "github.com/your-username/cronwhy/internal/cronexpr"

func init() {
	register(everyMinute{meta{"every-minute", Info, "Jobs that run every minute"}})
}

type everyMinute struct{ meta }

func (r everyMinute) Check(ctx *Context) []Finding {
	var out []Finding
	for _, e := range ctx.Valid() {
		s := e.Schedule
		if s.Reboot() || !s.Fields[cronexpr.Minute].Full() || !s.Fields[cronexpr.Hour].Full() {
			continue
		}
		out = append(out, r.finding(e.Line,
			"if this is intended, make sure the job finishes well within a minute (or use flock); otherwise pick a specific minute or a step like */5",
			"job runs every minute (up to 1440 times a day)"))
	}
	return out
}
