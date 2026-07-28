//go:build windows

package updater

import (
	"strconv"
	"syscall"
	"time"

	core "dappco.re/go"
)

// spawnWatcherImpl spawns a background process that watches for the current
// process to exit, then restarts the binary with --version to confirm the
// update. It is reached through the spawnWatcher var so tests can stub it.
func spawnWatcherImpl() core.Result {
	args := core.Args()
	if len(args) == 0 || args[0] == "" {
		return core.Fail(core.E("spawnWatcher", "missing executable path", nil))
	}
	executable := args[0]

	pid := core.Getpid()

	// Spawn: core update --watch-pid=<pid>
	// StartProcess returns (pid, handle, err) — three values. The unix file
	// next door calls ForkExec, which returns two, and this file had never
	// been compiled, so the mismatch sat here unnoticed.
	_, _, err := syscall.StartProcess(executable, []string{executable, "update", "--watch-pid", strconv.Itoa(pid)}, &syscall.ProcAttr{
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
	_, _, err := syscall.StartProcess(executable, []string{executable, "--version"}, &syscall.ProcAttr{
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
	if pid <= 0 {
		return false
	}
	// SYNCHRONIZE is what makes the handle waitable; QUERY_INFORMATION keeps a
	// denied pid distinguishable from an absent one.
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE|syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = syscall.CloseHandle(handle) }()

	// A handle can still be opened to a process that has exited but not yet
	// been reaped, so opening it is not the answer. Waiting zero milliseconds
	// is: WAIT_TIMEOUT means still running, WAIT_OBJECT_0 means it has exited.
	// GetExitCodeProcess is the usual alternative and cannot tell a running
	// process from one that exited with STILL_ACTIVE (259).
	state, err := syscall.WaitForSingleObject(handle, 0)
	if err != nil {
		return false
	}
	return state == uint32(syscall.WAIT_TIMEOUT)
}
