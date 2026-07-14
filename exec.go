package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/joho/godotenv"
)

// runExec executes a specific command in the directory of a matching task
func runExec(taskQuery string, cmdArgs []string) {
	_ = godotenv.Load()

	config := mustLoadConfig()

	var targetDir string
	var resolvedTaskName string
	found := false

	if taskQuery == "." {
		targetDir = "."
		resolvedTaskName = "."
		found = true
	}

	if !found {
		resolvedTaskName, targetDir, found = resolveExecTarget(config, taskQuery)
	}

	if !found {
		fmt.Printf("Error: Directory '%s' not found in bnm.json.\n", taskQuery)
		if s := closestMatch(taskQuery, sortedKeys(config.Directories)); s != "" {
			fmt.Printf("Did you mean '%s'?\n", s)
		}
		if len(config.Directories) > 0 {
			fmt.Printf("Available directories: %s\n", strings.Join(sortedKeys(config.Directories), ", "))
		}
		os.Exit(1)
	}

	commandStr := strings.Join(cmdArgs, " ")

	task := Task{
		Name:    resolvedTaskName,
		Dir:     targetDir,
		Command: Command(commandStr),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n[bnm] Received termination signal. Stopping process...")
		cancel()
	}()

	sharedEnv := buildEnv(config)

	fmt.Printf("[bnm] Executing '%s' in directory '%s' (Target: %s)...\n", commandStr, targetDir, resolvedTaskName)
	if err := runProcess(ctx, task, taskEnv(sharedEnv, task)); err != nil {
		os.Exit(exitCodeOf(err))
	}
}

// resolveExecTarget finds the directory a query refers to. Matching is
// staged — keys, then paths, then aliases, then directories referenced only
// by script tasks — matching the priority of script filters, so a name or
// path always wins over another directory's alias.
func resolveExecTarget(config *Config, query string) (name, dir string, found bool) {
	cleanQuery := strings.TrimPrefix(query, "./")
	matchers := []func(key string, d Directory) bool{
		func(key string, d Directory) bool { return strings.EqualFold(key, cleanQuery) },
		func(key string, d Directory) bool {
			return strings.EqualFold(strings.TrimPrefix(d.Path, "./"), cleanQuery)
		},
		func(key string, d Directory) bool { return d.Alias != "" && strings.EqualFold(d.Alias, cleanQuery) },
	}
	for _, match := range matchers {
		for _, key := range sortedKeys(config.Directories) {
			if d := config.Directories[key]; match(key, d) {
				return key, d.Path, true
			}
		}
	}

	// Fall back to directories referenced only by script tasks
	for _, scriptGroup := range config.Scripts {
		for _, task := range scriptGroup.Tasks {
			actualDir := task.Dir
			if mappedDir, exists := config.Directories[task.Dir]; exists {
				actualDir = mappedDir.Path
			}
			if strings.EqualFold(strings.TrimPrefix(actualDir, "./"), cleanQuery) {
				return task.Dir, actualDir, true
			}
		}
	}
	return "", "", false
}

// runExecAll executes a command in every configured directory, sequentially
// in sorted key order. Failures don't stop the remaining directories, but
// any failure makes bnm exit non-zero.
func runExecAll(cmdArgs []string) {
	_ = godotenv.Load()

	config := mustLoadConfig()

	if len(config.Directories) == 0 {
		fmt.Println("Error: No directories are configured in bnm.json.")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var interrupted atomic.Bool
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		interrupted.Store(true)
		fmt.Println("\n[bnm] Received termination signal. Stopping process...")
		cancel()
	}()

	sharedEnv := buildEnv(config)
	commandStr := strings.Join(cmdArgs, " ")

	fmt.Printf("[bnm] Executing '%s' in all directories...\n", commandStr)

	var failedDirs []string
	failedDirs = runExecAllProcesses(ctx, config, sharedEnv, func(ctx context.Context, task Task, env []string) error {
		return runProcess(ctx, task, env)
	}, commandStr)

	if interrupted.Load() {
		os.Exit(130)
	}
	if len(failedDirs) > 0 {
		fmt.Printf("[bnm] Command failed in: %s\n", strings.Join(failedDirs, ", "))
		os.Exit(1)
	}
}

func runExecAllProcesses(
	ctx context.Context,
	config *Config,
	sharedEnv []string,
	run func(context.Context, Task, []string) error,
	commandStr string,
) []string {
	var failedDirs []string
	for _, key := range sortedKeys(config.Directories) {
		if ctx.Err() != nil {
			break
		}
		task := Task{
			Name:    key,
			Dir:     config.Directories[key].Path,
			Command: Command(commandStr),
		}
		if err := run(ctx, task, taskEnv(sharedEnv, task)); err != nil && ctx.Err() == nil {
			failedDirs = append(failedDirs, key)
		}
	}
	return failedDirs
}
