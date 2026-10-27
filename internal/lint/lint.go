// Package lint checks parsed crontabs for common pitfalls. Each rule lives
// in its own rule_*.go file and registers itself in init.
package lint

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/your-username/cronwhy/internal/crontab"
)

// Severity of a finding.
type Severity int

// Severities, from least to most severe.
const (
	Info Severity = iota
	Warning
	Error
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	}
	return "info"
}

// MarshalText encodes the severity as its name in JSON.
func (s Severity) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

// Finding is a single problem reported by a rule.
type Finding struct {
	Rule     string
	Severity Severity
	Line     int
	Message  string
	Fix      string
}

// Config holds the settings shared by all rules.
type Config struct {
	// Location is the timezone of jobs without CRON_TZ (system default or --tz).
	Location *time.Location
	// From is the start of simulated windows (pile-up, dst-risk).
	From time.Time
	// Window is the length of the pile-up simulation.
	Window time.Duration
	// PileUpThreshold is the number of jobs per minute that counts as a pile-up.
	PileUpThreshold int
}

// DefaultConfig returns the defaults used by the CLI.
func DefaultConfig(now time.Time) Config {
	return Config{Location: time.Local, From: now, Window: 7 * 24 * time.Hour, PileUpThreshold: 5}
}

// Context is what rules inspect.
type Context struct {
	File *crontab.File
	Config
}

// Valid returns the entries whose schedule parsed without error.
func (c *Context) Valid() []*crontab.Entry {
	var out []*crontab.Entry
	for _, e := range c.File.Entries {
		if e.Err == nil && e.Schedule != nil {
			out = append(out, e)
		}
	}
	return out
}

// Location returns the effective timezone for e: CRON_TZ when set and
// valid, otherwise the configured default.
func (c *Context) Location(e *crontab.Entry) *time.Location {
	if tz := e.Env["CRON_TZ"]; tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	if c.Config.Location == nil {
		return time.Local
	}
	return c.Config.Location
}

// Rule is implemented by every lint rule.
type Rule interface {
	ID() string
	Severity() Severity
	Summary() string
	Check(ctx *Context) []Finding
}

// meta implements the descriptive half of Rule.
type meta struct {
	id       string
	severity Severity
	summary  string
}

func (m meta) ID() string         { return m.id }
func (m meta) Severity() Severity { return m.severity }
func (m meta) Summary() string    { return m.summary }

func (m meta) finding(line int, fix, format string, args ...any) Finding {
	return Finding{Rule: m.id, Severity: m.severity, Line: line, Message: fmt.Sprintf(format, args...), Fix: fix}
}

var registry = map[string]Rule{}

func register(r Rule) {
	if _, dup := registry[r.ID()]; dup {
		panic("lint: duplicate rule " + r.ID())
	}
	registry[r.ID()] = r
}

// Rules returns all registered rules sorted by ID.
func Rules() []Rule {
	out := make([]Rule, 0, len(registry))
	for _, r := range registry {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// Lookup returns the rule with the given ID.
func Lookup(id string) (Rule, bool) {
	r, ok := registry[id]
	return r, ok
}

// Run applies every rule to f and returns findings sorted by line,
// severity (most severe first) and rule ID.
func Run(f *crontab.File, cfg Config) []Finding {
	ctx := &Context{File: f, Config: cfg}
	var out []Finding
	for _, r := range Rules() {
		out = append(out, r.Check(ctx)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Severity != b.Severity {
			return a.Severity > b.Severity
		}
		return a.Rule < b.Rule
	})
	return out
}

func joinInts(vals []int, sep string) string {
	s := make([]string, len(vals))
	for i, v := range vals {
		s[i] = fmt.Sprint(v)
	}
	return strings.Join(s, sep)
}
