// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const downloadTimeout = 60 * time.Second

// Downloader writes release assets to disk with resume support via a .part file.
type Downloader struct {
	httpClient  *http.Client
	validateURL bool
}

// NewDownloader creates a downloader with the given HTTP client or a secure 60s default.
func NewDownloader(client *http.Client) *Downloader {
	validateURL := client == nil
	if client == nil {
		client = newHTTPClient(downloadTimeout)
	}
	return &Downloader{httpClient: client, validateURL: validateURL}
}

// Download fetches url into destDir/name, resuming from an existing .part file when sizes match.
func (d *Downloader) Download(ctx context.Context, rawURL, destDir, name string) (string, error) {
	if d.validateURL {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return "", fmt.Errorf("parse download URL: %w", err)
		}
		if err := validateHTTPSURL(parsed); err != nil {
			return "", err
		}
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create download dir: %w", err)
	}

	finalPath := filepath.Join(destDir, name)
	partPath := finalPath + ".part"

	if info, err := os.Stat(finalPath); err == nil && info.Size() > 0 {
		return finalPath, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	if strings.Contains(rawURL, "api.github.com") && strings.Contains(rawURL, "/releases/assets/") {
		req.Header.Set("Accept", "application/octet-stream")
	}

	var offset int64
	if info, err := os.Stat(partPath); err == nil {
		offset = info.Size()
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", name, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		offset = 0
	case http.StatusPartialContent:
		// resume
	case http.StatusRequestedRangeNotSatisfiable:
		_ = os.Remove(partPath)
		return d.Download(ctx, rawURL, destDir, name)
	default:
		return "", fmt.Errorf("download %s: status %d", name, resp.StatusCode)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if offset == 0 {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}

	f, err := os.OpenFile(partPath, flags, 0o644)
	if err != nil {
		return "", fmt.Errorf("open part file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("write %s: %w", name, err)
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	if err := os.Rename(partPath, finalPath); err != nil {
		return "", fmt.Errorf("finalize download: %w", err)
	}
	return finalPath, nil
}
