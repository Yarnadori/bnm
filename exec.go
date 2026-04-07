package main

import (
	"context"
	"encoding/json"
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

	configFile := "bnm.json"
	file, err := os.Open(configFile)
	if err != nil {
		fmt.Printf("Error: %s not found. Please initialize the project with 'bnm init'.\n", configFile)
		os.Exit(1)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		fmt.Printf("Error: Failed to parse %s: %v\n", configFile, err)
		os.Exit(1)
	}

	var targetDir string
	var resolvedTaskName string
	found := false

	isShorthand := strings.HasPrefix(taskQuery, "-")

outer:
	for _, tasks := range config.Scripts {
		for _, task := range tasks {
			if isShorthand {
				searchStr := strings.TrimPrefix(taskQuery, "-")
				if task.Alias != "" && strings.EqualFold(task.Alias, searchStr) {
					targetDir = task.Dir
					resolvedTaskName = task.Name
					found = true
					break outer
				}
			} else {
				cleanDir := strings.TrimPrefix(task.Dir, "./")
				cleanQuery := strings.TrimPrefix(taskQuery, "./")

				if strings.EqualFold(cleanDir, cleanQuery) {
					targetDir = task.Dir
					resolvedTaskName = task.Name
					found = true
					break outer
				}
			}
		}
	}

	if !found {
		if isShorthand {
			fmt.Printf("Error: Task with alias '%s' not found in bnm.json.\n", strings.TrimPrefix(taskQuery, "-"))
		} else {
			fmt.Printf("Error: Task with directory '%s' not found in bnm.json.\n", taskQuery)
		}
		os.Exit(1)
	}

	commandStr := strings.Join(cmdArgs, " ")

	task := Task{
		Name:    fmt.Sprintf("exec: %s", resolvedTaskName),
		Dir:     targetDir,
		Command: commandStr,
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

	sharedEnv := os.Environ()

	fmt.Printf("[bnm] Executing '%s' in directory '%s' (Task: %s)...\n", commandStr, targetDir, resolvedTaskName)
	runProcess(ctx, task, sharedEnv)
}
