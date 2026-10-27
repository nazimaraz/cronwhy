package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nazimaraz/cronwhy/internal/cronexpr"
	"github.com/nazimaraz/cronwhy/internal/explain"
)

func cmdExplain(args []string, e *env, timesOnly bool) int {
	name, synopsis := "explain", `cronwhy explain "<expr>" [flags]`
	if timesOnly {
		name, synopsis = "next", `cronwhy next "<expr>" [flags]`
	}
	fs := newFlagSet(name, e, synopsis)
	n := fs.Int("n", 10, "number of upcoming runs to show")
	from := fs.String("from", "", `start time "YYYY-MM-DD HH:MM" (default now)`)
	tz := fs.String("tz", "", "IANA timezone, e.g. Europe/Istanbul (default local)")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return flagExit(err)
	}
	if len(pos) == 0 {
		fs.Usage()
		return exitUsage
	}
	if *n < 0 || *n > 1000 {
		fmt.Fprintln(e.stderr, "cronwhy: -n must be between 0 and 1000")
		return exitUsage
	}
	loc, err := loadLocation(*tz)
	if err != nil {
		fmt.Fprintln(e.stderr, "cronwhy:", err)
		return exitUsage
	}
	start, err := parseFrom(*from, loc, e.now())
	if err != nil {
		fmt.Fprintln(e.stderr, "cronwhy:", err)
		return exitUsage
	}

	// Allow the expression unquoted: cronwhy explain 0 0 * * *
	expr := strings.Join(pos, " ")
	s, err := cronexpr.Parse(expr)
	if err != nil {
		fmt.Fprintf(e.stderr, "cronwhy: invalid expression %q\n  %v\n", expr, err)
		var pe *cronexpr.ParseError
		if errors.As(err, &pe) && pe.Hint != "" {
			fmt.Fprintf(e.stderr, "  hint: %s\n", pe.Hint)
		}
		return exitErrors
	}

	times, nextErr := s.NextN(start, *n)
	if timesOnly {
		for _, t := range times {
			fmt.Fprintln(e.stdout, t.Format(time.RFC3339))
		}
		if nextErr != nil {
			fmt.Fprintln(e.stderr, "cronwhy:", nextErr)
			return exitErrors
		}
		return exitOK
	}

	w := e.stdout
	fmt.Fprintf(w, "%s\n%s\n", s.Raw, explain.Describe(s))
	if s.Macro != "" && !s.Reboot() {
		fmt.Fprintf(w, "(%s is %s)\n", s.Macro, s.String())
	}
	if !s.Reboot() {
		fmt.Fprintln(w)
		for _, f := range s.Fields {
			fmt.Fprintf(w, "  %-13s %-10s %s\n", f.Kind, f.Raw, explain.CompactValues(f))
		}
		dom, dow := s.Fields[cronexpr.DayOfMonth], s.Fields[cronexpr.DayOfWeek]
		if !dom.Star && !dow.Star {
			fmt.Fprintln(w, "\nNote: day-of-month and day-of-week are both restricted, so cron runs the job")
			fmt.Fprintln(w, "when EITHER one matches (Vixie OR rule), not only when both match.")
		}
	}
	if nextErr != nil {
		fmt.Fprintf(w, "\n%s\n", nextErr)
		if errors.Is(nextErr, cronexpr.ErrReboot) {
			return exitOK
		}
		return exitErrors
	}
	if len(times) == 0 {
		return exitOK
	}
	fmt.Fprintf(w, "\nNext %d runs (%s, from %s):\n", len(times), locName(loc), start.Format("Mon 2006-01-02 15:04"))
	for _, t := range times {
		fmt.Fprintf(w, "  %s  %s\n", t.Format("Mon 2006-01-02 15:04 MST"), explain.Relative(t.Sub(start)))
	}
	return exitOK
}

func locName(loc *time.Location) string {
	if loc == time.Local {
		return "local time"
	}
	return loc.String()
}
