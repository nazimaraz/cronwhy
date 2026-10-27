package lint

import "testing"

func TestUnescapedPercent(t *testing.T) {
	runRuleCases(t, "unescaped-percent", []ruleCase{
		{"date format", quiet + "0 3 * * * /bin/tar czf /b/$(date +%F).tgz /d\n", []int{3}},
		{"several", quiet + "0 3 * * * /bin/x $(date +%Y-%m-%d)\n", []int{3}},
		{"escaped", quiet + "0 3 * * * /bin/x $(date +\\%Y-\\%m-\\%d)\n", nil},
		{"none", quiet + "0 3 * * * /bin/x\n", nil},
		{"on invalid line too", quiet + "0 99 * * * /bin/x %s\n", []int{3}},
	})
}

func TestEscapePercents(t *testing.T) {
	got, n := escapePercents(`date +%F \%H`)
	if got != `date +\%F \%H` || n != 1 {
		t.Errorf("got %q, %d", got, n)
	}
}
