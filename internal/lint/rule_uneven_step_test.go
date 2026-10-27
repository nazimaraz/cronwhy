package lint

import (
	"strings"
	"testing"

	"github.com/your-username/cronwhy/internal/cronexpr"
)

func TestUnevenStep(t *testing.T) {
	runRuleCases(t, "uneven-step", []ruleCase{
		{"minute */7", quiet + "*/7 * * * * /bin/true\n", []int{3}},
		{"hour */5", quiet + "0 */5 * * * /bin/true\n", []int{3}},
		{"full range step", quiet + "0-59/7 * * * * /bin/true\n", []int{3}},
		{"dow */2", quiet + "0 0 * * */2 /bin/true\n", []int{3}},
		{"two fields", quiet + "*/7 */5 * * * /bin/true\n", []int{3, 3}},
		{"even minute", quiet + "*/15 * * * * /bin/true\n", nil},
		{"even hour", quiet + "0 */6 * * * /bin/true\n", nil},
		{"partial range does not wrap", quiet + "10-50/7 * * * * /bin/true\n", nil},
		{"month */3", quiet + "0 0 1 */3 * /bin/true\n", nil},
	})
}

func TestUnevenStepMessage(t *testing.T) {
	s, _ := cronexpr.Parse("*/7 * * * *")
	msg, fix, ok := unevenTerm(cronexpr.Minute, s.Fields[0].Terms[0])
	if !ok || !strings.Contains(msg, "0,7,14,21,28,35,42,49,56") || !strings.Contains(msg, "gap of 4 minute(s)") {
		t.Errorf("msg = %q", msg)
	}
	if !strings.Contains(fix, "*/5") {
		t.Errorf("fix = %q", fix)
	}
}
