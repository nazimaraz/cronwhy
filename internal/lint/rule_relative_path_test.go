package lint

import "testing"

func TestRelativePath(t *testing.T) {
	mail := "MAILTO=ops@example.com\n"
	runRuleCases(t, "relative-path", []ruleCase{
		{"bare name", mail + "0 3 * * * python3 /opt/x.py\n", []int{2}},
		{"dot slash", mail + "0 3 * * * ./run.sh\n", []int{2}},
		{"env prefix", mail + "0 3 * * * LANG=C backup.sh\n", []int{2}},
		{"absolute", mail + "0 3 * * * /usr/bin/python3 /opt/x.py\n", nil},
		{"builtin", mail + "0 3 * * * cd /opt && /opt/run\n", nil},
		{"variable", mail + "0 3 * * * $HOME/bin/run\n", nil},
		{"path set", "PATH=/usr/local/bin:/usr/bin\n" + mail + "0 3 * * * python3 x.py\n", nil},
		{"path set after", mail + "0 3 * * * python3 x.py\nPATH=/usr/bin\n", []int{2}},
	})
}
