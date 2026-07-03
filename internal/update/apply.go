// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alisaitteke/sideguard/internal/paths"
)

// Applier orchestrates download, verify, extract, and atomic binary replacement.
type Applier struct {
	downloader *Downloader
	goos       string
}

// NewApplier creates an applier for the current or injected platform.
func NewApplier(client *http.Client, goos string) *Applier {
	if goos == "" {
		goos = runtime.GOOS
	}
	return &Applier{
		downloader: NewDownloader(client),
		goos:       goos,
	}
}

// Download fetches the release archive and checksums.txt into the version run dir.
// Returns the local archive path.
func (a *Applier) Download(ctx context.Context, rel ReleaseInfo, checksumsURL string) (archivePath string, err error) {
	dir, err := paths.UpdateRunDir(rel.Version)
	if err != nil {
		return "", err
	}

	archivePath, err = a.downloader.Download(ctx, rel.DownloadURL, dir, rel.AssetName)
	if err != nil {
		return "", err
	}

	if checksumsURL != "" {
		if _, err := a.downloader.Download(ctx, checksumsURL, dir, "checksums.txt"); err != nil {
			return "", fmt.Errorf("download checksums: %w", err)
		}
	}
	return archivePath, nil
}

// Verify checks archivePath against checksumsPath on disk.
func (a *Applier) Verify(archivePath, checksumsPath string) error {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}
	if err := VerifyArchive(archivePath, data); err != nil {
		_ = os.Remove(archivePath)
		return err
	}
	return nil
}

// Apply downloads (unless skipped), verifies, extracts, backs up, and atomically replaces the binary.
func (a *Applier) Apply(ctx context.Context, rel ReleaseInfo, opts ApplyOptions) error {
	dir, err := paths.UpdateRunDir(rel.Version)
	if err != nil {
		return err
	}

	archivePath := opts.ArchivePath
	if archivePath == "" && !opts.SkipDownload {
		archivePath, err = a.Download(ctx, rel, opts.ChecksumsURL)
		if err != nil {
			return err
		}
	}
	if archivePath == "" {
		archivePath = filepath.Join(dir, rel.AssetName)
	}

	checksumsPath := opts.ChecksumsPath
	if checksumsPath == "" {
		checksumsPath = filepath.Join(dir, "checksums.txt")
	}
	if err := a.Verify(archivePath, checksumsPath); err != nil {
		return err
	}

	stagingDir := filepath.Join(dir, "staging")
	if err := os.RemoveAll(stagingDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(stagingDir)

	binaryName := expectedBinaryName(a.goos)
	extracted, err := extractArchive(archivePath, stagingDir, binaryName)
	if err != nil {
		return err
	}

	persistPath := filepath.Join(dir, binaryName)
	if err := copyFile(extracted, persistPath); err != nil {
		return err
	}

	target, err := resolveTarget(opts.TargetPath)
	if err != nil {
		return err
	}
	if err := assertWritable(target); err != nil {
		return err
	}

	if err := backupBinary(target, rel.Version); err != nil {
		return err
	}

	platform := opts.Platform
	if platform == nil {
		platform = NoopPlatformApplier{}
	}
	if err := platform.Stop(ctx); err != nil {
		log.Printf("update: stop services (best-effort): %v", err)
	}

	if err := platform.SwapBinary(ctx, persistPath, target); err != nil {
		_ = platform.Start(ctx)
		return err
	}

	if err := platform.Start(ctx); err != nil {
		return fmt.Errorf("start services: %w", err)
	}

	store, err := NewFileStateStore()
	if err != nil {
		return err
	}
	st, err := store.Load()
	if err != nil {
		return err
	}
	st.DownloadPath = persistPath
	st.LatestKnown = rel.Version
	return store.Save(st)
}

func resolveTarget(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	return filepath.EvalSymlinks(exe)
}

func assertWritable(path string) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("target directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target directory is not a directory: %s", dir)
	}
	// Create a probe file to detect write permission before stopping services.
	probe := filepath.Join(dir, ".sideguard-write-probe")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("target not writable: %w", err)
	}
	_ = f.Close()
	_ = os.Remove(probe)
	return nil
}

func backupBinary(target, version string) error {
	backups, err := paths.BackupsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(backups, 0o755); err != nil {
		return err
	}
	ts := time.Now().UTC().Format("20060102-150405")
	name := fmt.Sprintf("sideguard-%s-%s", version, ts)
	dest := filepath.Join(backups, name)
	return copyFile(target, dest)
}

func expectedBinaryName(goos string) string {
	if goos == "windows" {
		return "sideguard.exe"
	}
	return "sideguard"
}

func extractArchive(archivePath, destDir, binaryName string) (string, error) {
	if strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return extractZip(archivePath, destDir, binaryName)
	}
	return extractTarGz(archivePath, destDir, binaryName)
}

func extractTarGz(archivePath, destDir, binaryName string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		if strings.Contains(hdr.Name, "..") {
			return "", fmt.Errorf("path traversal in archive: %s", hdr.Name)
		}
		out := filepath.Join(destDir, binaryName)
		if err := writeExtracted(out, tr); err != nil {
			return "", err
		}
		return out, nil
	}
	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}

func extractZip(archivePath, destDir, binaryName string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) != binaryName {
			continue
		}
		if strings.Contains(f.Name, "..") {
			return "", fmt.Errorf("path traversal in archive: %s", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		out := filepath.Join(destDir, binaryName)
		err = writeExtracted(out, rc)
		rc.Close()
		if err != nil {
			return "", err
		}
		return out, nil
	}
	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}

func writeExtracted(path string, r io.Reader) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return err
	}
	return f.Close()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode()&0o777)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
