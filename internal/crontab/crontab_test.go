package crontab

import (
	"errors"
	"strings"
	"testing"

	"github.com/your-username/cronwhy/internal/cronexpr"
)

const sample = `# comment
SHELL=/bin/bash
MAILTO="ops@example.com"

*/5 * * * * /usr/bin/check
PATH = /usr/local/bin:/usr/bin
CRON_TZ=America/New_York
@daily /usr/bin/backup >/dev/null 2>&1
0 25 * * * /bin/broken
0 0 * *
   # indented comment
0 0 L * * /bin/quartz
* * * * * echo "a=b"
`

func TestParse(t *testing.T) {
	f, err := Parse(strings.NewReader(sample), "sample", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Env) != 4 {
		t.Fatalf("env = %+v", f.Env)
	}
	if f.Env[1].Name != "MAILTO" || f.Env[1].Value != "ops@example.com" || f.Env[2].Name != "PATH" {
		t.Errorf("env parsed wrong: %+v", f.Env)
	}
	type want struct {
		line     int
		sched    string
		cmd      string
		hasPath  bool
		tz       string
		errField cronexpr.FieldKind
		syntax   bool
	}
	wants := []want{
		{5, "*/5 * * * *", "/usr/bin/check", false, "", -2, false},
		{8, "@daily", "/usr/bin/backup >/dev/null 2>&1", true, "America/New_York", -2, false},
		{9, "0 25 * * *", "/bin/broken", true, "America/New_York", cronexpr.Hour, false},
		{10, "", "", true, "America/New_York", -2, true},
		{12, "0 0 L * *", "/bin/quartz", true, "America/New_York", cronexpr.DayOfMonth, false},
		{13, "* * * * *", `echo "a=b"`, true, "America/New_York", -2, false},
	}
	if len(f.Entries) != len(wants) {
		t.Fatalf("got %d entries, want %d", len(f.Entries), len(wants))
	}
	for i, w := range wants {
		e := f.Entries[i]
		if e.Line != w.line || e.ScheduleText != w.sched || e.Command != w.cmd {
			t.Errorf("entry %d = line %d %q %q, want %+v", i, e.Line, e.ScheduleText, e.Command, w)
		}
		if _, ok := e.Env["PATH"]; ok != w.hasPath || e.Env["CRON_TZ"] != w.tz {
			t.Errorf("line %d env = %v", e.Line, e.Env)
		}
		var pe *cronexpr.ParseError
		var se *SyntaxError
		switch {
		case w.errField != -2:
			if !errors.As(e.Err, &pe) || pe.Field != w.errField {
				t.Errorf("line %d err = %v, want field %v", e.Line, e.Err, w.errField)
			}
		case w.syntax:
			if !errors.As(e.Err, &se) {
				t.Errorf("line %d err = %v, want syntax error", e.Line, e.Err)
			}
		case e.Err != nil:
			t.Errorf("line %d unexpected err %v", e.Line, e.Err)
		}
	}
}

func TestParseSystem(t *testing.T) {
	src := "17 * * * * root cd / && run-parts --report /etc/cron.hourly\n@reboot www-data /usr/bin/start\n0 0 * * * /usr/bin/nouser\n"
	f, err := Parse(strings.NewReader(src), "crontab", Options{System: true})
	if err != nil {
		t.Fatal(err)
	}
	e := f.Entries
	if e[0].User != "root" || e[0].Command != "cd / && run-parts --report /etc/cron.hourly" || e[0].Err != nil {
		t.Errorf("entry 0 = %+v", e[0])
	}
	if e[1].User != "www-data" || !e[1].Schedule.Reboot() {
		t.Errorf("entry 1 = %+v", e[1])
	}
	// "/usr/bin/nouser" is taken as the user and the command is missing.
	if e[2].Err == nil {
		t.Errorf("entry 2 should fail: %+v", e[2])
	}
}

func TestCommandLine(t *testing.T) {
	tests := map[string]string{
		`/bin/backup`:                 `/bin/backup`,
		`date +%Y-%m-%d`:              `date +`,
		`date +\%Y-\%m-\%d > /tmp/x`:  `date +%Y-%m-%d > /tmp/x`,
		`mail -s hi root%body%line 2`: `mail -s hi root`,
	}
	for cmd, want := range tests {
		e := &Entry{Command: cmd}
		if got := e.CommandLine(); got != want {
			t.Errorf("CommandLine(%q) = %q, want %q", cmd, got, want)
		}
	}
}
