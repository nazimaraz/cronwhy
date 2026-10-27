package lint

import "testing"

func TestImpossibleDate(t *testing.T) {
	runRuleCases(t, "impossible-date", []ruleCase{
		{"feb 31", quiet + "0 0 31 2 * /bin/true\n", []int{3}},
		{"april 31", quiet + "0 0 31 4 * /bin/true\n", []int{3}},
		{"feb 30 and 31", quiet + "0 0 30,31 FEB * /bin/true\n", []int{3}},
		{"leap day is possible", quiet + "0 0 29 2 * /bin/true\n", nil},
		{"or with weekday is possible", quiet + "0 0 31 2 MON /bin/true\n", nil},
		{"one valid month", quiet + "0 0 31 2,3 * /bin/true\n", nil},
	})
}
