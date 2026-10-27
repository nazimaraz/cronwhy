package lint

import (
	"fmt"
	"strings"
)

func init() {
	register(relativePath{meta{"relative-path", Warning,
		"Command is not an absolute path and PATH is not set in the crontab"}})
}

type relativePath struct{ meta }

// shellWords are builtins and keywords that do not need a PATH lookup.
var shellWords = map[string]bool{
	"cd": true, "echo": true, "test": true, "[": true, "[[": true, "true": true, "false": true,
	"exec": true, "export": true, "set": true, "unset": true, ":": true, ".": true, "source": true,
	"printf": true, "read": true, "eval": true, "exit": true, "wait": true, "umask": true,
	"if": true, "for": true, "while": true, "until": true, "case": true, "!": true, "{": true, "(": true,
}

// firstWord returns the program the shell will look up, skipping
// leading VAR=value assignments.
func firstWord(cmd string) string {
	for _, w := range strings.Fields(cmd) {
		if i := strings.IndexByte(w, '='); i > 0 && !strings.ContainsAny(w[:i], "/$'\"") {
			continue
		}
		return w
	}
	return ""
}

func (r relativePath) Check(ctx *Context) []Finding {
	var out []Finding
	for _, e := range ctx.Valid() {
		if _, ok := e.Env["PATH"]; ok {
			continue
		}
		w := firstWord(e.CommandLine())
		if w == "" || shellWords[w] || strings.HasPrefix(w, "/") || strings.HasPrefix(w, "$") || strings.HasPrefix(w, "~") {
			continue
		}
		var msg string
		if strings.Contains(w, "/") {
			msg = fmt.Sprintf("%q is relative to the job's working directory (usually $HOME) and PATH is not set", w)
		} else {
			msg = fmt.Sprintf("%q is not an absolute path and PATH is not set; cron's default PATH is usually only /usr/bin:/bin", w)
		}
		out = append(out, r.finding(e.Line,
			fmt.Sprintf("use the full path (find it with: command -v %s) or add PATH=/usr/local/bin:/usr/bin:/bin at the top of the crontab", strings.TrimPrefix(w, "./")),
			"%s", msg))
	}
	return out
}
