package staging

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// R3 review remediation (PR #43, Codex P1): the staging isolation gate
// fails CLOSED. A fake `docker` on PATH stands in for every probe outcome;
// only "control reached, outbound refused" (exit 4) counts as isolation.
// The real-container rehearsal runs in release-ci.yml (stage.sh verify) and
// in drill.sh.
func TestIsolationProbeFailsClosed(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	if _, err := exec.LookPath("timeout"); err != nil {
		t.Skip("coreutils timeout not available")
	}
	cases := []struct {
		name, body, want string
		pass             bool
	}{
		{"blocked after positive control", "exit 4", "blocked", true},
		{"outbound reachable", "exit 0", "OPEN", false},
		{"positive control failed", "exit 3", "UNKNOWN(probe_exit_3)", false},
		{"docker daemon error", "exit 125", "UNKNOWN(probe_exit_125)", false},
		{"cannot invoke", "exit 126", "UNKNOWN(probe_exit_126)", false},
		{"command not found", "exit 127", "UNKNOWN(probe_exit_127)", false},
		{"unexpected failure", "exit 1", "UNKNOWN(probe_exit_1)", false},
		{"probe hangs", "sleep 30", "UNKNOWN(probe_exit_124)", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bin := t.TempDir()
			argsFile := filepath.Join(bin, "args")
			script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > " + argsFile + "\n" + c.body + "\n"
			if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "stage.sh", "probe-isolation")
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "CRM_STAGING_PROBE_TIMEOUT=2")
			out, err := cmd.Output()
			passed := err == nil
			if passed != c.pass {
				t.Fatalf("gate passed=%v, want %v (output %s, err %v)", passed, c.pass, out, err)
			}
			var got struct {
				Egress string `json:"internal_egress"`
			}
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("output %q: %v", out, err)
			}
			if got.Egress != c.want {
				t.Fatalf("internal_egress = %q, want %q", got.Egress, c.want)
			}
			args, _ := os.ReadFile(argsFile)
			if !strings.Contains(string(args), "--network crm-staging_internal") {
				t.Fatalf("the probe must run on the staging internal network: %q", args)
			}
		})
	}
}

// verify must use the same fail-closed probe and gate on "blocked".
func TestVerifyGatesOnProvenIsolation(t *testing.T) {
	src, err := os.ReadFile("stage.sh")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	i := strings.Index(s, "\nverify)")
	if i < 0 {
		t.Fatal("verify subcommand missing")
	}
	body := s[i : i+strings.Index(s[i+1:], "\n  ;;")+1]
	if !strings.Contains(body, `egress="$(isolation_probe)"`) || !strings.Contains(body, `[ "$egress" = "blocked" ]`) {
		t.Fatal("verify must derive isolation from isolation_probe and require \"blocked\"")
	}
	if strings.Contains(body, `egress="blocked"`) {
		t.Fatal("verify must never default isolation to blocked")
	}
}
