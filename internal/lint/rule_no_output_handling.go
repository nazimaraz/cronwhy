package lint

import "strings"

func init() {
	register(noOutputHandling{meta{"no-output-handling", Info,
		"Jobs that neither redirect output nor have MAILTO set"}})
}

type noOutputHandling struct{ meta }

func handlesOutput(cmd string) bool {
	if strings.Contains(cmd, ">") {
		return true
	}
	for _, sink := range []string{"logger", "systemd-cat", "tee"} {
		if strings.Contains(cmd, "| "+sink) || strings.Contains(cmd, "|"+sink) {
			return true
		}
	}
	return false
}

func (r noOutputHandling) Check(ctx *Context) []Finding {
	var out []Finding
	for _, e := range ctx.Valid() {
		if _, ok := e.Env["MAILTO"]; ok || handlesOutput(e.CommandLine()) {
			continue
		}
		out = append(out, r.finding(e.Line,
			`append ">> /var/log/job.log 2>&1", or set MAILTO=you@example.com (MAILTO="" to discard output)`,
			"output is not redirected and MAILTO is not set; cron mails it to the local user, where it piles up unread or is lost if no MTA is installed"))
	}
	return out
}
