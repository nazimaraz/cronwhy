package lint

import (
	"strings"
	"testing"
	"time"

	"github.com/nazimaraz/cronwhy/internal/crontab"
)

func TestDSTRisk(t *testing.T) {
	ny := quiet + "CRON_TZ=America/New_York\n"
	london := quiet + "CRON_TZ=Europe/London\n"
	ist := quiet + "CRON_TZ=Europe/Istanbul\n"
	runRuleCases(t, "dst-risk", []ruleCase{
		{"ny 02:30 skipped", ny + "30 2 * * * /bin/true\n", []int{4}},
		{"ny 01:30 doubled", ny + "30 1 * * * /bin/true\n", []int{4}},
		{"ny hourly", ny + "0 * * * * /bin/true\n", []int{4}},
		{"ny safe", ny + "30 3 * * * /bin/true\n", nil},
		{"ny wrong day", ny + "30 2 * 6 * /bin/true\n", nil},
		{"london 01:30", london + "30 1 * * * /bin/true\n", []int{4}},
		{"london 02:30 safe", london + "30 2 * * * /bin/true\n", nil},
		{"istanbul has no DST", ist + "30 2 * * * /bin/true\n", nil},
		{"utc default", quiet + "30 2 * * * /bin/true\n", nil},
	})
}

func TestDSTRiskMessage(t *testing.T) {
	f, _ := crontab.Parse(strings.NewReader("CRON_TZ=America/New_York\n0 * * * * /bin/true\n"), "t", crontab.Options{})
	fs := dstRisk{meta{"dst-risk", Warning, ""}}.Check(&Context{File: f, Config: testConfig()})
	if len(fs) != 1 {
		t.Fatalf("findings = %+v", fs)
	}
	for _, want := range []string{"from 02:00 to 03:00", "runs at 02:00 may be skipped", "from 02:00 to 01:00", "runs at 01:00 may happen twice"} {
		if !strings.Contains(fs[0].Message, want) {
			t.Errorf("message %q lacks %q", fs[0].Message, want)
		}
	}
}

func TestDSTRiskConfigLocation(t *testing.T) {
	lon, _ := time.LoadLocation("Europe/London")
	f, _ := crontab.Parse(strings.NewReader("30 1 * * * /bin/true\n"), "t", crontab.Options{})
	cfg := testConfig()
	cfg.Location = lon
	if fs := (dstRisk{meta{"dst-risk", Warning, ""}}).Check(&Context{File: f, Config: cfg}); len(fs) != 1 {
		t.Errorf("--tz Europe/London should trigger: %+v", fs)
	}
}
