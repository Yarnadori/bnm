package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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
	if opts != (scriptOptions{}) {
		t.Errorf("opts: got %+v, want zero", opts)
	}

	filters, extra, opts, _ = splitScriptArgs(nil)
	if len(filters) != 0 || len(extra) != 0 || opts != (scriptOptions{}) {
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
