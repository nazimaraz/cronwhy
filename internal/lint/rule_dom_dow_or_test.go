package lint

import "testing"

func TestDomDowOr(t *testing.T) {
	runRuleCases(t, "dom-dow-or", []ruleCase{
		{"both restricted", quiet + "0 0 1 * MON /bin/true\n", []int{3}},
		{"range and range", quiet + "0 0 1-7 * 5 /bin/true\n", []int{3}},
		{"only dom", quiet + "0 0 1 * * /bin/true\n", nil},
		{"only dow", quiet + "0 0 * * 1-5 /bin/true\n", nil},
		{"star step dom uses AND", quiet + "0 0 */2 * 1 /bin/true\n", nil},
		{"macro", quiet + "@monthly /bin/true\n", nil},
	})
}
