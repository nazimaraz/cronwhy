package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/your-username/cronwhy/internal/crontab"
	"github.com/your-username/cronwhy/internal/lint"
)

func sampleFile(t *testing.T) (*crontab.File, []lint.Finding) {
	f, err := crontab.Parse(strings.NewReader("MAILTO=a@b\n0 3 * * * /bin/x\n"), "jobs.cron", crontab.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return f, []lint.Finding{
		{Rule: "invalid-field", Severity: lint.Error, Line: 2, Message: "bad", Fix: "fix it"},
		{Rule: "every-minute", Severity: lint.Info, Line: 2, Message: "often"},
	}
}

func TestText(t *testing.T) {
	f, fs := sampleFile(t)
	var b bytes.Buffer
	Text(&b, []Result{{f, fs}}, false)
	want := `jobs.cron:2: error: bad (invalid-field)
    0 3 * * * /bin/x
    fix: fix it
jobs.cron:2: info: often (every-minute)
    0 3 * * * /bin/x

jobs.cron: 1 error, 0 warnings, 1 info (1 job checked)
`
	if b.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", b.String(), want)
	}
}

func TestTextColorAndClean(t *testing.T) {
	f, fs := sampleFile(t)
	var b bytes.Buffer
	Text(&b, []Result{{f, fs}}, true)
	if !strings.Contains(b.String(), "\x1b[31merror:\x1b[0m") {
		t.Errorf("no color: %q", b.String())
	}
	b.Reset()
	Text(&b, []Result{{f, nil}}, false)
	if b.String() != "ok jobs.cron: no problems found (1 job checked)\n" {
		t.Errorf("clean output = %q", b.String())
	}
}

func TestUseColor(t *testing.T) {
	if UseColor(false, "") || UseColor(true, "1") || !UseColor(true, "") {
		t.Error("UseColor logic wrong")
	}
}

func TestJSON(t *testing.T) {
	f, fs := sampleFile(t)
	var b bytes.Buffer
	if err := JSON(&b, []Result{{f, fs}}); err != nil {
		t.Fatal(err)
	}
	var rep map[string]any
	if err := json.Unmarshal(b.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if rep["version"].(float64) != 1 || rep["summary"].(map[string]any)["errors"].(float64) != 1 {
		t.Errorf("header = %v", rep)
	}
	first := rep["findings"].([]any)[0].(map[string]any)
	if first["file"] != "jobs.cron" || first["severity"] != "error" || first["line"].(float64) != 2 || first["source"] != "0 3 * * * /bin/x" {
		t.Errorf("finding = %v", first)
	}
	b.Reset()
	_ = JSON(&b, []Result{{f, nil}})
	if !strings.Contains(b.String(), `"findings": []`) {
		t.Errorf("empty findings must be [], got %s", b.String())
	}
}

func TestTextMultipleFiles(t *testing.T) {
	f, fs := sampleFile(t)
	g, _ := sampleFile(t)
	g.Name = "other.cron"
	var b bytes.Buffer
	Text(&b, []Result{{f, fs}, {g, nil}}, false)
	if !strings.HasSuffix(b.String(), "\n2 files: 1 error, 0 warnings, 1 info (2 jobs checked)\n") {
		t.Errorf("got:\n%s", b.String())
	}
}
