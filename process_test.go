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
	// A line longer than the read buffer is displayed as multiple prefixed
	// chunks so memory stays bounded, with no byte dropped and the pipe
	// drained to the end. The log writer must receive the original bytes
	// unsplit.
	longLine := strings.Repeat("a", 2*1024*1024)
	input := longLine + "\nshort\n"
	var out, log strings.Builder
	prefixLogger("TEST", strings.NewReader(input), &out, &log)

	if log.String() != input {
		t.Errorf("log content was altered (got %d bytes, want %d)", log.Len(), len(input))
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("got %d lines, want at least 2", len(lines))
	}
	var joined strings.Builder
	for _, line := range lines[:len(lines)-1] {
		content, ok := strings.CutPrefix(line, "[TEST] ")
		if !ok {
			t.Fatalf("line without prefix: %.40q", line)
		}
		joined.WriteString(content)
	}
	if joined.String() != longLine {
		t.Errorf("long line was truncated or mangled (got %d bytes, want %d)", joined.Len(), len(longLine))
	}
	if lines[len(lines)-1] != "[TEST] short" {
		t.Errorf("got %q, want %q", lines[len(lines)-1], "[TEST] short")
	}
}

func TestPrefixLoggerNoTrailingNewline(t *testing.T) {
	var out, log strings.Builder
	prefixLogger("TEST", strings.NewReader("partial"), &out, &log)
	if out.String() != "[TEST] partial\n" {
		t.Errorf("got %q", out.String())
	}
	if log.String() != "partial" {
		t.Errorf("log: got %q, want %q", log.String(), "partial")
	}
}

func TestPrefixLoggerKeepsEmptyLines(t *testing.T) {
	var out, log strings.Builder
	prefixLogger("TEST", strings.NewReader("a\n\nb\n"), &out, &log)
	if out.String() != "[TEST] a\n[TEST] \n[TEST] b\n" {
		t.Errorf("got %q", out.String())
	}
	if log.String() != "a\n\nb\n" {
		t.Errorf("log: got %q", log.String())
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
