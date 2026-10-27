package lint

import "testing"

func TestInvalidField(t *testing.T) {
	runRuleCases(t, "invalid-field", []ruleCase{
		{"valid", quiet + "0 3 * * * /bin/true\n", nil},
		{"out of range", quiet + "60 3 * * * /bin/true\n", []int{3}},
		{"reversed range", quiet + "0 5-1 * * * /bin/true\n", []int{3}},
		{"bad step", quiet + "*/0 * * * * /bin/true\n", []int{3}},
		{"unknown name", quiet + "0 0 * * FUN /bin/true\n", []int{3}},
		{"quartz", quiet + "0 0 L * * /bin/true\n", []int{3}},
		{"incomplete", quiet + "0 0 * *\n", []int{3}},
		{"missing command", quiet + "0 0 * * *\n", []int{3}},
		{"keeps going", quiet + "0 99 * * * /bin/a\n0 1 * * * /bin/b\n@often /bin/c\n", []int{3, 5}},
	})
}
