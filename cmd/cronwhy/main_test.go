package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func testEnv(stdin string) (*env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &env{
		stdin:    strings.NewReader(stdin),
		stdout:   &out,
		stderr:   &errb,
		now:      func() time.Time { return time.Date(2026, 10, 27, 17, 0, 0, 0, time.UTC) },
		getenv:   func(string) string { return "" },
		crontabL: func() ([]byte, error) { return []byte("0 3 * * * /usr/bin/true >/dev/null 2>&1\n"), nil },
	}, &out, &errb
}

func TestExplainDefinitionOfDone(t *testing.T) {
	e, out, _ := testEnv("")
	code := run([]string{"explain", "*/15 9-17 * * 1-5", "-n", "5", "--from", "2026-10-27 20:00", "--tz", "Europe/Istanbul"}, e)
	if code != exitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{
		"At every 15th minute, from 09:00 through 17:59, Monday through Friday",
		"Wed 2026-10-28 09:00 +03  in 13h 0m",
		"Wed 2026-10-28 10:00 +03  in 14h 0m",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if got := strings.Count(out.String(), "+03  in"); got != 5 {
		t.Errorf("got %d run lines, want 5", got)
	}
}

func TestExplainExitCodes(t *testing.T) {
	tests := []struct {
		args []string
		code int
		frag string
	}{
		{[]string{"explain", "0 25 * * *"}, exitErrors, `hour field "25"`},
		{[]string{"explain", "0 0 31 2 *"}, exitErrors, "never fire"},
		{[]string{"explain", "@reboot"}, exitOK, "startup"},
		{[]string{"explain"}, exitUsage, "Usage"},
		{[]string{"explain", "* * * * *", "--tz", "Mars/Olympus"}, exitUsage, "unknown timezone"},
		{[]string{"explain", "5/15 * * * *"}, exitErrors, "hint: use a range: 5-59/15"},
		{[]string{"bogus"}, exitUsage, "unknown command"},
		{[]string{"next", "0 0 * * *", "-n", "2", "--tz", "UTC"}, exitOK, "2026-10-28T00:00:00Z\n2026-10-29T00:00:00Z\n"},
		{[]string{"next", "0", "12", "*", "*", "*", "-n", "1", "--tz", "UTC"}, exitOK, "2026-10-28T12:00:00Z"},
	}
	for _, tt := range tests {
		e, out, errb := testEnv("")
		code := run(tt.args, e)
		all := out.String() + errb.String()
		if code != tt.code || !strings.Contains(all, tt.frag) {
			t.Errorf("%v: exit %d (want %d), output:\n%s\nwant fragment %q", tt.args, code, tt.code, all, tt.frag)
		}
	}
}
