package lint

import (
	"fmt"
	"strings"
	"time"

	"github.com/your-username/cronwhy/internal/cronexpr"
	"github.com/your-username/cronwhy/internal/crontab"
)

func init() {
	register(dstRisk{meta{"dst-risk", Warning,
		"Jobs scheduled inside a daylight saving time transition window"}})
}

type dstRisk struct{ meta }

// transition is a wall-clock window that is skipped (spring) or repeated
// (autumn) on a given local date.
type transition struct {
	spring   bool
	date     time.Time // local midnight-ish date of the window
	startMin int       // minute of day where the window begins
	length   int       // minutes
	from, to string    // e.g. "02:00" -> "03:00"
}

// transitions lists the offset changes of loc within a year after from.
func transitions(loc *time.Location, from time.Time) []transition {
	var out []transition
	limit := from.AddDate(1, 0, 0)
	t := from.In(loc)
	for i := 0; i < 8; i++ {
		_, end := t.ZoneBounds()
		if end.IsZero() || end.After(limit) {
			break
		}
		_, before := t.Zone()
		_, after := end.In(loc).Zone()
		delta := (after - before) / 60
		if delta != 0 {
			tr := transition{spring: delta > 0}
			var wall time.Time
			if tr.spring {
				wall = end.In(time.FixedZone("", before))
				tr.length = delta
			} else {
				wall = end.In(loc)
				tr.length = -delta
			}
			tr.date = wall
			tr.startMin = wall.Hour()*60 + wall.Minute()
			oldWall := end.In(time.FixedZone("", before))
			tr.from = oldWall.Format("15:04")
			tr.to = end.In(loc).Format("15:04")
			out = append(out, tr)
		}
		t = end.In(loc)
	}
	return out
}

func (r dstRisk) Check(ctx *Context) []Finding {
	var out []Finding
	cache := map[*time.Location][]transition{}
	for _, e := range ctx.Valid() {
		if e.Schedule.Reboot() {
			continue
		}
		loc := ctx.Location(e)
		trs, ok := cache[loc]
		if !ok {
			trs = transitions(loc, ctx.From)
			cache[loc] = trs
		}
		var parts []string
		seen := map[bool]bool{}
		for _, tr := range trs {
			if seen[tr.spring] {
				continue
			}
			hits := windowHits(e, tr)
			if len(hits) == 0 {
				continue
			}
			seen[tr.spring] = true
			if tr.spring {
				parts = append(parts, fmt.Sprintf("when DST starts clocks jump from %s to %s, so the %s run may be skipped",
					tr.from, tr.to, summarizeTimes(hits)))
			} else {
				parts = append(parts, fmt.Sprintf("when DST ends clocks go back from %s to %s, so the %s run may happen twice",
					tr.from, tr.to, summarizeTimes(hits)))
			}
		}
		if len(parts) == 0 {
			continue
		}
		out = append(out, r.finding(e.Line,
			"schedule the job outside the transition window (e.g. 03:30 or later) or set CRON_TZ=UTC for it",
			"in %s %s (behaviour depends on the cron implementation)", loc, strings.Join(parts, "; ")))
	}
	return out
}

func windowHits(e *crontab.Entry, tr transition) []string {
	s := e.Schedule
	if !s.DayMatches(tr.date) {
		return nil
	}
	var hits []string
	for m := tr.startMin; m < tr.startMin+tr.length && m < 24*60; m++ {
		if s.Fields[cronexpr.Hour].Has(m/60) && s.Fields[cronexpr.Minute].Has(m%60) {
			hits = append(hits, fmt.Sprintf("%02d:%02d", m/60, m%60))
		}
	}
	return hits
}

func summarizeTimes(hits []string) string {
	if len(hits) <= 3 {
		return strings.Join(hits, ", ")
	}
	return fmt.Sprintf("%s ... %s (%d runs)", hits[0], hits[len(hits)-1], len(hits))
}
