//go:build !windows

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
	_, err := syscall.ForkExec(executable, []string{executable, "update", "--watch-pid", strconv.Itoa(pid)}, &syscall.ProcAttr{
		Env:   core.Environ(),
		Files: []uintptr{0, 1, 2},
		Sys:   &syscall.SysProcAttr{Setpgid: true},
	})
	return core.ResultOf(nil, err)
}

// watchAndRestart waits for the given PID to exit, then restarts the binary.
func watchAndRestart(pid int) core.Result {
	// Wait for the parent process to die
	for isProcessRunning(pid) {

		time.Sleep(100 * time.Millisecond)
	}

	// Small delay to ensure file handle is released
	time.Sleep(200 * time.Millisecond)

	// Get executable path
	args := core.Args()
	if len(args) == 0 || args[0] == "" {
		return core.Fail(core.E("watchAndRestart", "missing executable path", nil))
	}
	executable := args[0]

	// Use exec to replace this process
	return core.ResultOf(nil, syscall.Exec(executable, []string{executable, "--version"}, core.Environ()))
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	// On Unix, FindProcess always succeeds, so we need to send signal 0
	// to check if the process actually exists
	return syscall.Kill(pid, syscall.Signal(0)) == nil
}
