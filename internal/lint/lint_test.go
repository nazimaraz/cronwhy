package lint

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/your-username/cronwhy/internal/crontab"
)

var testFrom = time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)

func testConfig() Config {
	return Config{Location: time.UTC, From: testFrom, Window: 7 * 24 * time.Hour, PileUpThreshold: 5}
}

// ruleCase is shared by the table-driven tests of every rule.
type ruleCase struct {
	name  string
	src   string
	lines []int // lines expected to trigger the rule; nil means none
}

// headerless crontabs get PATH and MAILTO so that unrelated rules stay
// quiet; rules that need them unset build their own source.
const quiet = "PATH=/usr/bin:/bin\nMAILTO=ops@example.com\n"

func runRuleCases(t *testing.T, id string, cases []ruleCase) {
	t.Helper()
	r, ok := Lookup(id)
	if !ok {
		t.Fatalf("rule %s not registered", id)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := crontab.Parse(strings.NewReader(tc.src), "test", crontab.Options{})
			if err != nil {
				t.Fatal(err)
			}
			findings := r.Check(&Context{File: f, Config: testConfig()})
			var lines []int
			for _, fd := range findings {
				if fd.Rule != id || fd.Severity != r.Severity() || fd.Message == "" || fd.Fix == "" {
					t.Errorf("malformed finding %+v", fd)
				}
				lines = append(lines, fd.Line)
			}
			if !reflect.DeepEqual(lines, tc.lines) {
				t.Errorf("lines = %v, want %v", lines, tc.lines)
				for _, fd := range findings {
					t.Logf("  %d: %s", fd.Line, fd.Message)
				}
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	want := []string{"dom-dow-or", "dst-risk", "duplicate-job", "every-minute", "impossible-date",
		"invalid-field", "no-output-handling", "pile-up", "relative-path", "unescaped-percent", "uneven-step"}
	var got []string
	for _, r := range Rules() {
		got = append(got, r.ID())
		if r.Summary() == "" {
			t.Errorf("%s has no summary", r.ID())
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rules = %v", got)
	}
}

func TestRunSorted(t *testing.T) {
	src := "* * * * * date +%s\n0 25 * * * /bin/x\n"
	f, _ := crontab.Parse(strings.NewReader(src), "t", crontab.Options{})
	fs := Run(f, testConfig())
	for i := 1; i < len(fs); i++ {
		a, b := fs[i-1], fs[i]
		if a.Line > b.Line || (a.Line == b.Line && a.Severity < b.Severity) {
			t.Fatalf("not sorted: %+v before %+v", a, b)
		}
	}
	if fs[0].Rule != "unescaped-percent" {
		t.Errorf("first finding = %+v", fs[0])
	}
}
