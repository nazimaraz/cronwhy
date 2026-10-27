// Command cronwhy explains cron expressions and lints crontab files.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Exit codes.
const (
	exitOK       = 0
	exitWarnings = 1
	exitErrors   = 2
	exitUsage    = 3
)

var version = "dev"

const usage = `cronwhy explains cron expressions and lints crontab files.

Usage:
  cronwhy explain "<expr>" [-n 10] [--from "2026-10-27 20:00"] [--tz Europe/Istanbul]
  cronwhy next "<expr>" [-n 10] [--from ...] [--tz ...]
  cronwhy lint [file... | -] [--system] [--format text|json] [--strict] [--tz ...] [--window 7d]
  cronwhy version

Run "cronwhy <command> -h" for command flags.
`

type env struct {
	stdin          io.Reader
	stdout, stderr io.Writer
	now            func() time.Time
	getenv         func(string) string
	isTTY          bool
	crontabL       func() ([]byte, error)
}

func main() {
	e := &env{
		stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr,
		now: time.Now, getenv: os.Getenv, isTTY: isTerminal(os.Stdout),
		crontabL: readUserCrontab,
	}
	os.Exit(run(os.Args[1:], e))
}

func run(args []string, e *env) int {
	if len(args) == 0 {
		fmt.Fprint(e.stderr, usage)
		return exitUsage
	}
	switch args[0] {
	case "explain":
		return cmdExplain(args[1:], e, false)
	case "next":
		return cmdExplain(args[1:], e, true)
	case "lint":
		return cmdLint(args[1:], e)
	case "version", "--version":
		fmt.Fprintln(e.stdout, "cronwhy", version)
		return exitOK
	case "help", "-h", "--help":
		fmt.Fprint(e.stdout, usage)
		return exitOK
	}
	fmt.Fprintf(e.stderr, "cronwhy: unknown command %q\n\n%s", args[0], usage)
	return exitUsage
}

// parseInterspersed parses flags that may appear before or after
// positional arguments and returns the positionals.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		if rest[0] == "--" && len(fs.Args()) < len(args) && args[len(args)-len(rest)] == "--" {
			return append(pos, rest[1:]...), nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

func newFlagSet(name string, e *env, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	fs.Usage = func() {
		fmt.Fprintf(e.stderr, "Usage: %s\n\nFlags:\n", synopsis)
		fs.PrintDefaults()
	}
	return fs
}

func flagExit(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}
	return exitUsage
}

func loadLocation(name string) (*time.Location, error) {
	if name == "" {
		return time.Local, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("unknown timezone %q", name)
	}
	return loc, nil
}

var timeLayouts = []string{"2006-01-02 15:04", "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"}

func parseFrom(s string, loc *time.Location, now time.Time) (time.Time, error) {
	if s == "" {
		return now.In(loc), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(loc), nil
	}
	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, strings.TrimSpace(s), loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse --from %q; use \"YYYY-MM-DD HH:MM\"", s)
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
