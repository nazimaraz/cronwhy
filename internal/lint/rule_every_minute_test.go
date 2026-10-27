package lint

import "testing"

func TestEveryMinute(t *testing.T) {
	runRuleCases(t, "every-minute", []ruleCase{
		{"stars", quiet + "* * * * * /bin/true\n", []int{3}},
		{"step one", quiet + "*/1 * * * 1-5 /bin/true\n", []int{3}},
		{"full range", quiet + "0-59 0-23 * * * /bin/true\n", []int{3}},
		{"every five", quiet + "*/5 * * * * /bin/true\n", nil},
		{"one hour only", quiet + "* 3 * * * /bin/true\n", nil},
		{"reboot", quiet + "@reboot /bin/true\n", nil},
	})
}
