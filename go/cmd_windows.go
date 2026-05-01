//go:build windows

package updater

import (
	"strconv"
	"syscall"
	"time"

	core "dappco.re/go"
)

// spawnWatcher spawns a background process that watches for the current process
// to exit, then restarts the binary with --version to confirm the update.
func spawnWatcher() core.Result {
	args := core.Args()
	if len(args) == 0 || args[0] == "" {
		return core.Fail(core.E("spawnWatcher", "missing executable path", nil))
	}
	executable := args[0]

	pid := core.Getpid()

	// Spawn: core update --watch-pid=<pid>
	_, err := syscall.StartProcess(executable, []string{executable, "update", "--watch-pid", strconv.Itoa(pid)}, &syscall.ProcAttr{
		Env:   core.Environ(),
		Files: []uintptr{0, 1, 2},
		Sys:   &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP},
	})
	return core.ResultOf(nil, err)
}

// watchAndRestart waits for the given PID to exit, then restarts the binary.
func watchAndRestart(pid int) core.Result {
	// Wait for the parent process to die
	for {
		if !isProcessRunning(pid) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Small delay to ensure file handle is released
	time.Sleep(500 * time.Millisecond)

	// Get executable path
	args := core.Args()
	if len(args) == 0 || args[0] == "" {
		return core.Fail(core.E("watchAndRestart", "missing executable path", nil))
	}
	executable := args[0]

	// On Windows, spawn new process and exit
	_, err := syscall.StartProcess(executable, []string{executable, "--version"}, &syscall.ProcAttr{
		Env:   core.Environ(),
		Files: []uintptr{0, 1, 2},
	})
	if err != nil {
		return core.Fail(err)
	}

	core.Exit(0)
	return core.Ok(nil)
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	// On Windows, try to open the process with query rights
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(handle)
	return true
}
