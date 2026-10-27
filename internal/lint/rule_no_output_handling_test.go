package lint

import "testing"

func TestNoOutputHandling(t *testing.T) {
	path := "PATH=/usr/bin:/bin\n"
	runRuleCases(t, "no-output-handling", []ruleCase{
		{"bare", path + "0 3 * * * /bin/a\n", []int{2}},
		{"redirect", path + "0 3 * * * /bin/a >/dev/null 2>&1\n", nil},
		{"append", path + "0 3 * * * /bin/a >> /var/log/a.log\n", nil},
		{"logger", path + "0 3 * * * /bin/a 2>&1 | logger -t a\n", nil},
		{"mailto", path + "MAILTO=me@example.com\n0 3 * * * /bin/a\n", nil},
		{"empty mailto", path + "MAILTO=\"\"\n0 3 * * * /bin/a\n", nil},
		{"mailto after job", path + "0 3 * * * /bin/a\nMAILTO=me@example.com\n", []int{2}},
	})
}
