package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
)

// runScript executes the tasks for the specified script name
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

	tasks, exists := config.Scripts[targetScript]
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
	var wg sync.WaitGroup

	fmt.Printf("[bnm] Starting script '%s'...\n", targetScript)
	for _, task := range tasks {
		t := task

		t.Name = t.Dir

		if resolvedDir, exists := config.Directories[t.Dir]; exists {
			t.Dir = resolvedDir.Path
		}

		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			runProcess(ctx, t, sharedEnv)
		}(t)
	}

	wg.Wait()
	fmt.Println("[bnm] All tasks have finished.")
}