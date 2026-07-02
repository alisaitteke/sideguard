package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupProviderHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestRunProviderAddAndList(t *testing.T) {
	setupProviderHome(t)

	providerAddID = "my-openai"
	providerAddDriver = "openai"
	providerAddModel = "gpt-4o-mini"
	providerAddBaseURL = ""
	providerAddDefault = true

	if err := runProviderAdd(nil, nil); err != nil {
		t.Fatalf("add: %v", err)
	}

	if err := configSetProviderKeyForTest("my-openai", "sk-test-secret-key"); err != nil {
		t.Fatalf("set key: %v", err)
	}

	providerListJSON = false
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := runProviderList(nil, nil); err != nil {
		t.Fatalf("list: %v", err)
	}
	w.Close()
	os.Stdout = oldStdout
	_, _ = buf.ReadFrom(r)

	// Re-run list capturing to buf via direct call
	providerListJSON = true
	oldStdout = os.Stdout
	r, w, _ = os.Pipe()
	os.Stdout = w
	if err := runProviderList(nil, nil); err != nil {
		t.Fatalf("list json: %v", err)
	}
	w.Close()
	os.Stdout = oldStdout
	_, _ = buf.ReadFrom(r)

	providerListJSON = false
	oldStdout = os.Stdout
	r, w, _ = os.Pipe()
	os.Stdout = w
	if err := runProviderList(nil, nil); err != nil {
		t.Fatalf("list table: %v", err)
	}
	w.Close()
	os.Stdout = oldStdout
	var outBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(r)
	out := outBuf.String()

	if !strings.Contains(out, "my-openai") || !strings.Contains(out, "openai") || !strings.Contains(out, "gpt-4o-mini") {
		t.Fatalf("table missing provider row:\n%s", out)
	}
	if !strings.Contains(out, "*") {
		t.Fatalf("expected default marker in table:\n%s", out)
	}
	if strings.Contains(out, "sk-test-secret-key") {
		t.Fatalf("raw key leaked in list output:\n%s", out)
	}
}

func TestRunProviderListJSON(t *testing.T) {
	setupProviderHome(t)

	providerAddID = "p1"
	providerAddDriver = "anthropic"
	providerAddModel = "claude-3-5-sonnet"
	providerAddDefault = false
	if err := runProviderAdd(nil, nil); err != nil {
		t.Fatal(err)
	}

	providerListJSON = true
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	if err := runProviderList(nil, nil); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	var rows []providerListRow
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("json: %v\n%s", err, buf.String())
	}
	if len(rows) != 1 || rows[0].ID != "p1" || rows[0].Driver != "anthropic" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestRunProviderAddDuplicate(t *testing.T) {
	setupProviderHome(t)

	providerAddID = "dup"
	providerAddDriver = "openai"
	providerAddModel = "gpt-4o"
	if err := runProviderAdd(nil, nil); err != nil {
		t.Fatal(err)
	}
	err := runProviderAdd(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err = %v, want duplicate error", err)
	}
}

func TestRunProviderSetKeyRejectsEmpty(t *testing.T) {
	setupProviderHome(t)

	providerAddID = "k1"
	providerAddDriver = "openai"
	providerAddModel = "gpt-4o"
	if err := runProviderAdd(nil, nil); err != nil {
		t.Fatal(err)
	}

	providerSetKeyID = "k1"
	providerSetKeyValue = "   "
	err := runProviderSetKey(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("err = %v, want empty key rejection", err)
	}
}

func TestRunProviderSetKeyDoesNotEchoKey(t *testing.T) {
	setupProviderHome(t)

	providerAddID = "k2"
	providerAddDriver = "openai"
	providerAddModel = "gpt-4o"
	if err := runProviderAdd(nil, nil); err != nil {
		t.Fatal(err)
	}

	providerSetKeyID = "k2"
	providerSetKeyValue = "sk-super-secret-key"
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	if err := runProviderSetKey(nil, nil); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	if strings.Contains(out, "sk-super-secret-key") {
		t.Fatalf("key echoed in output: %q", out)
	}
	if !strings.Contains(out, `api key set for provider "k2"`) {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRunProviderRemoveDefaultRejected(t *testing.T) {
	setupProviderHome(t)

	providerAddID = "def"
	providerAddDriver = "openai"
	providerAddModel = "gpt-4o"
	providerAddDefault = true
	if err := runProviderAdd(nil, nil); err != nil {
		t.Fatal(err)
	}

	providerRemoveID = "def"
	err := runProviderRemove(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot remove default provider") {
		t.Fatalf("err = %v, want default removal error", err)
	}
}

func TestRunProviderRemoveAndSetDefault(t *testing.T) {
	setupProviderHome(t)

	providerAddID = "keep"
	providerAddDriver = "openai"
	providerAddModel = "gpt-4o"
	providerAddDefault = true
	if err := runProviderAdd(nil, nil); err != nil {
		t.Fatal(err)
	}
	providerAddID = "drop"
	if err := runProviderAdd(nil, nil); err != nil {
		t.Fatal(err)
	}

	providerDefaultID = "keep"
	if err := runProviderSetDefault(nil, nil); err != nil {
		t.Fatal(err)
	}

	providerRemoveID = "drop"
	if err := runProviderRemove(nil, nil); err != nil {
		t.Fatalf("remove: %v", err)
	}

	providerListJSON = true
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	if err := runProviderList(nil, nil); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	var rows []providerListRow
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "keep" {
		t.Fatalf("rows after remove = %+v", rows)
	}
}

func configSetProviderKeyForTest(id, key string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".sideguard")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	providerSetKeyID = id
	providerSetKeyValue = key
	return runProviderSetKey(nil, nil)
}
