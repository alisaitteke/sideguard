// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

// Package daemon orchestrates the SideGuard background service lifecycle:
// pid file, Unix socket + HTTP API, SQLite queue, and LaunchAgent wiring.
// See docs/plans/2026-07-01-0127-sideguard-foundation/ (vgf-phase-2.0-daemon-core.md).
package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/alisaitteke/sideguard/internal/api"
	"github.com/alisaitteke/sideguard/internal/config"
	"github.com/alisaitteke/sideguard/internal/paths"
	"github.com/alisaitteke/sideguard/internal/store"
)

const healthWaitTimeout = 5 * time.Second

// Start forks a background daemon process if not already running.
func Start(version string) error {
	if running, _ := IsRunning(); running {
		return fmt.Errorf("daemon already running")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	cmd := exec.Command(exe, "daemon", "run")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = detachSysProcAttr()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon process: %w", err)
	}

	return waitForHealthy(healthWaitTimeout)
}

// Run starts the daemon in the foreground (used by daemon run and LaunchAgent).
func Run(version string) error {
	if running, pid := IsRunning(); running && pid != os.Getpid() {
		return fmt.Errorf("daemon already running (pid %d)", pid)
	}

	runDir, err := paths.RunDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return fmt.Errorf("create run dir: %w", err)
	}

	pidPath, err := paths.PIDPath()
	if err != nil {
		return err
	}
	if err := writePID(pidPath, os.Getpid()); err != nil {
		return err
	}
	defer removePID(pidPath)

	dbPath, err := paths.AuditDBPath()
	if err != nil {
		return err
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	if histCfg, err := config.LoadHistory(); err != nil {
		log.Printf("sideguard history: load retention config: %v", err)
	} else if err := st.PruneEvents(histCfg.RetentionDays, histCfg.MaxEvents); err != nil {
		log.Printf("sideguard history: prune on start: %v", err)
	}

	socketPath, err := paths.SocketPath()
	if err != nil {
		return err
	}

	srv := api.NewServer(version, st)
	if err := srv.Listen(socketPath); err != nil {
		return err
	}

	sweeperCtx, sweeperCancel := context.WithCancel(context.Background())
	defer sweeperCancel()
	go st.StartTimeoutSweeper(sweeperCtx)

	if err := srv.Serve(); err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, shutdownSignals()...)
	<-sigCh

	sweeperCancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return srv.Shutdown(shutdownCtx)
}

// Restart stops a running daemon (if any) and starts a fresh process with the
// current binary. No-op when the daemon is not running.
func Restart(version string) error {
	if running, _ := IsRunning(); !running {
		return nil
	}
	if err := Stop(); err != nil {
		return err
	}
	return Start(version)
}

// StopBestEffort stops the daemon when running and ignores not-running errors.
// Used by update apply before swapping the binary (vgu-phase-5.0-update-platform.md).
func StopBestEffort() error {
	if err := Stop(); err != nil {
		if strings.Contains(err.Error(), "not running") {
			return nil
		}
		return err
	}
	return nil
}

// Stop sends SIGTERM to the running daemon.
func Stop() error {
	pidPath, err := paths.PIDPath()
	if err != nil {
		return err
	}
	pid, err := readPID(pidPath)
	if err != nil {
		return fmt.Errorf("daemon is not running: %w", err)
	}
	if !processAlive(pid) {
		removePID(pidPath)
		return fmt.Errorf("daemon is not running (stale pid file)")
	}

	if err := terminateProcess(pid); err != nil {
		return fmt.Errorf("signal daemon: %w", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			removePID(pidPath)
			socketPath, _ := paths.SocketPath()
			_ = os.Remove(socketPath)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("daemon did not stop within timeout")
}

// Status returns a human-readable daemon status line.
func Status() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := api.NewClient()
	health, err := client.Health(ctx)
	if err != nil {
		return "", err
	}

	pending, err := client.ListPending(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("daemon running (version %s, %d pending)", health.Version, len(pending)), nil
}

// IsRunning reports whether the daemon health endpoint responds.
func IsRunning() (bool, int) {
	pidPath, err := paths.PIDPath()
	if err != nil {
		return false, 0
	}
	pid, err := readPID(pidPath)
	if err != nil || !processAlive(pid) {
		return false, 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := api.Ping(ctx); err != nil {
		return false, pid
	}
	return true, pid
}

func waitForHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := api.Ping(ctx)
		cancel()
		if err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("daemon failed to become healthy within %s", timeout)
}

// EnsureRunDir creates ~/.sideguard/run with correct permissions.
func EnsureRunDir() error {
	dir, err := paths.RunDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(dir), 0o700)
}
