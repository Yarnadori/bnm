package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

func runProcess(ctx context.Context, task Task, env []string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", task.Command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", task.Command)
	}

	if task.Dir != "" {
		cmd.Dir = task.Dir
	}
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("[%s] Error: Failed to get stdout: %v\n", task.Name, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("[%s] Error: Failed to get stderr: %v\n", task.Name, err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("[%s] Startup error: %v\n", task.Name, err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		prefixLogger(task.Name, stdout, os.Stdout)
	}()
	go func() {
		defer wg.Done()
		prefixLogger(task.Name, stderr, os.Stderr)
	}()

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != context.Canceled {
			fmt.Printf("[%s] Exit code error: %v\n", task.Name, err)
		}
	}
}

func prefixLogger(name string, r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Fprintf(w, "[%s] %s\n", name, scanner.Text())
	}
}
