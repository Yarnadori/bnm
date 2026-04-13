package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func runScript(targetScript string) {
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

	scriptGroup, exists := config.Scripts[targetScript]
	if !exists {
		fmt.Printf("Error: Script '%s' is not defined.\n", targetScript)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n[bnm] Received termination signal. Stopping all processes...")
		cancel()
	}()

	sharedEnv := os.Environ()
	mode := scriptGroup.Mode
	if mode == "" {
		mode = "parallel"
	}

	fmt.Printf("[bnm] Starting script '%s' (Mode: %s)...\n", targetScript, mode)
	resolveName := func(t *Task) {
		if resolvedDir, exists := config.Directories[t.Dir]; exists {
			t.Name = t.Dir
			t.Dir = resolvedDir.Path
		} else if t.Dir == "." {
			t.Name = "."
		} else {
			t.Name = t.Dir
		}
	}

	if mode == "sequential" {
		for _, task := range scriptGroup.Tasks {
			t := task
			resolveName(&t)
			runProcess(ctx, t, sharedEnv)
			if ctx.Err() != nil {
				break
			}
		}
	} else {
		var wg sync.WaitGroup
		for _, task := range scriptGroup.Tasks {
			t := task
			resolveName(&t)

			wg.Add(1)
			go func(t Task) {
				defer wg.Done()
				runProcess(ctx, t, sharedEnv)
			}(t)
		}
		wg.Wait()
	}

	fmt.Println("[bnm] All tasks have finished.")
}
