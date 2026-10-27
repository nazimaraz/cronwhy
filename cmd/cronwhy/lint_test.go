package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-username/cronwhy/internal/lint"
)

var update = flag.Bool("update", false, "rewrite golden files")

// inRepoRoot runs the test from the repository root so file names in the
// output read "testdata/...", exactly like the README examples.
func inRepoRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join("..", "..")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

var fixedLint = []string{"--from", "2026-10-27 20:00", "--tz", "UTC"}

func golden(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test ./cmd/cronwhy -update)", err)
	}
	if got != string(want) {
		t.Errorf("output differs from %s (run go test ./cmd/cronwhy -update to accept):\n%s", path, got)
	}
}

func TestLintGolden(t *testing.T) {
	inRepoRoot(t)
	for _, format := range []string{"text", "json"} {
		e, out, errb := testEnv("")
		code := run(append([]string{"lint", "testdata/broken.crontab", "--format", format}, fixedLint...), e)
		if code != exitErrors {
			t.Fatalf("%s: exit %d, want %d; stderr: %s", format, code, exitErrors, errb)
		}
		name := "testdata/broken.golden.txt"
		if format == "json" {
			name = "testdata/broken.golden.json"
		}
		golden(t, name, out.String())
	}
}

func TestBrokenTriggersEveryRule(t *testing.T) {
	inRepoRoot(t)
	e, out, _ := testEnv("")
	run(append([]string{"lint", "testdata/broken.crontab", "--format", "json"}, fixedLint...), e)
	var rep struct {
		Findings []struct{ Rule string } `json:"findings"`
	}
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range rep.Findings {
		seen[f.Rule] = true
	}
	for _, r := range lint.Rules() {
		if !seen[r.ID()] {
			t.Errorf("rule %s not triggered by testdata/broken.crontab", r.ID())
		}
	}
}

func TestLintExitCodes(t *testing.T) {
	inRepoRoot(t)
	tests := []struct {
		name  string
		args  []string
		stdin string
		code  int
		frag  string
	}{
		{"clean strict", []string{"lint", "testdata/clean.crontab", "--strict"}, "", exitOK, "no problems found (7 jobs checked)"},
		{"system strict", []string{"lint", "--system", "testdata/system.crontab", "--strict"}, "", exitOK, "no problems found"},
		{"broken", []string{"lint", "testdata/broken.crontab"}, "", exitErrors, "4 errors"},
		{"warnings only", []string{"lint", "-"}, "*/7 * * * * /bin/x >/dev/null\n", exitOK, "uneven-step"},
		{"warnings strict", []string{"lint", "-", "--strict"}, "*/7 * * * * /bin/x >/dev/null\n", exitWarnings, "uneven-step"},
		{"info strict", []string{"lint", "-", "--strict"}, "PATH=/bin\n3 3 * * * x\n", exitWarnings, "no-output-handling"},
		{"stdin error", []string{"lint", "-"}, "0 0 31 2 * /bin/x\n", exitErrors, "<stdin>:1:"},
		{"crontab -l", []string{"lint"}, "", exitOK, "crontab -l: no problems found"},
		{"missing file", []string{"lint", "testdata/nope.crontab"}, "", exitUsage, "no such file"},
		{"bad format", []string{"lint", "--format", "xml", "testdata/clean.crontab"}, "", exitUsage, "unknown --format"},
		{"bad window", []string{"lint", "--window", "7w", "testdata/clean.crontab"}, "", exitUsage, "invalid --window"},
		{"two files", []string{"lint", "a", "b"}, "", exitUsage, "at most one file"},
		{"bad flag", []string{"lint", "--bogus"}, "", exitUsage, "flag provided but not defined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, out, errb := testEnv(tt.stdin)
			code := run(append(tt.args, "--tz", "UTC"), e)
			all := out.String() + errb.String()
			if code != tt.code || !strings.Contains(all, tt.frag) {
				t.Errorf("exit %d (want %d), output:\n%s\nwant fragment %q", code, tt.code, all, tt.frag)
			}
		})
	}
}

func TestNoColorWhenNotTTY(t *testing.T) {
	inRepoRoot(t)
	for _, tc := range []struct {
		tty     bool
		noColor string
		want    bool
	}{{false, "", false}, {true, "1", false}, {true, "", true}} {
		e, out, _ := testEnv("")
		e.isTTY = tc.tty
		e.getenv = func(k string) string {
			if k == "NO_COLOR" {
				return tc.noColor
			}
			return ""
		}
		run(append([]string{"lint", "testdata/broken.crontab"}, fixedLint...), e)
		if got := strings.Contains(out.String(), "\x1b["); got != tc.want {
			t.Errorf("tty=%v NO_COLOR=%q: colored=%v", tc.tty, tc.noColor, got)
		}
	}
}
