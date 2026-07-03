// Copyright (c) 2026 Ali Sait Teke
// SPDX-License-Identifier: MIT

//go:build windows

package tray

import (
	"os/exec"
	"syscall"
)

func setDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
