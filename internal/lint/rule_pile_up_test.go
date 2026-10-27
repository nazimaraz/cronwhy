package lint

import (
	"strings"
	"testing"

	"github.com/nazimaraz/cronwhy/internal/crontab"
)

func TestPileUp(t *testing.T) {
	five := quiet + "0 0 * * * /bin/a\n0 0 * * * /bin/b\n0 * * * * /bin/c\n@daily /bin/d\n@midnight /bin/e\n"
	runRuleCases(t, "pile-up", []ruleCase{
		{"five at midnight", five, []int{3}},
		{"four only", quiet + "0 0 * * * /bin/a\n0 0 * * * /bin/b\n0 * * * * /bin/c\n@daily /bin/d\n", nil},
		{"staggered", quiet + "3 0 * * * /bin/a\n17 0 * * * /bin/b\n34 * * * * /bin/c\n48 0 * * * /bin/d\n51 0 * * * /bin/e\n", nil},
		{"reboot ignored", quiet + "@reboot /bin/a\n@reboot /bin/b\n@reboot /bin/c\n@reboot /bin/d\n@reboot /bin/e\n", nil},
		{"weekly cluster", quiet + "0 6 * * 1 /bin/a\n0 6 * * 1 /bin/b\n0 6 * * 1 /bin/c\n0 6 * * 1 /bin/d\n0 6 * * 1 /bin/e\n", []int{3}},
	})
}

func TestPileUpMessage(t *testing.T) {
	src := "0 0 * * * /bin/a\n0 0 * * * /bin/b\n0 * * * * /bin/c\n@daily /bin/d\n@midnight /bin/e\n"
	f, _ := crontab.Parse(strings.NewReader(src), "t", crontab.Options{})
	fs := pileUp{meta{"pile-up", Warning, ""}}.Check(&Context{File: f, Config: testConfig()})
	if len(fs) != 1 || !strings.Contains(fs[0].Message, "5 jobs start in the same minute at 00:00 (lines 1, 2, 3, 4, 5); this happens 7 time(s) in the 7d window") {
		t.Errorf("findings = %+v", fs)
	}
}
