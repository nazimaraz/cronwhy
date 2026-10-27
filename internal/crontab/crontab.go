// Package crontab parses user and system crontab files the way Vixie cron
// reads them: environment assignments, comments, @-macros and five-field
// entries, optionally followed by a user name (system format).
package crontab

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/your-username/cronwhy/internal/cronexpr"
)

// EnvVar is a NAME=value line.
type EnvVar struct {
	Line  int
	Name  string
	Value string
}

// Entry is one job line. When Err is non-nil the line could not be
// parsed and Schedule may be nil; Command is still filled in when possible.
type Entry struct {
	Line         int
	Raw          string // the full source line
	ScheduleText string // "*/5 * * * *" or "@daily"
	Schedule     *cronexpr.Schedule
	User         string // system format only
	Command      string
	// Env is the environment in effect for this entry (assignments that
	// appear above it in the file). It is shared between entries that see
	// the same assignments and must not be modified.
	Env map[string]string
	Err error
}

// File is a parsed crontab.
type File struct {
	Name    string
	System  bool
	Lines   []string // source lines, Lines[0] is line 1
	Env     []EnvVar
	Entries []*Entry
}

// Options control parsing.
type Options struct {
	// System expects the /etc/crontab format with a user field.
	System bool
}

// SyntaxError is a line-level problem that is not a field error.
type SyntaxError struct{ Msg string }

func (e *SyntaxError) Error() string { return e.Msg }

var envRe = regexp.MustCompile(`^\s*("[^"]*"|'[^']*'|[^\s=]+)\s*=\s*(.*)$`)

// Parse reads a crontab. A malformed line never stops parsing; it is
// returned as an Entry with Err set. Only read errors are returned.
func Parse(r io.Reader, name string, opts Options) (*File, error) {
	f := &File{Name: name, System: opts.System}
	env := map[string]string{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		f.Lines = append(f.Lines, line)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if name, value, ok := parseEnv(trimmed); ok {
			f.Env = append(f.Env, EnvVar{Line: n, Name: name, Value: value})
			next := make(map[string]string, len(env)+1)
			for k, v := range env {
				next[k] = v
			}
			next[name] = value
			env = next
			continue
		}
		e := parseEntry(trimmed, opts.System)
		e.Line, e.Raw, e.Env = n, line, env
		f.Entries = append(f.Entries, e)
	}
	if err := sc.Err(); err != nil {
		return f, fmt.Errorf("%s: %w", name, err)
	}
	return f, nil
}

// parseEnv recognises NAME=value. Like Vixie's load_env, a line whose
// first token starts like a schedule is never an assignment.
func parseEnv(line string) (name, value string, ok bool) {
	if c := line[0]; c == '*' || c == '@' || (c >= '0' && c <= '9') {
		return "", "", false
	}
	m := envRe.FindStringSubmatch(line)
	if m == nil {
		return "", "", false
	}
	name, value = unquote(m[1]), strings.TrimSpace(m[2])
	if strings.ContainsAny(name, " \t") || name == "" {
		return "", "", false
	}
	return name, unquote(value), true
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// splitFields splits off the first n whitespace-separated tokens and
// returns them plus the untouched remainder.
func splitFields(s string, n int) ([]string, string) {
	var toks []string
	for len(toks) < n {
		s = strings.TrimLeft(s, " \t")
		if s == "" {
			break
		}
		i := strings.IndexAny(s, " \t")
		if i < 0 {
			toks = append(toks, s)
			s = ""
			break
		}
		toks = append(toks, s[:i])
		s = s[i:]
	}
	return toks, strings.TrimSpace(s)
}

func parseEntry(line string, system bool) *Entry {
	e := &Entry{}
	nsched := 5
	if strings.HasPrefix(line, "@") {
		nsched = 1
	}
	want := nsched
	if system {
		want++
	}
	toks, rest := splitFields(line, want)
	e.Command = rest
	if len(toks) < nsched {
		e.Err = &SyntaxError{Msg: fmt.Sprintf("incomplete entry: expected %d schedule fields, got %d", nsched, len(toks))}
		return e
	}
	e.ScheduleText = strings.Join(toks[:nsched], " ")
	if system {
		if len(toks) > nsched {
			e.User = toks[nsched]
		}
	}
	e.Schedule, e.Err = cronexpr.Parse(e.ScheduleText)
	if e.Err != nil {
		return e
	}
	switch {
	case system && e.User == "":
		e.Err = &SyntaxError{Msg: "missing user name (system crontab format: schedule, user, command)"}
	case e.Command == "":
		e.Err = &SyntaxError{Msg: "missing command"}
	}
	return e
}

// CommandLine returns the part of the command cron passes to the shell:
// everything before the first unescaped '%', with "\%" unescaped.
func (e *Entry) CommandLine() string {
	var b strings.Builder
	for i := 0; i < len(e.Command); i++ {
		c := e.Command[i]
		if c == '\\' && i+1 < len(e.Command) && e.Command[i+1] == '%' {
			b.WriteByte('%')
			i++
			continue
		}
		if c == '%' {
			break
		}
		b.WriteByte(c)
	}
	return b.String()
}
