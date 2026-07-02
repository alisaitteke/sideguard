//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

func shutdownSignals() []os.Signal {
	return []os.Signal{syscall.SIGTERM, syscall.SIGINT}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func terminateProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
