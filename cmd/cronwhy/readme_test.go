package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// consoleBlock matches README examples of the form
//
//	```console
//	$ cronwhy <args>
//	<output>
//	```
var consoleBlock = regexp.MustCompile("(?s)```console\n\\$ cronwhy ([^\n]*)\n(.*?)```")

// shellSplit splits a command line, honouring single and double quotes.
func shellSplit(s string) []string {
	var args []string
	var cur strings.Builder
	var quote rune
	inArg := false
	for _, r := range s {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote != 0:
			cur.WriteRune(r)
		case r == '"' || r == '\'':
			quote, inArg = r, true
		case r == ' ':
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			cur.WriteRune(r)
			inArg = true
		}
	}
	if inArg {
		args = append(args, cur.String())
	}
	return args
}

// TestREADMEExamples runs every console example in README.md and checks
// that the documented output is exactly what cronwhy prints.
func TestREADMEExamples(t *testing.T) {
	inRepoRoot(t)
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	readme := string(data)
	matches := consoleBlock.FindAllStringSubmatchIndex(readme, -1)
	if len(matches) < 4 {
		t.Fatalf("found only %d console examples", len(matches))
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		cmdline, want := readme[m[2]:m[3]], readme[m[4]:m[5]]
		e, out, errb := testEnv("")
		run(shellSplit(cmdline), e)
		got := out.String() + errb.String()
		if got != want && !*update {
			t.Errorf("README example %q is out of date (run go test ./cmd/cronwhy -update):\n--- README\n%s--- actual\n%s", cmdline, want, got)
		}
		b.WriteString(readme[last:m[4]])
		b.WriteString(got)
		last = m[5]
	}
	b.WriteString(readme[last:])
	if *update {
		if err := os.WriteFile("README.md", []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestShellSplit(t *testing.T) {
	got := shellSplit(`explain "*/15 9-17 * * 1-5" -n 5 --from '2026-10-27 20:00'`)
	want := []string{"explain", "*/15 9-17 * * 1-5", "-n", "5", "--from", "2026-10-27 20:00"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q", got)
	}
}
