//go:build windows

package main

import (
	"context"
	"os/exec"
	"strconv"
	"time"
)

// newTaskCommand builds a shell command whose cancellation terminates the
// whole process tree via taskkill, not just the cmd.exe shell.
func newTaskCommand(ctx context.Context, cmdStr string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd", "/C", cmdStr)
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	}
	cmd.WaitDelay = 10 * time.Second
	return cmd
}
