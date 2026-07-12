//go:build !windows

package main

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// newTaskCommand builds a shell command that runs in its own process group,
// so that cancellation terminates the whole process tree (e.g. dev servers
// spawned by npm), not just the shell.
func newTaskCommand(ctx context.Context, cmdStr string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// Negative PID signals the whole process group
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	// If the process ignores SIGTERM, force-kill it after a grace period
	cmd.WaitDelay = 10 * time.Second
	return cmd
}
