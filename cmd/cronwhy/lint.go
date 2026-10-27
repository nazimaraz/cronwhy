package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/your-username/cronwhy/internal/crontab"
	"github.com/your-username/cronwhy/internal/lint"
	"github.com/your-username/cronwhy/internal/report"
)

func cmdLint(args []string, e *env) int {
	fs := newFlagSet("lint", e, "cronwhy lint [file | -] [flags]")
	system := fs.Bool("system", false, "parse /etc/crontab and /etc/cron.d format (sixth field is the user)")
	format := fs.String("format", "text", "output format: text or json")
	strict := fs.Bool("strict", false, "exit 1 when only warnings or info findings are present")
	tz := fs.String("tz", "", "timezone of jobs without CRON_TZ (default: system local time)")
	window := fs.String("window", "7d", "pile-up simulation window, e.g. 7d, 36h")
	pileUp := fs.Int("pile-up", 5, "number of jobs starting in the same minute that counts as a pile-up")
	from := fs.String("from", "", `start of the simulation "YYYY-MM-DD HH:MM" (default now)`)
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return flagExit(err)
	}
	if len(pos) > 1 {
		fmt.Fprintln(e.stderr, "cronwhy: lint takes at most one file")
		return exitUsage
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(e.stderr, "cronwhy: unknown --format %q (want text or json)\n", *format)
		return exitUsage
	}
	if *pileUp < 2 {
		fmt.Fprintln(e.stderr, "cronwhy: --pile-up must be at least 2")
		return exitUsage
	}
	win, err := parseWindow(*window)
	if err != nil {
		fmt.Fprintln(e.stderr, "cronwhy:", err)
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

	name, data, err := readInput(pos, e)
	if err != nil {
		fmt.Fprintln(e.stderr, "cronwhy:", err)
		return exitUsage
	}
	file, err := crontab.Parse(bytes.NewReader(data), name, crontab.Options{System: *system})
	if err != nil {
		fmt.Fprintln(e.stderr, "cronwhy:", err)
		return exitUsage
	}

	cfg := lint.Config{Location: loc, From: start, Window: win, PileUpThreshold: *pileUp}
	findings := lint.Run(file, cfg)
	if *format == "json" {
		if err := report.JSON(e.stdout, file, findings); err != nil {
			fmt.Fprintln(e.stderr, "cronwhy:", err)
			return exitUsage
		}
	} else {
		report.Text(e.stdout, file, findings, report.UseColor(e.isTTY, e.getenv("NO_COLOR")))
	}

	sum := report.Summarize(file, findings)
	switch {
	case sum.Errors > 0:
		return exitErrors
	case *strict && len(findings) > 0:
		return exitWarnings
	}
	return exitOK
}

func readInput(pos []string, e *env) (name string, data []byte, err error) {
	switch {
	case len(pos) == 0:
		data, err = e.crontabL()
		return "crontab -l", data, err
	case pos[0] == "-":
		data, err = io.ReadAll(e.stdin)
		return "<stdin>", data, err
	}
	data, err = os.ReadFile(pos[0])
	return pos[0], data, err
}

func readUserCrontab() ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("crontab", "-l")
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("crontab -l: %s", msg)
		}
		return nil, fmt.Errorf("crontab -l: %w", err)
	}
	return out, nil
}

// parseWindow accepts Go durations plus a "d" (day) suffix.
func parseWindow(s string) (time.Duration, error) {
	var d time.Duration
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil {
			return 0, fmt.Errorf("invalid --window %q", s)
		}
		d = time.Duration(n) * 24 * time.Hour
	} else {
		var err error
		if d, err = time.ParseDuration(s); err != nil {
			return 0, fmt.Errorf("invalid --window %q", s)
		}
	}
	if d < time.Minute || d > 366*24*time.Hour {
		return 0, fmt.Errorf("--window must be between 1m and 366d")
	}
	return d, nil
}
