package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func testConfig() *Config {
	return &Config{
		Name: "my-app",
		Directories: map[string]Directory{
			"FRONTEND": {Alias: "F", Path: "./frontend"},
			"BACKEND":  {Alias: "B", Path: "./backend"},
		},
		Scripts: map[string]ScriptGroup{
			"build": {Tasks: []Task{{Dir: "FRONTEND", Command: "echo build"}}},
			"test":  {DependsOn: []string{"build"}, Tasks: []Task{{Dir: "FRONTEND", Command: "echo test"}}},
			"deploy": {
				DependsOn: []string{"test", "build"},
				Tasks:     []Task{{Dir: "BACKEND", Command: "echo deploy"}},
			},
		},
	}
}

func TestExecutionOrder(t *testing.T) {
	config := testConfig()

	got := executionOrder(config, "deploy")
	want := []string{"build", "test", "deploy"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	got = executionOrder(config, "build")
	if !reflect.DeepEqual(got, []string{"build"}) {
		t.Errorf("got %v, want [build]", got)
	}
}

func TestFilterTasksByAlias(t *testing.T) {
	config := testConfig()
	tasks := []Task{
		{Dir: "FRONTEND", Command: "echo f"},
		{Dir: "BACKEND", Command: "echo b"},
	}

	got, err := filterTasks(config, tasks, []string{"F"})
	if err != nil {
		t.Fatalf("filterTasks failed: %v", err)
	}
	if len(got) != 1 || got[0].Dir != "FRONTEND" {
		t.Errorf("got %v, want only FRONTEND task", got)
	}
}

func TestFilterTasksByKeyAndPath(t *testing.T) {
	config := testConfig()
	tasks := []Task{
		{Dir: "FRONTEND", Command: "echo f"},
		{Dir: "BACKEND", Command: "echo b"},
	}

	for _, filter := range []string{"backend", "./backend", "BACKEND"} {
		got, err := filterTasks(config, tasks, []string{filter})
		if err != nil {
			t.Fatalf("filterTasks(%q) failed: %v", filter, err)
		}
		if len(got) != 1 || got[0].Dir != "BACKEND" {
			t.Errorf("filter %q: got %v, want only BACKEND task", filter, got)
		}
	}
}

func TestFilterTasksNoMatch(t *testing.T) {
	config := testConfig()
	tasks := []Task{{Dir: "FRONTEND", Command: "echo f"}}

	if _, err := filterTasks(config, tasks, []string{"BACKEND"}); err == nil {
		t.Error("expected error for filter matching no tasks")
	}
	if _, err := filterTasks(config, tasks, []string{"X"}); err == nil {
		t.Error("expected error for unknown name filter")
	}
}

func TestFilterTasksEmptyFiltersKeepsAll(t *testing.T) {
	config := testConfig()
	tasks := []Task{
		{Dir: "FRONTEND", Command: "echo f"},
		{Dir: "BACKEND", Command: "echo b"},
	}

	got, err := filterTasks(config, tasks, nil)
	if err != nil {
		t.Fatalf("filterTasks failed: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d tasks, want 2", len(got))
	}
}

func TestFilterTasksNamePriorityOverAlias(t *testing.T) {
	// "Z" is both a directory key and another directory's alias: the key
	// match must win, and the alias-only task must not be pulled in
	config := &Config{Directories: map[string]Directory{
		"Z":     {Path: "./real-z"},
		"OTHER": {Alias: "Z", Path: "./other"},
	}}
	tasks := []Task{
		{Dir: "Z", Command: "echo z"},
		{Dir: "OTHER", Command: "echo other"},
	}

	got, err := filterTasks(config, tasks, []string{"Z"})
	if err != nil {
		t.Fatalf("filterTasks failed: %v", err)
	}
	if len(got) != 1 || got[0].Dir != "Z" {
		t.Errorf("got %v, want only the Z task", got)
	}

	// With no key or path named Z, the alias still matches
	delete(config.Directories, "Z")
	got, err = filterTasks(config, tasks[1:], []string{"Z"})
	if err != nil {
		t.Fatalf("filterTasks failed: %v", err)
	}
	if len(got) != 1 || got[0].Dir != "OTHER" {
		t.Errorf("alias fallback: got %v", got)
	}
}

func TestResolveTask(t *testing.T) {
	config := testConfig()
	cases := []struct {
		in       Task
		name     string
		dir      string
		dirLabel string
	}{
		// Explicit names survive resolution; dir resolves by key, alias, or path
		{Task{Name: "lint", Dir: "FRONTEND"}, "lint", "./frontend", "FRONTEND"},
		{Task{Name: "lint", Dir: "F"}, "lint", "./frontend", "F"},
		{Task{Name: "lint", Dir: "./elsewhere"}, "lint", "./elsewhere", "./elsewhere"},
		{Task{Name: "lint", Dir: "."}, "lint", ".", "."},
		{Task{Name: "lint"}, "lint", ".", "."},
		// Legacy tasks without a name fall back to the directory
		{Task{Dir: "FRONTEND"}, "FRONTEND", "./frontend", "FRONTEND"},
		{Task{Dir: "./elsewhere"}, "./elsewhere", "./elsewhere", "./elsewhere"},
		{Task{}, "my-app", ".", "."},
	}
	for _, c := range cases {
		got := c.in
		resolveTask(&got, config)
		if got.Name != c.name || got.Dir != c.dir || got.DirLabel != c.dirLabel {
			t.Errorf("resolveTask(%+v): got name %q dir %q label %q, want %q %q %q",
				c.in, got.Name, got.Dir, got.DirLabel, c.name, c.dir, c.dirLabel)
		}
	}
}

func TestFilterTasksByTaskName(t *testing.T) {
	config := testConfig()
	tasks := []Task{
		{Name: "frontend-lint", Dir: "FRONTEND", Command: "echo lint"},
		{Name: "frontend-test", Dir: "FRONTEND", Command: "echo test"},
		{Name: "backend-test", Dir: "BACKEND", Command: "echo b"},
	}

	got, err := filterTasksByTaskName(config, tasks, tasks, []string{"frontend-lint"}, "check")
	if err != nil {
		t.Fatalf("filterTasksByTaskName failed: %v", err)
	}
	if len(got) != 1 || got[0].Name != "frontend-lint" {
		t.Errorf("got %v, want only frontend-lint", got)
	}

	// Multiple names accumulate, keeping task order
	got, err = filterTasksByTaskName(config, tasks, tasks, []string{"backend-test", "frontend-lint"}, "check")
	if err != nil {
		t.Fatalf("filterTasksByTaskName failed: %v", err)
	}
	if len(got) != 2 || got[0].Name != "frontend-lint" || got[1].Name != "backend-test" {
		t.Errorf("got %v, want frontend-lint and backend-test", got)
	}

	// Empty filter list keeps everything
	got, err = filterTasksByTaskName(config, tasks, tasks, nil, "check")
	if err != nil || len(got) != 3 {
		t.Errorf("got %v / %v, want all tasks", got, err)
	}

	// Legacy tasks without explicit names match by their derived name
	legacy := []Task{{Dir: "FRONTEND", Command: "echo f"}}
	got, err = filterTasksByTaskName(config, legacy, legacy, []string{"FRONTEND"}, "check")
	if err != nil || len(got) != 1 {
		t.Errorf("legacy: got %v / %v", got, err)
	}
}

func TestFilterTasksByTaskNameUnknown(t *testing.T) {
	config := testConfig()
	tasks := []Task{{Name: "frontend-lint", Dir: "FRONTEND", Command: "echo lint"}}

	_, err := filterTasksByTaskName(config, tasks, tasks, []string{"frontend-lnit"}, "check")
	if err == nil {
		t.Fatal("expected error for unknown task name")
	}
	msg := err.Error()
	if !strings.Contains(msg, "no task named 'frontend-lnit' exists in script 'check'") {
		t.Errorf("error: got %q", msg)
	}
	if !strings.Contains(msg, "Did you mean 'frontend-lint'?") {
		t.Errorf("error is missing the suggestion: %q", msg)
	}
}

func TestFilterTasksByTaskNameAndDirectoryFilterAreAnded(t *testing.T) {
	config := testConfig()
	all := []Task{
		{Name: "frontend-lint", Dir: "FRONTEND", Command: "echo lint"},
		{Name: "frontend-test", Dir: "FRONTEND", Command: "echo test"},
		{Name: "backend-test", Dir: "BACKEND", Command: "echo b"},
	}

	// Directory filter first, task filter on the result
	byDir, err := filterTasks(config, all, []string{"FRONTEND"})
	if err != nil {
		t.Fatalf("filterTasks failed: %v", err)
	}
	if len(byDir) != 2 {
		t.Fatalf("dir filter: got %v, want both FRONTEND tasks", byDir)
	}
	got, err := filterTasksByTaskName(config, byDir, all, []string{"frontend-lint"}, "check")
	if err != nil {
		t.Fatalf("filterTasksByTaskName failed: %v", err)
	}
	if len(got) != 1 || got[0].Name != "frontend-lint" {
		t.Errorf("got %v, want only frontend-lint", got)
	}

	// A task that exists but is excluded by the directory filter is an error,
	// not a typo suggestion
	_, err = filterTasksByTaskName(config, byDir, all, []string{"backend-test"}, "check")
	if err == nil || !strings.Contains(err.Error(), "does not match the directory filter") {
		t.Errorf("got %v, want directory-filter mismatch error", err)
	}
}

func TestFilterTasksDirectorySelectsAllTasksInDir(t *testing.T) {
	config := testConfig()
	tasks := []Task{
		{Name: "frontend-lint", Dir: "FRONTEND", Command: "echo lint"},
		{Name: "frontend-test", Dir: "FRONTEND", Command: "echo test"},
		{Name: "backend-test", Dir: "BACKEND", Command: "echo b"},
	}

	for _, filter := range []string{"FRONTEND", "F", "./frontend"} {
		got, err := filterTasks(config, tasks, []string{filter})
		if err != nil {
			t.Fatalf("filterTasks(%q) failed: %v", filter, err)
		}
		if len(got) != 2 || got[0].Name != "frontend-lint" || got[1].Name != "frontend-test" {
			t.Errorf("filter %q: got %v, want both FRONTEND tasks", filter, got)
		}
	}
}

func TestFilterTasksDirWrittenAsAlias(t *testing.T) {
	// A task whose dir is an alias must be reachable by the directory's
	// formal name, the alias itself, and the path
	config := &Config{Directories: map[string]Directory{
		"WEB": {Alias: "F", Path: "./apps/frontend"},
	}}
	tasks := []Task{{Name: "lint", Dir: "F", Command: "echo lint"}}

	for _, filter := range []string{"WEB", "F", "./apps/frontend", "apps/frontend"} {
		got, err := filterTasks(config, tasks, []string{filter})
		if err != nil {
			t.Errorf("filter %q: %v", filter, err)
			continue
		}
		if len(got) != 1 || got[0].Name != "lint" {
			t.Errorf("filter %q: got %v, want the lint task", filter, got)
		}
	}
}

func TestFilterTasksRootDir(t *testing.T) {
	config := testConfig()
	tasks := []Task{
		{Dir: "", Command: "echo root"},
		{Dir: "FRONTEND", Command: "echo f"},
	}

	got, err := filterTasks(config, tasks, []string{"."})
	if err != nil {
		t.Fatalf("filterTasks failed: %v", err)
	}
	if len(got) != 1 || got[0].Command.String() != "echo root" {
		t.Errorf("got %v, want only root task", got)
	}
}

func TestShellJoin(t *testing.T) {
	got := shellJoin([]string{"--port", "3000"})
	var want string
	if runtime.GOOS == "windows" {
		want = `"--port" "3000"`
	} else {
		want = "'--port' '3000'"
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestShellQuotePOSIX(t *testing.T) {
	cases := map[string]string{
		"hello world":  "'hello world'",
		"line1\nline2": "'line1\nline2'",
		"it's":         "'it'\\''s'",
		"#comment":     "'#comment'",
		"~":            "'~'",
	}
	for input, want := range cases {
		if got := shellQuotePOSIX(input); got != want {
			t.Errorf("shellQuotePOSIX(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestShellQuoteWindows(t *testing.T) {
	if got, want := shellQuoteWindows(`%PATH% "quoted"`), `"%PATH% ""quoted"""`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTaskEnvOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FROM_FILE=file\nSHARED=file\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	task := Task{
		Dir: dir,
		Env: map[string]string{"FROM_TASK": "task", "SHARED": "task"},
	}
	env := taskEnv([]string{"SHARED=shared", "BASE=1"}, task)

	// Later entries win, so the task-level SHARED must come after the others
	last := map[string]string{}
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				last[e[:i]] = e[i+1:]
				break
			}
		}
	}
	if last["FROM_FILE"] != "file" {
		t.Errorf("FROM_FILE: got %q", last["FROM_FILE"])
	}
	if last["FROM_TASK"] != "task" {
		t.Errorf("FROM_TASK: got %q", last["FROM_TASK"])
	}
	if last["SHARED"] != "task" {
		t.Errorf("SHARED: got %q, want task-level value to win", last["SHARED"])
	}
	if last["BASE"] != "1" {
		t.Errorf("BASE: got %q", last["BASE"])
	}
}

func TestTaskEnvDoesNotMutateShared(t *testing.T) {
	shared := make([]string, 1, 8)
	shared[0] = "BASE=1"
	env1 := taskEnv(shared, Task{Env: map[string]string{"A": "1"}})
	env2 := taskEnv(shared, Task{Env: map[string]string{"B": "2"}})
	if env1[1] != "A=1" || env2[1] != "B=2" {
		t.Errorf("taskEnv results interfere: %v vs %v", env1, env2)
	}
}

func TestSplitScriptArgs(t *testing.T) {
	filters, extra, opts, err := splitScriptArgs([]string{"frontend", "--", "--port", "3000"})
	if err != nil {
		t.Fatalf("splitScriptArgs failed: %v", err)
	}
	if !reflect.DeepEqual(filters, []string{"frontend"}) {
		t.Errorf("filters: got %v", filters)
	}
	if !reflect.DeepEqual(extra, []string{"--port", "3000"}) {
		t.Errorf("extra: got %v", extra)
	}
	if !reflect.DeepEqual(opts, scriptOptions{}) {
		t.Errorf("opts: got %+v, want zero", opts)
	}

	filters, extra, opts, _ = splitScriptArgs(nil)
	if len(filters) != 0 || len(extra) != 0 || !reflect.DeepEqual(opts, scriptOptions{}) {
		t.Errorf("got %v / %v / %+v", filters, extra, opts)
	}
}

func TestSplitScriptArgsFilterFlag(t *testing.T) {
	// Positional names and --filter/-F values accumulate in order
	filters, _, _, err := splitScriptArgs([]string{"--filter", "frontend", "-F", "backend", "docs"})
	if err != nil {
		t.Fatalf("splitScriptArgs failed: %v", err)
	}
	if !reflect.DeepEqual(filters, []string{"frontend", "backend", "docs"}) {
		t.Errorf("filters: got %v", filters)
	}

	// -F now takes a value; bare -F is an error
	if _, _, _, err := splitScriptArgs([]string{"-F"}); err == nil {
		t.Error("expected error for -F without a value")
	}
}

func TestSplitScriptArgsOptions(t *testing.T) {
	filters, extra, opts, err := splitScriptArgs([]string{"--watch", "-F", "frontend", "--dry-run", "--no-color", "--", "--watch"})
	if err != nil {
		t.Fatalf("splitScriptArgs failed: %v", err)
	}
	if !opts.Watch || !opts.DryRun || !opts.NoColor {
		t.Errorf("opts: got %+v, want watch, dry-run, and no-color set", opts)
	}
	if !reflect.DeepEqual(filters, []string{"frontend"}) {
		t.Errorf("filters: got %v", filters)
	}
	// Flags after "--" are pass-through arguments, not options
	if !reflect.DeepEqual(extra, []string{"--watch"}) {
		t.Errorf("extra: got %v", extra)
	}

	_, _, opts, _ = splitScriptArgs([]string{"-w", "-n"})
	if !opts.Watch || !opts.DryRun {
		t.Errorf("short flags: got %+v", opts)
	}
}

func TestAnyTaskWatch(t *testing.T) {
	if anyTaskWatch([]Task{{Command: "a"}, {Command: "b"}}) {
		t.Error("got true for tasks without watch")
	}
	if !anyTaskWatch([]Task{{Command: "a"}, {Command: "b", Watch: true}}) {
		t.Error("got false for a watch task")
	}
}

// waitForLines polls file until it holds want non-empty lines or the
// deadline passes, and returns the count seen last.
func waitForLines(t *testing.T, file string, want int) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	got := 0
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(file)
		got = len(strings.Fields(string(data)))
		if got >= want {
			return got
		}
		time.Sleep(20 * time.Millisecond)
	}
	return got
}

func TestSuperviseTaskRestarts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	runs := filepath.Join(dir, "runs")
	// Records each start, then stays alive like a dev server
	task := Task{Name: "srv", Watch: true, Command: Command("echo x >> " + runs + " && sleep 30")}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	restart := make(chan struct{}, 1)
	stopped := make(chan struct{})
	go func() {
		superviseTask(ctx, task, os.Environ(), nil, restart)
		close(stopped)
	}()

	if got := waitForLines(t, runs, 1); got != 1 {
		t.Fatalf("initial run: got %d starts", got)
	}
	restart <- struct{}{}
	if got := waitForLines(t, runs, 2); got != 2 {
		t.Fatalf("after restart: got %d starts", got)
	}

	cancel()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("supervisor did not stop on cancel")
	}
}

func TestSuperviseTaskRestartAfterExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	runs := filepath.Join(dir, "runs")
	// A short-lived task: the supervisor must wait for the next change
	// instead of looping
	task := Task{Name: "build", Watch: true, Command: Command("echo x >> " + runs)}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	restart := make(chan struct{}, 1)
	stopped := make(chan struct{})
	go func() {
		superviseTask(ctx, task, os.Environ(), nil, restart)
		close(stopped)
	}()

	if got := waitForLines(t, runs, 1); got != 1 {
		t.Fatalf("initial run: got %d starts", got)
	}
	// No signal: it must stay at one run
	time.Sleep(200 * time.Millisecond)
	if got := waitForLines(t, runs, 1); got != 1 {
		t.Fatalf("idle: got %d starts, want still 1", got)
	}
	restart <- struct{}{}
	if got := waitForLines(t, runs, 2); got != 2 {
		t.Fatalf("after restart: got %d starts", got)
	}
	cancel()
	<-stopped
}

func TestSuperviseTaskWithoutWatchRunsOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	runs := filepath.Join(dir, "runs")
	task := Task{Name: "once", Command: Command("echo x >> " + runs)}

	done := make(chan struct{})
	go func() {
		superviseTask(context.Background(), task, os.Environ(), nil, nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("unwatched task did not return after exiting")
	}
	if got := waitForLines(t, runs, 1); got != 1 {
		t.Fatalf("got %d starts, want 1", got)
	}
}

func TestWatchEnabled(t *testing.T) {
	cases := []struct {
		group ScriptGroup
		opts  scriptOptions
		want  bool
	}{
		{ScriptGroup{}, scriptOptions{}, false},
		{ScriptGroup{}, scriptOptions{Watch: true}, true},
		{ScriptGroup{Watch: true}, scriptOptions{}, true},
		{ScriptGroup{Watch: true}, scriptOptions{NoWatch: true}, false},
		{ScriptGroup{Watch: true}, scriptOptions{Watch: true}, true},
	}
	for _, c := range cases {
		if got := watchEnabled(c.group, c.opts); got != c.want {
			t.Errorf("watchEnabled(watch=%v, %+v) = %v, want %v", c.group.Watch, c.opts, got, c.want)
		}
	}
}

func TestSplitScriptArgsNoWatch(t *testing.T) {
	_, _, opts, err := splitScriptArgs([]string{"--no-watch"})
	if err != nil || !opts.NoWatch {
		t.Errorf("got %+v / %v, want NoWatch set", opts, err)
	}

	// The two flags contradict each other
	if _, _, _, err := splitScriptArgs([]string{"--watch", "--no-watch"}); err == nil {
		t.Error("expected error for --watch with --no-watch")
	}
	// Also when everything after -- is pass-through
	if _, _, _, err := splitScriptArgs([]string{"-w", "--no-watch", "--", "x"}); err == nil {
		t.Error("expected error for -w with --no-watch before --")
	}
}

func TestSplitScriptArgsTaskFlag(t *testing.T) {
	// --task/-T values accumulate separately from directory filters
	filters, _, opts, err := splitScriptArgs([]string{"--task", "frontend-lint", "-T", "backend-test", "-F", "frontend"})
	if err != nil {
		t.Fatalf("splitScriptArgs failed: %v", err)
	}
	if !reflect.DeepEqual(opts.TaskFilters, []string{"frontend-lint", "backend-test"}) {
		t.Errorf("task filters: got %v", opts.TaskFilters)
	}
	if !reflect.DeepEqual(filters, []string{"frontend"}) {
		t.Errorf("filters: got %v", filters)
	}

	_, _, opts, err = splitScriptArgs([]string{"--task=lint"})
	if err != nil || !reflect.DeepEqual(opts.TaskFilters, []string{"lint"}) {
		t.Errorf("--task=lint: got %v / %v", opts.TaskFilters, err)
	}

	if _, _, _, err := splitScriptArgs([]string{"-T"}); err == nil {
		t.Error("expected error for -T without a value")
	}
}

func TestSplitScriptArgsValueFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--log-dir", "logs", "--summary", "json"},
		{"--log-dir=logs", "--summary=json"},
	} {
		_, _, opts, err := splitScriptArgs(args)
		if err != nil {
			t.Fatalf("splitScriptArgs(%v) failed: %v", args, err)
		}
		if opts.LogDir != "logs" || opts.Summary != "json" {
			t.Errorf("splitScriptArgs(%v): got %+v", args, opts)
		}
	}

	for _, args := range [][]string{
		{"--log-dir"},
		{"--log-dir", "--"},
		{"--summary", "yaml"},
	} {
		if _, _, _, err := splitScriptArgs(args); err == nil {
			t.Errorf("splitScriptArgs(%v): expected error", args)
		}
	}
}

func TestSanitizeLogName(t *testing.T) {
	cases := map[string]string{
		"FRONTEND":  "FRONTEND",
		"./backend": "backend",
		".":         "root",
		"my app":    "my_app",
		"a/b":       "a_b",
	}
	for input, want := range cases {
		if got := sanitizeLogName(input); got != want {
			t.Errorf("sanitizeLogName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestUniquePathCaseFold(t *testing.T) {
	orig := logPathFold
	t.Cleanup(func() { logPathFold = orig })

	// Case-insensitive filesystems (Windows, default macOS): A and a collide
	logPathFold = true
	used := map[string]bool{}
	if got := uniquePath(used, "A", ".log"); got != "A.log" {
		t.Errorf("got %q, want A.log", got)
	}
	if got := uniquePath(used, "a", ".log"); got != "a-2.log" {
		t.Errorf("got %q, want a-2.log", got)
	}

	// Case-sensitive filesystems: they are distinct files
	logPathFold = false
	used = map[string]bool{}
	if got := uniquePath(used, "A", ".log"); got != "A.log" {
		t.Errorf("got %q, want A.log", got)
	}
	if got := uniquePath(used, "a", ".log"); got != "a.log" {
		t.Errorf("got %q, want a.log", got)
	}
}

func TestSummaryJSONWithDir(t *testing.T) {
	results := []taskResult{
		{Name: "frontend-lint", Dir: "frontend", Status: statusOK, Duration: 1200 * time.Millisecond},
		{Name: "frontend-typecheck", Dir: "frontend", Status: statusFailed, Duration: 3800 * time.Millisecond},
	}
	got := string(summaryJSON("check", results, true))
	want := `{"script":"check","ok":false,"tasks":[` +
		`{"name":"frontend-lint","dir":"frontend","status":"ok","durationMs":1200},` +
		`{"name":"frontend-typecheck","dir":"frontend","status":"failed","durationMs":3800}]}`
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestRunGroupMultipleTasksSameDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	newTasks := func() []Task {
		return []Task{
			{Name: "one", Dir: dir, DirLabel: "app", Command: Command("touch one.marker")},
			{Name: "two", Dir: dir, DirLabel: "app", Command: Command("touch two.marker")},
		}
	}

	for _, mode := range []string{"parallel", "sequential"} {
		os.Remove(filepath.Join(dir, "one.marker"))
		os.Remove(filepath.Join(dir, "two.marker"))
		results, ok := runGroup(context.Background(), mode, 0, newTasks(), os.Environ())
		if !ok {
			t.Fatalf("%s: runGroup reported failure: %v", mode, results)
		}
		for _, marker := range []string{"one.marker", "two.marker"} {
			if _, err := os.Stat(filepath.Join(dir, marker)); err != nil {
				t.Errorf("%s: %s was not created: %v", mode, marker, err)
			}
		}
		if results[0].Name != "one" || results[1].Name != "two" {
			t.Errorf("%s: result names: got %v", mode, results)
		}
		if results[0].Dir != "app" || results[1].Dir != "app" {
			t.Errorf("%s: result dirs: got %v", mode, results)
		}
	}
}

func TestSummaryJSON(t *testing.T) {
	results := []taskResult{
		{Name: "FRONTEND", Status: statusOK, Duration: 1500 * time.Millisecond},
		{Name: "BACKEND", Status: statusFailed, Duration: 20 * time.Millisecond},
	}
	got := string(summaryJSON("dev", results, true))
	want := `{"script":"dev","ok":false,"tasks":[` +
		`{"name":"FRONTEND","status":"ok","durationMs":1500},` +
		`{"name":"BACKEND","status":"failed","durationMs":20}]}`
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestAssignLogPaths(t *testing.T) {
	logDir := t.TempDir()
	tasksByScript := map[string][]Task{
		"dev": {
			{Name: "FRONTEND", Command: "echo f"},
			{Name: "FRONTEND", Command: "echo f2"},
			{Name: "./api", Command: "echo a"},
		},
	}
	if err := assignLogPaths(logDir, []string{"dev"}, tasksByScript); err != nil {
		t.Fatalf("assignLogPaths failed: %v", err)
	}
	want := []string{
		filepath.Join(logDir, "dev", "FRONTEND.log"),
		filepath.Join(logDir, "dev", "FRONTEND-2.log"),
		filepath.Join(logDir, "dev", "api.log"),
	}
	for i, w := range want {
		if got := tasksByScript["dev"][i].LogPath; got != w {
			t.Errorf("task %d: got %q, want %q", i, got, w)
		}
		if _, err := os.Stat(w); err != nil {
			t.Errorf("log file %q was not created: %v", w, err)
		}
	}
}

func TestAssignLogPathsCollisions(t *testing.T) {
	logDir := t.TempDir()
	// Task names A, A, A-2 must not share a file, and the sanitized script
	// names a/b and a_b must not share a directory.
	tasksByScript := map[string][]Task{
		"a/b": {
			{Name: "A", Command: "echo 1"},
			{Name: "A", Command: "echo 2"},
			{Name: "A-2", Command: "echo 3"},
		},
		"a_b": {
			{Name: "A", Command: "echo 4"},
		},
	}
	if err := assignLogPaths(logDir, []string{"a/b", "a_b"}, tasksByScript); err != nil {
		t.Fatalf("assignLogPaths failed: %v", err)
	}
	seen := map[string]bool{}
	for _, tasks := range tasksByScript {
		for _, task := range tasks {
			if seen[task.LogPath] {
				t.Errorf("log path %q assigned twice", task.LogPath)
			}
			seen[task.LogPath] = true
		}
	}
	want := []string{
		filepath.Join(logDir, "a_b", "A.log"),
		filepath.Join(logDir, "a_b", "A-2.log"),
		filepath.Join(logDir, "a_b", "A-2-2.log"),
	}
	for i, w := range want {
		if got := tasksByScript["a/b"][i].LogPath; got != w {
			t.Errorf("a/b task %d: got %q, want %q", i, got, w)
		}
	}
	if got, w := tasksByScript["a_b"][0].LogPath, filepath.Join(logDir, "a_b-2", "A.log"); got != w {
		t.Errorf("a_b task: got %q, want %q", got, w)
	}
}

func TestRunTaskAttemptsRetries(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")
	// Fails on the first attempt, succeeds on the second
	task := Task{
		Name:    "retry",
		Command: Command("test -f " + marker + " || { touch " + marker + "; exit 1; }"),
		Retries: 1,
	}
	if err := runTaskAttempts(t.Context(), task, os.Environ()); err != nil {
		t.Errorf("expected success after retry, got %v", err)
	}

	// No retries: the same command fails
	os.Remove(marker)
	task.Retries = 0
	if err := runTaskAttempts(t.Context(), task, os.Environ()); err == nil {
		t.Error("expected failure without retries")
	}
}

func TestRunTaskAttemptsTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	task := Task{
		Name:    "slow",
		Command: "sleep 5",
		Timeout: Duration(100 * time.Millisecond),
	}
	start := time.Now()
	err := runTaskAttempts(t.Context(), task, os.Environ())
	if err == nil {
		t.Error("expected timeout error")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("task was not killed by timeout (took %s)", elapsed)
	}
}
