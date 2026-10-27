package lint

import "strings"

func init() {
	register(unescapedPercent{meta{"unescaped-percent", Error,
		"A % in the command that is not escaped as \\%; cron turns it into a newline"}})
}

type unescapedPercent struct{ meta }

// escapePercents escapes every unescaped '%' in cmd.
func escapePercents(cmd string) (string, int) {
	var b strings.Builder
	n := 0
	for i := 0; i < len(cmd); i++ {
		if cmd[i] == '%' && (i == 0 || cmd[i-1] != '\\') {
			b.WriteString(`\%`)
			n++
			continue
		}
		b.WriteByte(cmd[i])
	}
	return b.String(), n
}

func (r unescapedPercent) Check(ctx *Context) []Finding {
	var out []Finding
	for _, e := range ctx.File.Entries {
		if e.Command == "" {
			continue
		}
		fixed, n := escapePercents(e.Command)
		if n == 0 {
			continue
		}
		out = append(out, r.finding(e.Line, "escape it: "+fixed,
			"%d unescaped %% in the command; cron turns the first one into a newline and sends the rest to stdin, so the shell only runs: %s",
			n, strings.TrimSpace(e.CommandLine())))
	}
	return out
}
