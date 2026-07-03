// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package update

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// ParseChecksums reads checksums.txt lines in sha256sum format (hash + filename).
func ParseChecksums(data []byte) (map[string]string, error) {
	checksums := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		hash, name, ok := parseChecksumLine(line)
		if !ok {
			continue
		}
		checksums[name] = hash
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(checksums) == 0 {
		return nil, fmt.Errorf("no checksum entries found")
	}
	return checksums, nil
}

func parseChecksumLine(line string) (hash, name string, ok bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", false
	}
	hash = fields[0]
	name = strings.TrimPrefix(fields[len(fields)-1], "*")
	if len(hash) != 64 {
		return "", "", false
	}
	return hash, name, true
}

// VerifyFileSHA256 compares the file at path with the expected hex digest.
func VerifyFileSHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash %s: %w", path, err)
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch for %s", filepathBase(path))
	}
	return nil
}

func filepathBase(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	if i := strings.LastIndex(path, "\\"); i >= 0 {
		return path[i+1:]
	}
	return path
}

// VerifyArchive checks an archive file against checksums.txt content.
func VerifyArchive(archivePath string, checksumsData []byte) error {
	checksums, err := ParseChecksums(checksumsData)
	if err != nil {
		return err
	}
	name := filepathBase(archivePath)
	expected, ok := checksums[name]
	if !ok {
		return fmt.Errorf("checksum not found for %s", name)
	}
	return VerifyFileSHA256(archivePath, expected)
}
