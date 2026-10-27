// Package report renders lint findings as colored text or JSON.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/your-username/cronwhy/internal/crontab"
	"github.com/your-username/cronwhy/internal/lint"
)

// SchemaVersion is the version of the JSON output schema.
const SchemaVersion = 1

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
)

// UseColor decides whether to color output: only on a terminal and only
// when NO_COLOR is unset (https://no-color.org).
func UseColor(isTTY bool, noColorEnv string) bool {
	return isTTY && noColorEnv == ""
}

type painter bool

func (p painter) paint(code, s string) string {
	if !p {
		return s
	}
	return code + s + ansiReset
}

func severityColor(s lint.Severity) string {
	switch s {
	case lint.Error:
		return ansiRed
	case lint.Warning:
		return ansiYellow
	}
	return ansiCyan
}

// Summary counts findings by severity.
type Summary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
	Jobs     int `json:"jobs"`
}

// Summarize counts findings by severity.
func Summarize(f *crontab.File, findings []lint.Finding) Summary {
	s := Summary{Jobs: len(f.Entries)}
	for _, fd := range findings {
		switch fd.Severity {
		case lint.Error:
			s.Errors++
		case lint.Warning:
			s.Warnings++
		default:
			s.Info++
		}
	}
	return s
}

func source(f *crontab.File, line int) string {
	if line < 1 || line > len(f.Lines) {
		return ""
	}
	return strings.TrimSpace(f.Lines[line-1])
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// Text writes findings in a golangci-lint like format:
//
//	file:line: severity: message (rule-id)
//	    source line
//	    fix: suggestion
func Text(w io.Writer, f *crontab.File, findings []lint.Finding, color bool) {
	p := painter(color)
	for _, fd := range findings {
		fmt.Fprintf(w, "%s %s %s %s\n",
			p.paint(ansiBold, fmt.Sprintf("%s:%d:", f.Name, fd.Line)),
			p.paint(severityColor(fd.Severity), fd.Severity.String()+":"),
			fd.Message,
			p.paint(ansiDim, "("+fd.Rule+")"))
		if src := source(f, fd.Line); src != "" {
			fmt.Fprintf(w, "    %s\n", src)
		}
		if fd.Fix != "" {
			fmt.Fprintf(w, "    %s %s\n", p.paint(ansiGreen, "fix:"), fd.Fix)
		}
	}
	s := Summarize(f, findings)
	if len(findings) == 0 {
		fmt.Fprintf(w, "%s %s: no problems found (%s checked)\n", p.paint(ansiGreen, "ok"), f.Name, plural(s.Jobs, "job"))
		return
	}
	fmt.Fprintf(w, "\n%s: %s, %s, %d info (%s checked)\n", f.Name,
		p.paint(ansiRed, plural(s.Errors, "error")), p.paint(ansiYellow, plural(s.Warnings, "warning")), s.Info, plural(s.Jobs, "job"))
}

// JSONFinding is one element of the "findings" array.
type JSONFinding struct {
	Rule     string        `json:"rule"`
	Severity lint.Severity `json:"severity"`
	File     string        `json:"file"`
	Line     int           `json:"line"`
	Message  string        `json:"message"`
	Fix      string        `json:"fix"`
	Source   string        `json:"source"`
}

// JSONReport is the top-level JSON document.
type JSONReport struct {
	Version  int           `json:"version"`
	File     string        `json:"file"`
	Summary  Summary       `json:"summary"`
	Findings []JSONFinding `json:"findings"`
}

// JSON writes findings as an indented JSON document.
func JSON(w io.Writer, f *crontab.File, findings []lint.Finding) error {
	rep := JSONReport{Version: SchemaVersion, File: f.Name, Summary: Summarize(f, findings), Findings: []JSONFinding{}}
	for _, fd := range findings {
		rep.Findings = append(rep.Findings, JSONFinding{
			Rule: fd.Rule, Severity: fd.Severity, File: f.Name, Line: fd.Line,
			Message: fd.Message, Fix: fd.Fix, Source: source(f, fd.Line),
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(rep)
}
