package lint

func init() {
	register(impossibleDate{meta{"impossible-date", Error,
		"Expressions that can never fire, such as 0 0 31 2 *"}})
}

type impossibleDate struct{ meta }

func (r impossibleDate) Check(ctx *Context) []Finding {
	var out []Finding
	for _, e := range ctx.Valid() {
		if e.Schedule.Possible() {
			continue
		}
		out = append(out, r.finding(e.Line,
			"pick a day that exists in the selected months (February has at most 29 days; April, June, September and November have 30)",
			"schedule %q can never fire: no selected month has the selected day-of-month", e.ScheduleText))
	}
	return out
}
