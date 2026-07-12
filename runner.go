package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

type taskResult struct {
	Name     string
	Status   string
	Duration time.Duration
}

const (
	statusOK       = "ok"
	statusFailed   = "failed"
	statusSkipped  = "skipped"
	statusCanceled = "canceled"
)

func runScript(targetScript string, filters []string, extraArgs []string) {
	_ = godotenv.Load()

	config := mustLoadConfig()

	if _, exists := config.Scripts[targetScript]; !exists {
		fmt.Printf("Error: Script '%s' is not defined.\n", targetScript)
		if s := closestMatch(targetScript, sortedKeys(config.Scripts)); s != "" {
			fmt.Printf("Did you mean '%s'?\n", s)
		}
		printAvailableScripts(config)
		os.Exit(1)
	}

	order := executionOrder(config, targetScript)

	// Resolve every script's task list up front so filter errors abort
	// before any dependency has run.
	tasksByScript := map[string][]Task{}
	for _, name := range order {
		tasks := config.Scripts[name].Tasks
		if name == targetScript {
			var err error
			tasks, err = filterTasks(config, tasks, filters)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		}
		resolved := make([]Task, 0, len(tasks))
		for _, task := range tasks {
			t := task
			resolveTask(&t, config)
			// Extra args only apply to the requested script, not dependencies
			if name == targetScript && len(extraArgs) > 0 && strings.TrimSpace(t.Command.String()) != "" {
				t.Command = Command(t.Command.String() + " " + shellJoin(extraArgs))
			}
			resolved = append(resolved, t)
		}
		tasksByScript[name] = resolved
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

	var results []taskResult
	failed := false
	for i, name := range order {
		group := config.Scripts[name]
		mode := group.Mode
		if mode == "" {
			mode = "parallel"
		}

		if name == targetScript {
			fmt.Printf("[bnm] Starting script '%s' (Mode: %s)...\n", name, mode)
		} else {
			fmt.Printf("[bnm] Running dependency '%s' (Mode: %s)...\n", name, mode)
		}

		res, ok := runGroup(ctx, mode, group.MaxParallel, tasksByScript[name], sharedEnv)
		results = append(results, res...)
		if !ok {
			failed = true
			if ctx.Err() == nil && name != targetScript {
				fmt.Printf("[bnm] Dependency '%s' failed. Skipping remaining scripts.\n", name)
			}
			for _, rest := range order[i+1:] {
				for _, t := range tasksByScript[rest] {
					results = append(results, taskResult{Name: t.Name, Status: statusSkipped})
				}
			}
			break
		}
		if ctx.Err() != nil {
			break
		}
	}

	fmt.Println("[bnm] All tasks have finished.")
	printSummary(results)

	if interrupted.Load() {
		os.Exit(130)
	}
	if failed {
		os.Exit(1)
	}
}

// executionOrder returns the scripts to run for target: dependencies first
// (depth-first, deduplicated), target last. Cycles and unknown references
// are already rejected by loadConfig.
func executionOrder(config *Config, target string) []string {
	var order []string
	seen := map[string]bool{}
	var visit func(name string)
	visit = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		for _, dep := range config.Scripts[name].DependsOn {
			visit(dep)
		}
		order = append(order, name)
	}
	visit(target)
	return order
}

// resolveTask fills in the display name and turns a directory key into its
// configured path.
func resolveTask(t *Task, config *Config) {
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
	} else {
		t.Name = t.Dir
	}
}

// filterTasks returns the tasks matching any of the given directory filters.
// A filter is an alias prefixed with '-', a directory key, or a path.
// Every filter must match at least one task.
func filterTasks(config *Config, tasks []Task, filters []string) ([]Task, error) {
	if len(filters) == 0 {
		return tasks, nil
	}
	used := make([]bool, len(filters))
	var matched []Task
	for _, task := range tasks {
		for i, f := range filters {
			if taskMatchesFilter(config, task, f) {
				used[i] = true
				matched = append(matched, task)
				break
			}
		}
	}
	for i, f := range filters {
		if !used[i] {
			return nil, fmt.Errorf("no tasks in this script match directory filter '%s'", f)
		}
	}
	return matched, nil
}

func taskMatchesFilter(config *Config, task Task, filter string) bool {
	dirKey := task.Dir
	dirPath := task.Dir
	alias := ""
	if d, exists := config.Directories[task.Dir]; exists {
		dirPath = d.Path
		alias = d.Alias
	}
	if aliasFilter, ok := strings.CutPrefix(filter, "-"); ok {
		return alias != "" && strings.EqualFold(alias, aliasFilter)
	}
	if dirKey == "" {
		dirKey, dirPath = ".", "."
	}
	clean := func(s string) string { return strings.TrimPrefix(s, "./") }
	return strings.EqualFold(dirKey, filter) || strings.EqualFold(clean(dirPath), clean(filter))
}

// runGroup runs one script group and reports per-task results plus overall
// success. Tasks that never ran remain "skipped".
func runGroup(ctx context.Context, mode string, maxParallel int, tasks []Task, sharedEnv []string) ([]taskResult, bool) {
	results := make([]taskResult, len(tasks))
	for i, t := range tasks {
		results[i] = taskResult{Name: t.Name, Status: statusSkipped}
	}

	runOne := func(i int, t Task) bool {
		start := time.Now()
		err := runProcess(ctx, t, taskEnv(sharedEnv, t))
		results[i].Duration = time.Since(start)
		switch {
		case err == nil:
			results[i].Status = statusOK
			return true
		case ctx.Err() != nil:
			results[i].Status = statusCanceled
			return false
		default:
			results[i].Status = statusFailed
			return false
		}
	}

	ok := true
	if mode == "sequential" {
		for i, t := range tasks {
			if ctx.Err() != nil {
				break
			}
			if !runOne(i, t) {
				ok = false
				if ctx.Err() == nil {
					fmt.Printf("[bnm] Task in '%s' failed. Skipping remaining tasks.\n", t.Name)
				}
				break
			}
		}
	} else {
		var wg sync.WaitGroup
		var anyFailed atomic.Bool
		var sem chan struct{}
		if maxParallel > 0 {
			sem = make(chan struct{}, maxParallel)
		}
		for i, task := range tasks {
			wg.Add(1)
			go func(i int, t Task) {
				defer wg.Done()
				if sem != nil {
					select {
					case sem <- struct{}{}:
						defer func() { <-sem }()
					case <-ctx.Done():
						results[i].Status = statusCanceled
						anyFailed.Store(true)
						return
					}
				}
				if !runOne(i, t) {
					anyFailed.Store(true)
				}
			}(i, task)
		}
		wg.Wait()
		ok = !anyFailed.Load()
	}
	return results, ok
}

// taskEnv extends the shared environment with the task directory's .env file
// and the task's own env entries. Later entries win over earlier ones.
func taskEnv(shared []string, task Task) []string {
	env := append([]string(nil), shared...)
	// The project root .env is already loaded into the process environment
	if task.Dir != "" && task.Dir != "." {
		if m, err := godotenv.Read(filepath.Join(task.Dir, ".env")); err == nil {
			for _, k := range sortedKeys(m) {
				env = append(env, k+"="+m[k])
			}
		}
	}
	for _, k := range sortedKeys(task.Env) {
		env = append(env, k+"="+task.Env[k])
	}
	return env
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

// shellJoin joins pass-through arguments into a command-line fragment,
// quoting arguments that contain whitespace or shell metacharacters.
func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shellQuoteArg(a)
	}
	return strings.Join(quoted, " ")
}

func shellQuoteArg(a string) string {
	if runtime.GOOS == "windows" {
		return shellQuoteWindows(a)
	}
	return shellQuotePOSIX(a)
}

func shellQuotePOSIX(a string) string {
	return "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
}

func shellQuoteWindows(a string) string {
	// cmd.exe expands percent-delimited environment variables even inside
	// quotes. Doubling percent signs preserves them as argument data.
	a = strings.ReplaceAll(a, "%", "%%")
	return `"` + strings.ReplaceAll(a, `"`, `""`) + `"`
}

func printSummary(results []taskResult) {
	if len(results) <= 1 {
		return
	}
	fmt.Println("[bnm] Summary:")
	for _, r := range results {
		mark, color := summaryMark(r.Status)
		reset := ""
		if color != "" {
			reset = colorReset
		}
		line := fmt.Sprintf("  %s%s %-12s %-9s%s", color, mark, r.Name, r.Status, reset)
		if r.Status == statusOK || r.Status == statusFailed {
			line += " " + formatDuration(r.Duration)
		}
		fmt.Println(line)
	}
}

func summaryMark(status string) (string, string) {
	mark, color := "-", ""
	switch status {
	case statusOK:
		mark, color = "✔", "\033[32m"
	case statusFailed:
		mark, color = "✖", "\033[31m"
	case statusCanceled:
		mark, color = "✖", "\033[33m"
	}
	if !colorEnabled {
		color = ""
	}
	return mark, color
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(10 * time.Millisecond).String()
}
