package lint

import (
	"fmt"
	"sort"
	"time"

	"github.com/nazimaraz/cronwhy/internal/crontab"
)

func init() {
	register(pileUp{meta{"pile-up", Warning,
		"Many jobs starting in the same minute (simulated over --window)"}})
}

type pileUp struct{ meta }

type pileGroup struct {
	label string // wall clock "15:04"
	max   int
	times int
	lines map[int]bool
}

func (r pileUp) Check(ctx *Context) []Finding {
	threshold := ctx.PileUpThreshold
	if threshold <= 0 {
		threshold = 5
	}
	var jobs []*crontab.Entry
	var locs []*time.Location
	for _, e := range ctx.Valid() {
		if !e.Schedule.Reboot() {
			jobs = append(jobs, e)
			locs = append(locs, ctx.Location(e))
		}
	}
	if len(jobs) < threshold || ctx.Window <= 0 {
		return nil
	}

	groups := map[string]*pileGroup{}
	start := ctx.From.Truncate(time.Minute)
	end := ctx.From.Add(ctx.Window)
	var hit []int
	for t := start; t.Before(end); t = t.Add(time.Minute) {
		hit = hit[:0]
		for i, e := range jobs {
			if e.Schedule.Match(t.In(locs[i])) {
				hit = append(hit, i)
			}
		}
		if len(hit) < threshold {
			continue
		}
		// Label with the wall clock of the first job in the group.
		label := t.In(locs[hit[0]]).Format("15:04")
		g := groups[label]
		if g == nil {
			g = &pileGroup{label: label, lines: map[int]bool{}}
			groups[label] = g
		}
		g.times++
		if len(hit) > g.max {
			g.max = len(hit)
		}
		for _, i := range hit {
			g.lines[jobs[i].Line] = true
		}
	}

	var out []Finding
	for _, g := range groups {
		lines := make([]int, 0, len(g.lines))
		for l := range g.lines {
			lines = append(lines, l)
		}
		sort.Ints(lines)
		msg := fmt.Sprintf("%d jobs start in the same minute at %s (lines %s); this happens %d time(s) in the %s window",
			g.max, g.label, joinInts(lines, ", "), g.times, formatWindow(ctx.Window))
		if g.label[3:] == "00" {
			msg += "; the top of the hour (especially midnight) is the busiest minute on most systems"
		}
		out = append(out, r.finding(lines[0],
			`stagger the start minutes (e.g. 3, 17, 34, 48) or add a random delay such as: sleep $((RANDOM \% 300)); cmd`,
			"%s", msg))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Message < out[j].Message
	})
	return out
}

func formatWindow(d time.Duration) string {
	if d%(24*time.Hour) == 0 {
		return fmt.Sprintf("%dd", d/(24*time.Hour))
	}
	return d.String()
}
