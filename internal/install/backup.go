package install

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alisaitteke/sideguard/internal/paths"
)

const manifestName = "manifest.json"

// manifest records a single install backup session.
type manifest struct {
	Timestamp string            `json:"timestamp"`
	Files     map[string]string `json:"files"` // original path -> backup relative path
}

// BackupSession is a timestamped backup directory under ~/.sideguard/backups/.
type BackupSession struct {
	Dir       string
	Timestamp string
}

// CreateBackup copies the given files into ~/.sideguard/backups/<timestamp>/.
// Only files that exist are backed up. Returns the session metadata.
func CreateBackup(filePaths []string) (*BackupSession, error) {
	base, err := paths.BackupsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, fmt.Errorf("create backups dir: %w", err)
	}

	ts := time.Now().UTC().Format("20060102-150405")
	sessionDir := filepath.Join(base, ts)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup session: %w", err)
	}

	m := manifest{
		Timestamp: ts,
		Files:     make(map[string]string),
	}

	for _, src := range uniqueStrings(filePaths) {
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", src, err)
		}

		relName := backupRelName(src)
		dst := filepath.Join(sessionDir, relName)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, fmt.Errorf("mkdir backup parent: %w", err)
		}
		if err := copyFile(src, dst); err != nil {
			return nil, fmt.Errorf("backup %s: %w", src, err)
		}
		m.Files[src] = relName
	}

	manifestPath := filepath.Join(sessionDir, manifestName)
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return &BackupSession{Dir: sessionDir, Timestamp: ts}, nil
}

// RestoreLatest restores each file from the most recent backup session that contains it.
func RestoreLatest(filePaths []string) error {
	for _, original := range uniqueStrings(filePaths) {
		session, rel, err := findLatestBackup(original)
		if err != nil {
			return err
		}
		if session == "" {
			continue
		}
		if err := restoreFromSession(session, rel, original); err != nil {
			return err
		}
	}
	return nil
}

// RestoreFirst restores each file from the oldest backup session that contains it.
// Use this for uninstall fallback: the first install backup predates SideGuard patches.
func RestoreFirst(filePaths []string) error {
	for _, original := range uniqueStrings(filePaths) {
		session, rel, err := findFirstBackup(original)
		if err != nil {
			return err
		}
		if session == "" {
			continue
		}
		if err := restoreFromSession(session, rel, original); err != nil {
			return err
		}
	}
	return nil
}

func restoreFromSession(sessionDir, relName, original string) error {
	src := filepath.Join(sessionDir, relName)
	if err := copyFile(src, original); err != nil {
		return fmt.Errorf("restore %s: %w", original, err)
	}
	return nil
}

func listBackupSessions() ([]string, error) {
	base, err := paths.BackupsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var sessions []string
	for _, e := range entries {
		if e.IsDir() {
			sessions = append(sessions, e.Name())
		}
	}
	return sessions, nil
}

func findBackupInSessions(sessions []string, originalPath string, newestFirst bool) (sessionDir, relName string, err error) {
	base, err := paths.BackupsDir()
	if err != nil {
		return "", "", err
	}
	ordered := append([]string(nil), sessions...)
	if newestFirst {
		sort.Sort(sort.Reverse(sort.StringSlice(ordered)))
	} else {
		sort.Strings(ordered)
	}

	for _, name := range ordered {
		sessionDir := filepath.Join(base, name)
		manifestPath := filepath.Join(sessionDir, manifestName)
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var m manifest
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		if rel, ok := m.Files[originalPath]; ok {
			return sessionDir, rel, nil
		}
	}
	return "", "", nil
}

func findLatestBackup(originalPath string) (sessionDir, relName string, err error) {
	sessions, err := listBackupSessions()
	if err != nil {
		return "", "", err
	}
	return findBackupInSessions(sessions, originalPath, true)
}

func findFirstBackup(originalPath string) (sessionDir, relName string, err error) {
	sessions, err := listBackupSessions()
	if err != nil {
		return "", "", err
	}
	return findBackupInSessions(sessions, originalPath, false)
}

func backupRelName(original string) string {
	home, err := os.UserHomeDir()
	if err == nil {
		if strings.HasPrefix(original, home+string(os.PathSeparator)) {
			return "home" + strings.TrimPrefix(original, home)
		}
	}
	cwd, err := os.Getwd()
	if err == nil && strings.HasPrefix(original, cwd+string(os.PathSeparator)) {
		return "cwd" + strings.TrimPrefix(original, cwd)
	}
	return filepath.Base(original)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
