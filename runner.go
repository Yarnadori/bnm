package main

import (
	"context"
	"encoding/json"
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

type scriptOptions struct {
	Watch   bool
	DryRun  bool
	NoColor bool
	LogDir  string
	Summary string // "" or "text" for the table, "json" for machine-readable output
}

func runScript(targetScript string, filters []string, extraArgs []string, opts scriptOptions) {
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

	if opts.NoColor {
		colorEnabled = false
	}

	if opts.DryRun {
		printPlan(config, order, tasksByScript, targetScript)
		return
	}

	if opts.LogDir != "" {
		if err := assignLogPaths(opts.LogDir, order, tasksByScript); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
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

	if opts.Watch {
		runScriptWatch(ctx, config, order, tasksByScript, targetScript, sharedEnv, opts)
		if interrupted.Load() {
			os.Exit(130)
		}
		return
	}

	failed := runScriptOnce(ctx, config, order, tasksByScript, targetScript, sharedEnv, opts)

	if interrupted.Load() {
		os.Exit(130)
	}
	if failed {
		os.Exit(1)
	}
}

// assignLogPaths gives every task a log file under logDir/<script>/, creating
// the directories and truncating files left over from earlier invocations.
// Task names are deduplicated within a script so no file is shared.
func assignLogPaths(logDir string, order []string, tasksByScript map[string][]Task) error {
	for _, name := range order {
		scriptDir := filepath.Join(logDir, sanitizeLogName(name))
		if len(tasksByScript[name]) > 0 {
			if err := os.MkdirAll(scriptDir, 0o755); err != nil {
				return fmt.Errorf("failed to create log directory: %w", err)
			}
		}
		used := map[string]int{}
		for i := range tasksByScript[name] {
			t := &tasksByScript[name][i]
			base := sanitizeLogName(t.Name)
			used[base]++
			if n := used[base]; n > 1 {
				base = fmt.Sprintf("%s-%d", base, n)
			}
			t.LogPath = filepath.Join(scriptDir, base+".log")
			if err := os.WriteFile(t.LogPath, nil, 0o644); err != nil {
				return fmt.Errorf("failed to create log file: %w", err)
			}
		}
	}
	return nil
}

// sanitizeLogName maps a task or script name to a safe file name.
func sanitizeLogName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	s := strings.Trim(b.String(), "._")
	if s == "" {
		return "root"
	}
	return s
}

// runScriptOnce runs the resolved scripts in order and prints the summary.
// It reports whether any task failed.
func runScriptOnce(ctx context.Context, config *Config, order []string, tasksByScript map[string][]Task, targetScript string, sharedEnv []string, opts scriptOptions) bool {
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
	if opts.Summary == "json" {
		fmt.Println(string(summaryJSON(targetScript, results, failed)))
	} else {
		printSummary(results)
	}
	return failed
}

// summaryJSON renders the run summary as a single-line JSON object.
func summaryJSON(script string, results []taskResult, failed bool) []byte {
	type jsonTask struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		DurationMs int64  `json:"durationMs"`
	}
	out := struct {
		Script string     `json:"script"`
		OK     bool       `json:"ok"`
		Tasks  []jsonTask `json:"tasks"`
	}{Script: script, OK: !failed, Tasks: make([]jsonTask, 0, len(results))}
	for _, r := range results {
		out.Tasks = append(out.Tasks, jsonTask{
			Name:       r.Name,
			Status:     r.Status,
			DurationMs: r.Duration.Milliseconds(),
		})
	}
	b, _ := json.Marshal(out)
	return b
}

// printPlan shows what a script run would execute, without running anything.
func printPlan(config *Config, order []string, tasksByScript map[string][]Task, targetScript string) {
	fmt.Printf("[bnm] Dry run: execution plan for '%s'\n", targetScript)
	for i, name := range order {
		group := config.Scripts[name]
		mode := group.Mode
		if mode == "" {
			mode = "parallel"
		}
		kind := "script"
		if name != targetScript {
			kind = "dependency"
		}
		detail := mode
		if mode == "parallel" && group.MaxParallel > 0 {
			detail += fmt.Sprintf(", maxParallel %d", group.MaxParallel)
		}
		fmt.Printf("%d. %s '%s' (%s)\n", i+1, kind, name, detail)
		for _, t := range tasksByScript[name] {
			attrs := ""
			if t.Timeout > 0 {
				attrs += fmt.Sprintf("  [timeout %s]", t.Timeout)
			}
			if t.Retries > 0 {
				attrs += fmt.Sprintf("  [retries %d]", t.Retries)
			}
			fmt.Printf("     %-12s (%s) $ %s%s\n", t.Name, t.Dir, t.Command, attrs)
		}
	}
	fmt.Println("[bnm] No commands were executed.")
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
		err := runTaskAttempts(ctx, t, taskEnv(sharedEnv, t))
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

// runTaskAttempts runs a task, enforcing its timeout per attempt and
// retrying failed attempts up to task.Retries times.
func runTaskAttempts(ctx context.Context, t Task, env []string) error {
	for attempt := 0; ; attempt++ {
		attemptCtx := ctx
		cancel := context.CancelFunc(func() {})
		if t.Timeout > 0 {
			attemptCtx, cancel = context.WithTimeout(ctx, time.Duration(t.Timeout))
		}
		err := runProcess(attemptCtx, t, env)
		timedOut := attemptCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil
		cancel()
		if err == nil || ctx.Err() != nil {
			return err
		}
		if timedOut {
			fmt.Printf("[%s] Timed out after %s.\n", t.Name, t.Timeout)
		}
		if attempt >= t.Retries {
			return err
		}
		fmt.Printf("[%s] Attempt %d/%d failed. Retrying...\n", t.Name, attempt+1, t.Retries+1)
	}
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
	// cmd.exe expands %VAR% even inside quotes, and cmd /C offers no escape
	// for percent signs (%% doubling only works in batch files; carets are
	// literal inside quotes). Percents pass through unchanged: an undefined
	// %name% survives as-is, which is the least-bad outcome.
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
