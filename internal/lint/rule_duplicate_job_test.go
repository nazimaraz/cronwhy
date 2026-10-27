package lint

import "testing"

func TestDuplicateJob(t *testing.T) {
	runRuleCases(t, "duplicate-job", []ruleCase{
		{"identical", quiet + "0 3 * * * /bin/a\n0 3 * * * /bin/a\n", []int{4}},
		{"macro vs fields", quiet + "@daily /bin/a\n0 0 * * * /bin/a\n", []int{4}},
		{"whitespace", quiet + "0 3 * * * /bin/a  -v\n0 3 * * * /bin/a -v\n", []int{4}},
		{"three copies", quiet + "@hourly /bin/a\n@hourly /bin/a\n@hourly /bin/a\n", []int{4, 5}},
		{"different command", quiet + "0 3 * * * /bin/a\n0 3 * * * /bin/b\n", nil},
		{"different schedule", quiet + "0 3 * * * /bin/a\n0 4 * * * /bin/a\n", nil},
	})
}
