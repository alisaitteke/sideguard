package tray

import (
	"errors"
	"os"
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

func TestTrayLoadLaunchAgentBootoutThenBootstrap(t *testing.T) {
	runner := &mockLaunchctlRunner{}
	plist := "/Users/test/Library/LaunchAgents/com.sideguard.tray.plist"

	if err := loadTrayLaunchAgent(runner, "501", plist); err != nil {
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
}

func TestTrayUninstallServiceRemovesPlist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	plistDir := home + "/Library/LaunchAgents"
	if err := os.MkdirAll(plistDir, 0o755); err != nil {
		t.Fatal(err)
	}
	plist := plistDir + "/com.sideguard.tray.plist"
	if err := os.WriteFile(plist, []byte("<plist/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &mockLaunchctlRunner{}
	if err := uninstallServiceWithRunner(runner); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(plist); !os.IsNotExist(err) {
		t.Fatalf("expected plist removed, stat err=%v", err)
	}
	if len(runner.calls) == 0 || runner.calls[0][0] != "bootout" {
		t.Fatalf("expected bootout call, got %v", runner.calls)
	}
}

func TestTrayLoadLaunchAgentBootstrapAlreadyLoadedKickstart(t *testing.T) {
	runner := &mockLaunchctlRunner{
		errs: map[string]error{
			"bootstrap gui/501 /tmp/plist": errors.New("exit status 5"),
		},
		outs: map[string]string{
			"bootstrap gui/501 /tmp/plist": "Bootstrap failed: 5: Input/output error",
		},
	}

	if err := loadTrayLaunchAgent(runner, "501", "/tmp/plist"); err != nil {
		t.Fatal(err)
	}

	var sawKickstart bool
	for _, call := range runner.calls {
		if len(call) > 0 && call[0] == "kickstart" {
			sawKickstart = true
			if call[2] != "gui/501/com.sideguard.tray" {
				t.Fatalf("unexpected kickstart call: %v", call)
			}
		}
	}
	if !sawKickstart {
		t.Fatal("expected kickstart fallback")
	}
}
