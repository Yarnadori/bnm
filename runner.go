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
	Dir      string
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
	Watch       bool // --watch; forces watch mode on
	NoWatch     bool // --no-watch; overrides a "watch": true in the config
	DryRun      bool
	NoColor     bool
	LogDir      string
	Summary     string   // "" or "text" for the table, "json" for machine-readable output
	TaskFilters []string // task names from --task / -T; empty means all tasks
}

// watchEnabled reports whether the run should watch: the target script's
// config default, overridden by --watch / --no-watch. Watch settings on
// dependency scripts are ignored.
func watchEnabled(group ScriptGroup, opts scriptOptions) bool {
	if opts.NoWatch {
		return false
	}
	return opts.Watch || group.Watch
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
			all := tasks
			var err error
			tasks, err = filterTasks(config, tasks, filters)
			if err == nil {
				tasks, err = filterTasksByTaskName(config, tasks, all, opts.TaskFilters, targetScript)
			}
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

	if watchEnabled(config.Scripts[targetScript], opts) {
		runScriptWatch(ctx, config, order, tasksByScript, targetScript, sharedEnv, opts)
		if interrupted.Load() {
			os.Exit(130)
		}
		return
	}

	if !opts.NoWatch && anyTaskWatch(tasksByScript[targetScript]) {
		failed := runScriptTaskWatch(ctx, config, order, tasksByScript, targetScript, sharedEnv, opts)
		if interrupted.Load() {
			os.Exit(130)
		}
		if failed {
			os.Exit(1)
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
// Sanitized names can collide (both across scripts and across tasks), so
// uniqueness is enforced on the final paths, not the original names.
func assignLogPaths(logDir string, order []string, tasksByScript map[string][]Task) error {
	usedDirs := map[string]bool{}
	for _, name := range order {
		scriptDir := uniquePath(usedDirs, filepath.Join(logDir, sanitizeLogName(name)), "")
		if len(tasksByScript[name]) > 0 {
			if err := os.MkdirAll(scriptDir, 0o755); err != nil {
				return fmt.Errorf("failed to create log directory: %w", err)
			}
		}
		usedFiles := map[string]bool{}
		for i := range tasksByScript[name] {
			t := &tasksByScript[name][i]
			t.LogPath = uniquePath(usedFiles, filepath.Join(scriptDir, sanitizeLogName(t.Name)), ".log")
			if err := os.WriteFile(t.LogPath, nil, 0o644); err != nil {
				return fmt.Errorf("failed to create log file: %w", err)
			}
		}
	}
	return nil
}

// logPathFold makes log-path deduplication case-insensitive on filesystems
// that are (Windows, default macOS), where A.log and a.log are one file.
var logPathFold = runtime.GOOS == "windows" || runtime.GOOS == "darwin"

// uniquePath returns base+ext, or base-N+ext for the smallest N that is not
// in used yet, and records the result in used.
func uniquePath(used map[string]bool, base, ext string) string {
	key := func(s string) string {
		if logPathFold {
			return strings.ToLower(s)
		}
		return s
	}
	path := base + ext
	for n := 2; used[key(path)]; n++ {
		path = fmt.Sprintf("%s-%d%s", base, n, ext)
	}
	used[key(path)] = true
	return path
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

// anyTaskWatch reports whether any task carries "watch": true.
func anyTaskWatch(tasks []Task) bool {
	for _, t := range tasks {
		if t.Watch {
			return true
		}
	}
	return false
}

// runScriptTaskWatch runs the target script with its tasks marked
// "watch": true under supervision: when files in such a task's directory
// change, only that task is stopped and rerun; everything else keeps
// running. Dependencies run once up front. It returns when ctx is canceled
// (Ctrl+C) and reports whether a dependency failed.
func runScriptTaskWatch(ctx context.Context, config *Config, order []string, tasksByScript map[string][]Task, targetScript string, sharedEnv []string, opts scriptOptions) bool {
	for _, name := range order {
		if name == targetScript {
			break
		}
		group := config.Scripts[name]
		mode := group.Mode
		if mode == "" {
			mode = "parallel"
		}
		fmt.Printf("[bnm] Running dependency '%s' (Mode: %s)...\n", name, mode)
		if _, ok := runGroup(ctx, mode, group.MaxParallel, tasksByScript[name], sharedEnv); !ok {
			if ctx.Err() == nil {
				fmt.Printf("[bnm] Dependency '%s' failed. Skipping remaining scripts.\n", name)
			}
			return ctx.Err() == nil
		}
		if ctx.Err() != nil {
			return false
		}
	}

	tasks := tasksByScript[targetScript]
	// The log directory must not be watched (see runScriptWatch)
	ignore := ""
	if opts.LogDir != "" {
		ignore, _ = filepath.Abs(opts.LogDir)
	}
	changes, count, err := watchTaskDirs(ctx, tasks, ignore)
	if err != nil {
		fmt.Printf("Error: failed to start file watcher: %v\n", err)
		os.Exit(1)
	}

	restarts := make([]chan struct{}, len(tasks))
	watched := 0
	for i, t := range tasks {
		if t.Watch {
			restarts[i] = make(chan struct{}, 1)
			watched++
		}
	}
	fmt.Printf("[bnm] Starting script '%s' (Mode: parallel)...\n", targetScript)
	fmt.Printf("[bnm] Watch mode: watching %d directories for %d task(s). Press Ctrl+C to stop.\n", count, watched)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case idxs := <-changes:
				for _, i := range idxs {
					select {
					case restarts[i] <- struct{}{}:
					default: // a restart is already pending
					}
				}
			}
		}
	}()

	var sem chan struct{}
	if mp := config.Scripts[targetScript].MaxParallel; mp > 0 {
		sem = make(chan struct{}, mp)
	}
	var wg sync.WaitGroup
	for i, t := range tasks {
		wg.Add(1)
		go func(t Task, restart <-chan struct{}) {
			defer wg.Done()
			superviseTask(ctx, t, sharedEnv, sem, restart)
		}(t, restarts[i])
	}
	wg.Wait()
	return false
}

// superviseTask runs a task and, when restart is non-nil, keeps it under
// watch supervision: a restart signal stops the running process (or wakes a
// finished task) and runs it again. Tasks without a restart channel run
// once. Returns when ctx is canceled.
func superviseTask(ctx context.Context, t Task, sharedEnv []string, sem chan struct{}, restart <-chan struct{}) {
	for {
		if sem != nil {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
		attemptCtx, cancel := context.WithCancel(ctx)
		done := make(chan error, 1)
		go func() {
			// The task directory's .env is reread on every restart
			done <- runTaskAttempts(attemptCtx, t, taskEnv(sharedEnv, t))
		}()

		restarted := false
		select {
		case <-done:
		case <-restart:
			restarted = true
			fmt.Printf("[bnm] Task '%s' changed. Restarting...\n", t.Name)
			cancel()
			<-done
		}
		cancel()
		if sem != nil {
			<-sem
		}
		if ctx.Err() != nil || restart == nil {
			return
		}
		if !restarted {
			// The task exited on its own; wait for the next change
			select {
			case <-restart:
				fmt.Printf("[bnm] Task '%s' changed. Restarting...\n", t.Name)
			case <-ctx.Done():
				return
			}
		}
	}
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
					results = append(results, taskResult{Name: t.Name, Dir: t.DirLabel, Status: statusSkipped})
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
		Dir        string `json:"dir,omitempty"`
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
			Dir:        r.Dir,
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

// resolveTask fills in the display name for tasks that don't carry one
// (the legacy array form and internally built tasks) and turns the task's
// directory — a directories key, an alias, or a path — into the actual path.
// An explicit Name from the config is never regenerated here.
func resolveTask(t *Task, config *Config) {
	t.Name = taskName(*t, config)
	t.DirLabel = taskDirLabel(*t)
	t.Dir = resolveDirPath(config, t.Dir)
}

// taskDirLabel returns the task's directory as written in the config,
// "." for the project root.
func taskDirLabel(t Task) string {
	if t.Dir == "" {
		return "."
	}
	return t.Dir
}

// taskName returns the name a task runs and reports under: the explicit
// name from the config, or the legacy directory-derived fallback.
func taskName(t Task, config *Config) string {
	if t.Name != "" {
		return t.Name
	}
	if t.Dir == "" {
		if config.Name != "" {
			return config.Name
		}
		return "."
	}
	return t.Dir
}

// resolveDirEntry maps a task directory written as a directories key or an
// alias to the entry it refers to, returning the formal key. Keys take
// priority over aliases, matching the priority of filters and bnm exec.
func resolveDirEntry(config *Config, dir string) (string, Directory, bool) {
	if d, exists := config.Directories[dir]; exists {
		return dir, d, true
	}
	for _, key := range sortedKeys(config.Directories) {
		if d := config.Directories[key]; d.Alias != "" && strings.EqualFold(d.Alias, dir) {
			return key, d, true
		}
	}
	return "", Directory{}, false
}

// resolveDirPath maps a task directory — a directories key, an alias, or a
// path — to the path commands run in.
func resolveDirPath(config *Config, dir string) string {
	if dir == "" {
		return "."
	}
	if _, d, ok := resolveDirEntry(config, dir); ok {
		return d.Path
	}
	return dir
}

// filterTasks returns the tasks matching any of the given directory filters.
// A filter is a directory key, a path, or an alias; key and path matches
// take priority, so an alias colliding with another directory's name never
// pulls in extra tasks. Every filter must match at least one task.
func filterTasks(config *Config, tasks []Task, filters []string) ([]Task, error) {
	if len(filters) == 0 {
		return tasks, nil
	}
	include := make([]bool, len(tasks))
	for _, f := range filters {
		matchedAny := false
		for i, task := range tasks {
			if taskMatchesName(config, task, f) {
				include[i] = true
				matchedAny = true
			}
		}
		if !matchedAny {
			for i, task := range tasks {
				if taskMatchesAlias(config, task, f) {
					include[i] = true
					matchedAny = true
				}
			}
		}
		if !matchedAny {
			return nil, fmt.Errorf("no tasks in this script match directory filter '%s'", f)
		}
	}
	var matched []Task
	for i, task := range tasks {
		if include[i] {
			matched = append(matched, task)
		}
	}
	return matched, nil
}

// taskMatchesName reports whether the filter equals the task's directory as
// written, the directory's formal key (also when the task refers to it by
// alias), or its path.
func taskMatchesName(config *Config, task Task, filter string) bool {
	dirRaw := task.Dir
	if dirRaw == "" {
		dirRaw = "."
	}
	if strings.EqualFold(dirRaw, filter) {
		return true
	}
	dirPath := dirRaw
	if key, d, ok := resolveDirEntry(config, task.Dir); ok {
		if strings.EqualFold(key, filter) {
			return true
		}
		dirPath = d.Path
	}
	clean := func(s string) string { return strings.TrimPrefix(s, "./") }
	return strings.EqualFold(clean(dirPath), clean(filter))
}

// filterTasksByTaskName returns the tasks whose name equals any of the given
// task filters (--task / -T). It runs after the directory filters, so the
// two are an AND condition. all is the script's unfiltered task list, used
// to tell a nonexistent name apart from one excluded by a directory filter.
func filterTasksByTaskName(config *Config, tasks, all []Task, names []string, script string) ([]Task, error) {
	if len(names) == 0 {
		return tasks, nil
	}
	include := make([]bool, len(tasks))
	for _, n := range names {
		matchedAny := false
		for i, task := range tasks {
			if strings.EqualFold(taskName(task, config), n) {
				include[i] = true
				matchedAny = true
			}
		}
		if matchedAny {
			continue
		}
		candidates := make([]string, 0, len(all))
		for _, task := range all {
			name := taskName(task, config)
			if strings.EqualFold(name, n) {
				return nil, fmt.Errorf("task '%s' in script '%s' does not match the directory filter", n, script)
			}
			candidates = append(candidates, name)
		}
		msg := fmt.Sprintf("no task named '%s' exists in script '%s'", n, script)
		if s := closestMatch(n, candidates); s != "" {
			msg += fmt.Sprintf("\nDid you mean '%s'?", s)
		}
		return nil, fmt.Errorf("%s", msg)
	}
	var matched []Task
	for i, task := range tasks {
		if include[i] {
			matched = append(matched, task)
		}
	}
	return matched, nil
}

// taskMatchesAlias reports whether the filter equals the alias of the
// directory the task refers to (by key or by alias).
func taskMatchesAlias(config *Config, task Task, filter string) bool {
	_, d, ok := resolveDirEntry(config, task.Dir)
	return ok && d.Alias != "" && strings.EqualFold(d.Alias, filter)
}

// runGroup runs one script group and reports per-task results plus overall
// success. Tasks that never ran remain "skipped".
func runGroup(ctx context.Context, mode string, maxParallel int, tasks []Task, sharedEnv []string) ([]taskResult, bool) {
	results := make([]taskResult, len(tasks))
	for i, t := range tasks {
		results[i] = taskResult{Name: t.Name, Dir: t.DirLabel, Status: statusSkipped}
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
	nameWidth, dirWidth := 12, 0
	for _, r := range results {
		nameWidth = max(nameWidth, len(r.Name))
		dirWidth = max(dirWidth, len(r.Dir))
	}
	fmt.Println("[bnm] Summary:")
	for _, r := range results {
		mark, color := summaryMark(r.Status)
		reset := ""
		if color != "" {
			reset = colorReset
		}
		dir := ""
		if dirWidth > 0 {
			dir = fmt.Sprintf("%-*s ", dirWidth, r.Dir)
		}
		line := fmt.Sprintf("  %s%s %-*s %s%-9s%s", color, mark, nameWidth, r.Name, dir, r.Status, reset)
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
