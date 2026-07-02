//go:build windows

package daemon

import (
	"os"
	"syscall"
)

const (
	detachedProcess = 0x00000008 // DETACHED_PROCESS — https://learn.microsoft.com/en-us/windows/win32/procthread/process-creation-flags
	stillActive     = 259        // STILL_ACTIVE — https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-getexitcodeprocess
)

func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: detachedProcess,
	}
}

func shutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)

	var code uint32
	if err := syscall.GetExitCodeProcess(handle, &code); err != nil {
		return false
	}
	return code == stillActive
}

func terminateProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
