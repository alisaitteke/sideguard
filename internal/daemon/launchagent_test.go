package daemon

import (
	"errors"
	"strings"
	"testing"
)

type mockLaunchctlRunner struct {
	calls [][]string
	errs  map[string]error
	outs  map[string]string
}

func (m *mockLaunchctlRunner) run(args ...string) (string, error) {
	m.calls = append(m.calls, append([]string(nil), args...))
	key := strings.Join(args, " ")
	if m.outs != nil {
		if out, ok := m.outs[key]; ok {
			return out, m.errs[key]
		}
	}
	if m.errs != nil {
		if err, ok := m.errs[key]; ok {
			return "", err
		}
	}
	return "", nil
}

func TestLaunchctlDomain(t *testing.T) {
	if got := launchctlDomain("501"); got != "gui/501" {
		t.Fatalf("got %q", got)
	}
}

func TestLaunchctlServiceID(t *testing.T) {
	got := launchctlServiceID("gui/501", "com.sideguard.daemon")
	if got != "gui/501/com.sideguard.daemon" {
		t.Fatalf("got %q", got)
	}
}

func TestIsBootoutNotLoaded(t *testing.T) {
	cases := []struct {
		out  string
		want bool
	}{
		{"Boot-out failed: 3: No such process", true},
		{"Could not find specified service", true},
		{"service is not running", true},
		{"Boot-out failed: 5: Input/output error", false},
	}
	for _, tc := range cases {
		if got := isBootoutNotLoaded(tc.out, errors.New("exit status")); got != tc.want {
			t.Fatalf("isBootoutNotLoaded(%q) = %v, want %v", tc.out, got, tc.want)
		}
	}
}

func TestIsBootstrapAlreadyLoaded(t *testing.T) {
	cases := []struct {
		out  string
		want bool
	}{
		{"Bootstrap failed: 5: Input/output error", true},
		{"service already bootstrapped", true},
		{"Bootstrap failed: 1: Operation not permitted", false},
	}
	for _, tc := range cases {
		if got := isBootstrapAlreadyLoaded(tc.out, errors.New("exit status")); got != tc.want {
			t.Fatalf("isBootstrapAlreadyLoaded(%q) = %v, want %v", tc.out, got, tc.want)
		}
	}
}

func TestLoadLaunchAgentBootoutThenBootstrap(t *testing.T) {
	runner := &mockLaunchctlRunner{}
	plist := "/Users/test/Library/LaunchAgents/com.sideguard.daemon.plist"

	if err := loadLaunchAgent(runner, "501", plist); err != nil {
		t.Fatal(err)
	}

	if len(runner.calls) < 2 {
		t.Fatalf("expected at least 2 calls, got %v", runner.calls)
	}
	if runner.calls[0][0] != "bootout" {
		t.Fatalf("first call should be bootout, got %v", runner.calls[0])
	}
	if runner.calls[1][0] != "bootstrap" {
		t.Fatalf("second call should be bootstrap, got %v", runner.calls[1])
	}
	if runner.calls[1][2] != plist {
		t.Fatalf("bootstrap plist mismatch: %v", runner.calls[1])
	}
}

func TestLoadLaunchAgentBootstrapAlreadyLoadedKickstart(t *testing.T) {
	runner := &mockLaunchctlRunner{
		errs: map[string]error{
			"bootstrap gui/501 /tmp/plist": errors.New("exit status 5"),
		},
		outs: map[string]string{
			"bootstrap gui/501 /tmp/plist": "Bootstrap failed: 5: Input/output error",
		},
	}

	if err := loadLaunchAgent(runner, "501", "/tmp/plist"); err != nil {
		t.Fatal(err)
	}

	var sawKickstart bool
	for _, call := range runner.calls {
		if len(call) > 0 && call[0] == "kickstart" {
			sawKickstart = true
			if call[1] != "-k" || call[2] != "gui/501/com.sideguard.daemon" {
				t.Fatalf("unexpected kickstart call: %v", call)
			}
		}
	}
	if !sawKickstart {
		t.Fatal("expected kickstart fallback")
	}
}

func TestLoadLaunchAgentBootstrapFailureIncludesManualHint(t *testing.T) {
	runner := &mockLaunchctlRunner{
		errs: map[string]error{
			"bootstrap gui/501 /tmp/plist": errors.New("exit status 1"),
		},
		outs: map[string]string{
			"bootstrap gui/501 /tmp/plist": "Bootstrap failed: 1: Operation not permitted",
		},
	}

	err := loadLaunchAgent(runner, "501", "/tmp/plist")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "launchctl bootout gui/501 com.sideguard.daemon") {
		t.Fatalf("missing manual recovery hint: %s", msg)
	}
	if !strings.Contains(msg, "already be loaded") {
		t.Fatalf("missing already-loaded hint: %s", msg)
	}
}

func TestBootoutIgnoresNotLoaded(t *testing.T) {
	runner := &mockLaunchctlRunner{
		errs: map[string]error{
			"bootout gui/501/com.sideguard.daemon": errors.New("exit status 3"),
			"bootout gui/501 /tmp/plist":           errors.New("exit status 3"),
		},
		outs: map[string]string{
			"bootout gui/501/com.sideguard.daemon": "Boot-out failed: 3: No such process",
			"bootout gui/501 /tmp/plist":           "Boot-out failed: 3: No such process",
		},
	}

	bootoutLaunchAgent(runner, "gui/501", "com.sideguard.daemon", "/tmp/plist")
	if len(runner.calls) != 1 {
		t.Fatalf("expected single bootout when service not loaded, got %v", runner.calls)
	}
}
