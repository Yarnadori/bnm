package main

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPrefixLoggerLongLines(t *testing.T) {
	// Lines longer than bufio.Scanner's 64KB default must not be dropped
	longLine := strings.Repeat("a", 100*1024)
	var out strings.Builder
	prefixLogger("TEST", strings.NewReader(longLine+"\nshort\n"), &out)

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), out.String())
	}
	if !strings.HasPrefix(lines[0], "[TEST] ") || len(lines[0]) != len("[TEST] ")+100*1024 {
		t.Errorf("long line was truncated or mangled (len=%d)", len(lines[0]))
	}
	if lines[1] != "[TEST] short" {
		t.Errorf("got %q, want %q", lines[1], "[TEST] short")
	}
}

func TestRunProcessSuccess(t *testing.T) {
	task := Task{Name: "T", Command: "exit 0"}
	if err := runProcess(context.Background(), task, os.Environ()); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestRunProcessExitCode(t *testing.T) {
	task := Task{Name: "T", Command: "exit 3"}
	err := runProcess(context.Background(), task, os.Environ())
	if err == nil {
		t.Fatal("expected error for failing command")
	}
	if code := exitCodeOf(err); code != 3 {
		t.Errorf("exit code: got %d, want 3", code)
	}
}

func TestRunProcessEmptyCommand(t *testing.T) {
	task := Task{Name: "T", Command: ""}
	if err := runProcess(context.Background(), task, os.Environ()); err == nil {
		t.Error("expected error for empty command")
	}
}

func TestRunProcessCancelKillsProcessTree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-group test is unix-only")
	}

	ctx, cancel := context.WithCancel(context.Background())
	task := Task{Name: "T", Command: "sleep 30"}

	done := make(chan error, 1)
	go func() { done <- runProcess(ctx, task, os.Environ()) }()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// terminated promptly — success
	case <-time.After(5 * time.Second):
		t.Fatal("process was not terminated after context cancel")
	}
}
