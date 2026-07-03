// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

package proxy

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Upstream manages the child MCP server process and its STDIO pipes.
type Upstream struct {
	cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
}

// StartUpstream spawns the upstream MCP server with inherited environment.
// The proxy does not inject secrets into the child environment.
func StartUpstream(argv []string) (*Upstream, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("upstream command required")
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("upstream stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("upstream stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("upstream stderr pipe: %w", err)
	}

	go func() {
		_, _ = io.Copy(os.Stderr, stderr)
	}()

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("start upstream %q: %w", argv[0], err)
	}

	return &Upstream{
		cmd:    cmd,
		Stdin:  stdin,
		Stdout: stdout,
	}, nil
}

// Close shuts down upstream pipes and waits for the child process.
func (u *Upstream) Close() error {
	if u == nil {
		return nil
	}
	if u.Stdin != nil {
		_ = u.Stdin.Close()
	}
	if u.Stdout != nil {
		_ = u.Stdout.Close()
	}
	if u.cmd != nil && u.cmd.Process != nil {
		return u.cmd.Wait()
	}
	return nil
}
