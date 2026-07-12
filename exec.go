package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
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

	isShorthand := strings.HasPrefix(taskQuery, "-")

	if isShorthand {
		searchStr := strings.TrimPrefix(taskQuery, "-")
		for key, dir := range config.Directories {
			if strings.EqualFold(dir.Alias, searchStr) {
				targetDir = dir.Path
				resolvedTaskName = key
				found = true
				break
			}
		}
	} else if !found {
		cleanQuery := strings.TrimPrefix(taskQuery, "./")

		// Match a directory entry by key or by path
		for key, dir := range config.Directories {
			if strings.EqualFold(key, cleanQuery) || strings.EqualFold(strings.TrimPrefix(dir.Path, "./"), cleanQuery) {
				targetDir = dir.Path
				resolvedTaskName = key
				found = true
				break
			}
		}

		// Fall back to directories referenced only by script tasks
		if !found {
		outer:
			for _, scriptGroup := range config.Scripts {
				for _, task := range scriptGroup.Tasks {
					actualDir := task.Dir
					if mappedDir, exists := config.Directories[task.Dir]; exists {
						actualDir = mappedDir.Path
					}
					if strings.EqualFold(strings.TrimPrefix(actualDir, "./"), cleanQuery) {
						targetDir = actualDir
						resolvedTaskName = task.Dir
						found = true
						break outer
					}
				}
			}
		}
	}

	if !found {
		if isShorthand {
			fmt.Printf("Error: Directory alias '%s' not found in bnm.json.\n", strings.TrimPrefix(taskQuery, "-"))
		} else {
			fmt.Printf("Error: Directory '%s' not found in bnm.json.\n", taskQuery)
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
	if err := runProcess(ctx, task, sharedEnv); err != nil {
		os.Exit(exitCodeOf(err))
	}
}
