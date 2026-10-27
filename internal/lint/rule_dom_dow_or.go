package lint

import "github.com/your-username/cronwhy/internal/cronexpr"

func init() {
	register(domDowOr{meta{"dom-dow-or", Warning,
		"Day-of-month and day-of-week both restricted: cron uses OR, not AND"}})
}

type domDowOr struct{ meta }

func (r domDowOr) Check(ctx *Context) []Finding {
	var out []Finding
	for _, e := range ctx.Valid() {
		if e.Schedule.Reboot() {
			continue
		}
		dom, dow := e.Schedule.Fields[cronexpr.DayOfMonth], e.Schedule.Fields[cronexpr.DayOfWeek]
		if dom.Star || dow.Star {
			continue
		}
		out = append(out, r.finding(e.Line,
			`to require both, set day-of-week to * and test the weekday in the command, e.g. [ "$(date +\%u)" = 1 ] && cmd`,
			"day-of-month %q and day-of-week %q are both restricted, so cron runs the job when EITHER matches "+
				"(on day-of-month %s and also on every day-of-week %s), not only when both match",
			dom.Raw, dow.Raw, dom.Raw, dow.Raw))
	}
	return out
}
