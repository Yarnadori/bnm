package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
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

	got, err := filterTasks(config, tasks, []string{"-F"})
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
	if _, err := filterTasks(config, tasks, []string{"-X"}); err == nil {
		t.Error("expected error for unknown alias filter")
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
	if got != "--port 3000" {
		t.Errorf("got %q", got)
	}

	got = shellJoin([]string{"--name", "hello world"})
	var want string
	if runtime.GOOS == "windows" {
		want = `--name "hello world"`
	} else {
		want = "--name 'hello world'"
	}
	if got != want {
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
	filters, extra := splitScriptArgs([]string{"-F", "--", "--port", "3000"})
	if !reflect.DeepEqual(filters, []string{"-F"}) {
		t.Errorf("filters: got %v", filters)
	}
	if !reflect.DeepEqual(extra, []string{"--port", "3000"}) {
		t.Errorf("extra: got %v", extra)
	}

	filters, extra = splitScriptArgs([]string{"-F"})
	if !reflect.DeepEqual(filters, []string{"-F"}) || extra != nil {
		t.Errorf("got %v / %v", filters, extra)
	}

	filters, extra = splitScriptArgs(nil)
	if len(filters) != 0 || len(extra) != 0 {
		t.Errorf("got %v / %v", filters, extra)
	}
}
