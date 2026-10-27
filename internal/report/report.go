// Package report renders lint findings as colored text or JSON.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/nazimaraz/cronwhy/internal/crontab"
	"github.com/nazimaraz/cronwhy/internal/lint"
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

// Result is the lint outcome for one file.
type Result struct {
	File     *crontab.File
	Findings []lint.Finding
}

// Summary counts findings by severity.
type Summary struct {
	Files    int `json:"files"`
	Jobs     int `json:"jobs"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

// Summarize counts files, jobs and findings by severity.
func Summarize(results []Result) Summary {
	var s Summary
	for _, r := range results {
		s.Files++
		s.Jobs += len(r.File.Entries)
		for _, fd := range r.Findings {
			switch fd.Severity {
			case lint.Error:
				s.Errors++
			case lint.Warning:
				s.Warnings++
			default:
				s.Info++
			}
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
func Text(w io.Writer, results []Result, color bool) {
	p := painter(color)
	total := 0
	for _, r := range results {
		total += len(r.Findings)
		for _, fd := range r.Findings {
			fmt.Fprintf(w, "%s %s %s %s\n",
				p.paint(ansiBold, fmt.Sprintf("%s:%d:", r.File.Name, fd.Line)),
				p.paint(severityColor(fd.Severity), fd.Severity.String()+":"),
				fd.Message,
				p.paint(ansiDim, "("+fd.Rule+")"))
			if src := source(r.File, fd.Line); src != "" {
				fmt.Fprintf(w, "    %s\n", src)
			}
			if fd.Fix != "" {
				fmt.Fprintf(w, "    %s %s\n", p.paint(ansiGreen, "fix:"), fd.Fix)
			}
		}
	}
	if total == 0 {
		for _, r := range results {
			fmt.Fprintf(w, "%s %s: no problems found (%s checked)\n",
				p.paint(ansiGreen, "ok"), r.File.Name, plural(len(r.File.Entries), "job"))
		}
		return
	}
	s := Summarize(results)
	label := plural(s.Files, "file")
	if len(results) == 1 {
		label = results[0].File.Name
	}
	fmt.Fprintf(w, "\n%s: %s, %s, %d info (%s checked)\n", label,
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
	Summary  Summary       `json:"summary"`
	Findings []JSONFinding `json:"findings"`
}

// JSON writes all findings as one indented JSON document.
func JSON(w io.Writer, results []Result) error {
	rep := JSONReport{Version: SchemaVersion, Summary: Summarize(results), Findings: []JSONFinding{}}
	for _, r := range results {
		for _, fd := range r.Findings {
			rep.Findings = append(rep.Findings, JSONFinding{
				Rule: fd.Rule, Severity: fd.Severity, File: r.File.Name, Line: fd.Line,
				Message: fd.Message, Fix: fd.Fix, Source: source(r.File, fd.Line),
			})
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(rep)
}
