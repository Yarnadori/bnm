package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// runProcess runs a single task and returns an error if the command
// could not start or exited with a non-zero status.
func runProcess(ctx context.Context, task Task, env []string) error {
	cmdStr := task.Command.String()
	if strings.TrimSpace(cmdStr) == "" {
		err := fmt.Errorf("no command defined for this OS")
		fmt.Printf("[%s] Error: %v\n", task.Name, err)
		return err
	}

	cmd := newTaskCommand(ctx, cmdStr)

	if task.Dir != "" {
		cmd.Dir = task.Dir
	}
	cmd.Env = env

	pre, reset := colorFor(task.Name)
	fmt.Printf("%s[%s]%s $ %s\n", pre, task.Name, reset, cmdStr)

	var logW io.Writer
	if task.LogPath != "" {
		f, err := os.OpenFile(task.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Printf("[%s] Warning: cannot write log file: %v\n", task.Name, err)
		} else {
			defer f.Close()
			logW = &syncWriter{w: f}
			fmt.Fprintf(logW, "$ %s\n", cmdStr)
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("[%s] Error: Failed to get stdout: %v\n", task.Name, err)
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("[%s] Error: Failed to get stderr: %v\n", task.Name, err)
		return err
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("[%s] Startup error: %v\n", task.Name, err)
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		prefixLogger(task.Name, stdout, os.Stdout, logW)
	}()
	go func() {
		defer wg.Done()
		prefixLogger(task.Name, stderr, os.Stderr, logW)
	}()

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == nil {
			fmt.Printf("[%s] Exit code error: %v\n", task.Name, err)
		}
		if logW != nil {
			fmt.Fprintf(logW, "[bnm] %v\n", err)
		}
		return err
	}
	return nil
}

// syncWriter serializes writes from the stdout and stderr loggers so lines
// in a task's log file don't interleave mid-line.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

// exitCodeOf extracts the process exit code from an error returned by
// runProcess, defaulting to 1 for non-exit errors.
func exitCodeOf(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() > 0 {
		return exitErr.ExitCode()
	}
	return 1
}

// prefixLogger copies lines from r to w with a colored name prefix. When
// logW is non-nil, the raw lines are also written there.
func prefixLogger(name string, r io.Reader, w io.Writer, logW io.Writer) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	pre, reset := colorFor(name)
	for scanner.Scan() {
		fmt.Fprintf(w, "%s[%s]%s %s\n", pre, name, reset, scanner.Text())
		if logW != nil {
			fmt.Fprintln(logW, scanner.Text())
		}
	}
	// fs.ErrClosed happens when the pipe is force-closed on shutdown; not worth reporting
	if err := scanner.Err(); err != nil && !errors.Is(err, fs.ErrClosed) {
		fmt.Fprintf(w, "%s[%s]%s [bnm] output stream error: %v\n", pre, name, reset, err)
	}
}
