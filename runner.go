package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/joho/godotenv"
)

func runScript(targetScript string) {
	_ = godotenv.Load()

	config := mustLoadConfig()

	scriptGroup, exists := config.Scripts[targetScript]
	if !exists {
		fmt.Printf("Error: Script '%s' is not defined.\n", targetScript)
		printAvailableScripts(config)
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
		fmt.Println("\n[bnm] Received termination signal. Stopping all processes...")
		cancel()
	}()

	sharedEnv := buildEnv(config)
	mode := scriptGroup.Mode
	if mode == "" {
		mode = "parallel"
	}

	fmt.Printf("[bnm] Starting script '%s' (Mode: %s)...\n", targetScript, mode)
	resolveName := func(t *Task) {
		if t.Dir == "" {
			t.Dir = "."
			if config.Name != "" {
				t.Name = config.Name
			} else {
				t.Name = "."
			}
		} else if resolvedDir, exists := config.Directories[t.Dir]; exists {
			t.Name = t.Dir
			t.Dir = resolvedDir.Path
		} else if t.Dir == "." {
			t.Name = "."
		} else {
			t.Name = t.Dir
		}
	}

	failed := false
	if mode == "sequential" {
		for _, task := range scriptGroup.Tasks {
			t := task
			resolveName(&t)
			if err := runProcess(ctx, t, sharedEnv); err != nil {
				failed = true
				if ctx.Err() == nil {
					fmt.Printf("[bnm] Task in '%s' failed. Skipping remaining tasks.\n", t.Name)
				}
				break
			}
			if ctx.Err() != nil {
				break
			}
		}
	} else {
		var wg sync.WaitGroup
		var anyFailed atomic.Bool
		for _, task := range scriptGroup.Tasks {
			t := task
			resolveName(&t)

			wg.Add(1)
			go func(t Task) {
				defer wg.Done()
				if err := runProcess(ctx, t, sharedEnv); err != nil {
					anyFailed.Store(true)
				}
			}(t)
		}
		wg.Wait()
		failed = anyFailed.Load()
	}

	fmt.Println("[bnm] All tasks have finished.")

	if interrupted.Load() {
		os.Exit(130)
	}
	if failed {
		os.Exit(1)
	}
}

// buildEnv returns the environment for task processes, including
// PROJECT_NAME / PROJECT_VERSION from bnm.json.
func buildEnv(config *Config) []string {
	env := os.Environ()
	if config.Name != "" {
		env = append(env, "PROJECT_NAME="+config.Name)
	}
	if config.Version != "" {
		env = append(env, "PROJECT_VERSION="+config.Version)
	}
	return env
}
