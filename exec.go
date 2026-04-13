package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"os/signal"
	"strings"
	"syscall"
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
	} else {
		cleanQuery := strings.TrimPrefix(taskQuery, "./")

	outer:
		for _, scriptGroup := range config.Scripts {
			for _, task := range scriptGroup.Tasks {
				actualDir := task.Dir
				if mappedDir, exists := config.Directories[task.Dir]; exists {
					actualDir = mappedDir.Path
				}

				cleanDir := strings.TrimPrefix(actualDir, "./")
				if strings.EqualFold(cleanDir, cleanQuery) {
					targetDir = actualDir
					resolvedTaskName = task.Dir
					found = true
					break outer
				}
			}
		}
		if !found {
			for key, dir := range config.Directories {
				cleanDir := strings.TrimPrefix(dir.Path, "./")
				if strings.EqualFold(cleanDir, cleanQuery) {
					targetDir = dir.Path
					resolvedTaskName = key
					found = true
					break
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

	sharedEnv := os.Environ()
	if config.Name != "" {
		sharedEnv = append(sharedEnv, "PROJECT_NAME="+config.Name)
	}
	if config.Version != "" {
		sharedEnv = append(sharedEnv, "PROJECT_VERSION="+config.Version)
	}

	fmt.Printf("[bnm] Executing '%s' in directory '%s' (Target: %s)...\n", commandStr, targetDir, resolvedTaskName)
	runProcess(ctx, task, sharedEnv)
}
