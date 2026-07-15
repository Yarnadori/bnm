package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCommandUnmarshalString(t *testing.T) {
	var task Task
	if err := json.Unmarshal([]byte(`{"command": "echo hello"}`), &task); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if task.Command.String() != "echo hello" {
		t.Errorf("got %q, want %q", task.Command.String(), "echo hello")
	}
}

func TestCommandUnmarshalOSMap(t *testing.T) {
	data := []byte(`{"command": {"` + runtime.GOOS + `": "echo current-os", "default": "echo fallback"}}`)
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if task.Command.String() != "echo current-os" {
		t.Errorf("got %q, want %q", task.Command.String(), "echo current-os")
	}
}

func TestCommandUnmarshalDefaultFallback(t *testing.T) {
	data := []byte(`{"command": {"someother-os": "echo other", "default": "echo fallback"}}`)
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if task.Command.String() != "echo fallback" {
		t.Errorf("got %q, want %q", task.Command.String(), "echo fallback")
	}
}

func TestCommandUnmarshalInvalidType(t *testing.T) {
	for _, data := range []string{
		`{"command": 123}`,
		`{"command": ["echo", "hi"]}`,
		`{"command": {"linux": 123}}`,
	} {
		var task Task
		if err := json.Unmarshal([]byte(data), &task); err == nil {
			t.Errorf("unmarshal %s: expected error", data)
		}
	}
}

func TestCommandUnmarshalNoMatch(t *testing.T) {
	data := []byte(`{"command": {"someother-os": "echo other"}}`)
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if task.Command.String() != "" {
		t.Errorf("got %q, want empty command", task.Command.String())
	}
}

func writeConfig(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfigValid(t *testing.T) {
	writeConfig(t, `{
		"name": "my-app",
		"version": "1.0.0",
		"directories": {"FRONTEND": {"alias": "F", "path": "./frontend"}},
		"scripts": {"dev": {"mode": "parallel", "tasks": [{"dir": "FRONTEND", "command": "echo dev"}]}}
	}`)

	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if config.Name != "my-app" {
		t.Errorf("Name: got %q, want %q", config.Name, "my-app")
	}
	if config.Directories["FRONTEND"].Path != "./frontend" {
		t.Errorf("Path: got %q", config.Directories["FRONTEND"].Path)
	}
	if len(config.Scripts["dev"].Tasks) != 1 {
		t.Errorf("Tasks: got %d, want 1", len(config.Scripts["dev"].Tasks))
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := loadConfig(); err == nil {
		t.Error("expected error for missing bnm.json")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	writeConfig(t, `{invalid`)
	if _, err := loadConfig(); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadConfigUnknownMode(t *testing.T) {
	writeConfig(t, `{"scripts": {"dev": {"mode": "bogus", "tasks": []}}}`)
	if _, err := loadConfig(); err == nil {
		t.Error("expected error for unknown mode")
	}
}

func TestLoadConfigNegativeMaxParallel(t *testing.T) {
	writeConfig(t, `{"scripts": {"dev": {"maxParallel": -1, "tasks": []}}}`)
	if _, err := loadConfig(); err == nil {
		t.Error("expected error for negative maxParallel")
	}
}

func TestLoadConfigUnknownDependency(t *testing.T) {
	writeConfig(t, `{"scripts": {"dev": {"dependsOn": ["missing"], "tasks": []}}}`)
	if _, err := loadConfig(); err == nil {
		t.Error("expected error for unknown dependsOn reference")
	}
}

func TestLoadConfigDependencyCycle(t *testing.T) {
	writeConfig(t, `{"scripts": {
		"a": {"dependsOn": ["b"], "tasks": []},
		"b": {"dependsOn": ["a"], "tasks": []}
	}}`)
	if _, err := loadConfig(); err == nil {
		t.Error("expected error for dependency cycle")
	}
}

func TestLoadConfigSelfDependencyCycle(t *testing.T) {
	writeConfig(t, `{"scripts": {"a": {"dependsOn": ["a"], "tasks": []}}}`)
	if _, err := loadConfig(); err == nil {
		t.Error("expected error for self dependency")
	}
}

func TestLoadConfigValidDependencies(t *testing.T) {
	writeConfig(t, `{"scripts": {
		"build": {"tasks": []},
		"deploy": {"dependsOn": ["build"], "maxParallel": 2, "tasks": [
			{"command": "echo hi", "env": {"FOO": "bar"}}
		]}
	}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	deploy := config.Scripts["deploy"]
	if deploy.MaxParallel != 2 {
		t.Errorf("MaxParallel: got %d, want 2", deploy.MaxParallel)
	}
	if len(deploy.DependsOn) != 1 || deploy.DependsOn[0] != "build" {
		t.Errorf("DependsOn: got %v", deploy.DependsOn)
	}
	if deploy.Tasks[0].Env["FOO"] != "bar" {
		t.Errorf("Env: got %v", deploy.Tasks[0].Env)
	}
}

func TestLoadConfigTimeoutAndRetries(t *testing.T) {
	writeConfig(t, `{"scripts": {"dev": {"tasks": [
		{"command": "echo hi", "timeout": "30s", "retries": 2}
	]}}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	task := config.Scripts["dev"].Tasks[0]
	if task.Timeout != Duration(30*time.Second) {
		t.Errorf("Timeout: got %s, want 30s", task.Timeout)
	}
	if task.Retries != 2 {
		t.Errorf("Retries: got %d, want 2", task.Retries)
	}
}

func TestLoadConfigInvalidTimeout(t *testing.T) {
	for _, timeout := range []string{`"bogus"`, `30`, `"-1s"`} {
		writeConfig(t, `{"scripts": {"dev": {"tasks": [{"command": "echo hi", "timeout": `+timeout+`}]}}}`)
		if _, err := loadConfig(); err == nil {
			t.Errorf("expected error for timeout %s", timeout)
		}
	}
}

func TestLoadConfigNegativeRetries(t *testing.T) {
	writeConfig(t, `{"scripts": {"dev": {"tasks": [{"command": "echo hi", "retries": -1}]}}}`)
	if _, err := loadConfig(); err == nil {
		t.Error("expected error for negative retries")
	}
}

func TestLoadConfigDirectoryStringForm(t *testing.T) {
	writeConfig(t, `{
		"directories": {"web": "./frontend", "api": {"alias": "A", "path": "./backend"}},
		"scripts": {}
	}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if config.Directories["web"] != (Directory{Path: "./frontend"}) {
		t.Errorf("web: got %+v", config.Directories["web"])
	}
	// Legacy object form still works, alias included
	if config.Directories["api"] != (Directory{Alias: "A", Path: "./backend"}) {
		t.Errorf("api: got %+v", config.Directories["api"])
	}
}

func TestLoadConfigScriptMapForm(t *testing.T) {
	// Directory-to-command map; file order must be preserved
	writeConfig(t, `{"scripts": {"dev": {
		"backend": "go run .",
		"frontend": {"linux": "npm run dev", "default": "npm run dev"},
		"docs": {"command": "make serve", "timeout": "30s", "retries": 1, "env": {"PORT": "8080"}}
	}}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	tasks := config.Scripts["dev"].Tasks
	if len(tasks) != 3 {
		t.Fatalf("tasks: got %v", tasks)
	}
	if tasks[0].Dir != "backend" || tasks[0].Command != "go run ." {
		t.Errorf("task 0: got %+v", tasks[0])
	}
	if tasks[1].Dir != "frontend" || tasks[1].Command != "npm run dev" {
		t.Errorf("task 1: got %+v", tasks[1])
	}
	docs := tasks[2]
	if docs.Dir != "docs" || docs.Command != "make serve" || docs.Timeout != Duration(30*time.Second) || docs.Retries != 1 || docs.Env["PORT"] != "8080" {
		t.Errorf("task 2: got %+v", docs)
	}
}

func TestLoadConfigScriptTasksObjectForm(t *testing.T) {
	writeConfig(t, `{"scripts": {"test": {
		"mode": "sequential",
		"maxParallel": 2,
		"tasks": {
			"frontend": "npm test",
			"backend": {"command": "go test ./...", "retries": 1}
		}
	}}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	group := config.Scripts["test"]
	if group.Mode != "sequential" || group.MaxParallel != 2 {
		t.Errorf("group: got %+v", group)
	}
	if len(group.Tasks) != 2 || group.Tasks[0].Dir != "frontend" || group.Tasks[1].Retries != 1 {
		t.Errorf("tasks: got %+v", group.Tasks)
	}
}

func TestLoadConfigTaskNameAndDirSeparated(t *testing.T) {
	// Keys are task names; "dir" picks the directory, several tasks may
	// share one, and the file order is preserved
	writeConfig(t, `{"scripts": {"check": {
		"mode": "parallel",
		"tasks": {
			"frontend-lint": {"dir": "frontend", "command": "npm run lint"},
			"frontend-typecheck": {"dir": "frontend", "command": "npm run typecheck"},
			"backend-test": {"dir": "backend", "command": "go test ./..."}
		}
	}}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	tasks := config.Scripts["check"].Tasks
	want := []Task{
		{Name: "frontend-lint", Dir: "frontend", Command: "npm run lint"},
		{Name: "frontend-typecheck", Dir: "frontend", Command: "npm run typecheck"},
		{Name: "backend-test", Dir: "backend", Command: "go test ./..."},
	}
	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("tasks: got %+v, want %+v", tasks, want)
	}
}

func TestLoadConfigTaskDirDefaultsToName(t *testing.T) {
	// Without "dir" the key is both the name and the directory, in the
	// string form, the object form, and the shorthand map form
	writeConfig(t, `{"scripts": {
		"dev": {"frontend": "npm run dev"},
		"test": {"tasks": {
			"frontend": {"command": "npm test", "timeout": "2m"},
			"backend": "go test ./..."
		}}
	}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	dev := config.Scripts["dev"].Tasks
	if len(dev) != 1 || dev[0].Name != "frontend" || dev[0].Dir != "frontend" {
		t.Errorf("dev tasks: got %+v", dev)
	}
	test := config.Scripts["test"].Tasks
	if len(test) != 2 || test[0].Name != "frontend" || test[0].Dir != "frontend" || test[0].Timeout != Duration(2*time.Minute) {
		t.Errorf("test tasks: got %+v", test)
	}
	if test[1].Name != "backend" || test[1].Dir != "backend" {
		t.Errorf("test task 1: got %+v", test[1])
	}
}

func TestLoadConfigShorthandTaskWithDir(t *testing.T) {
	// The shorthand map form also accepts a task whose dir differs from
	// its key, and a "dir" equal to the key stays valid
	writeConfig(t, `{"scripts": {"check": {
		"lint": {"dir": "frontend", "command": "npm run lint"},
		"frontend": {"dir": "frontend", "command": "npm test"}
	}}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	tasks := config.Scripts["check"].Tasks
	if len(tasks) != 2 || tasks[0].Name != "lint" || tasks[0].Dir != "frontend" {
		t.Errorf("task 0: got %+v", tasks)
	}
	if tasks[1].Name != "frontend" || tasks[1].Dir != "frontend" {
		t.Errorf("task 1: got %+v", tasks[1])
	}
}

func TestLoadConfigWatchField(t *testing.T) {
	writeConfig(t, `{"scripts": {
		"dev": {"watch": true, "tasks": {"frontend": "echo dev"}},
		"build": {"tasks": {"frontend": "echo build"}}
	}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if !config.Scripts["dev"].Watch {
		t.Error("dev: Watch not set")
	}
	if config.Scripts["build"].Watch {
		t.Error("build: Watch unexpectedly set")
	}

	// watch without tasks is a pointed error, not a broken task
	writeConfig(t, `{"scripts": {"dev": {"watch": true}}}`)
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), `"watch" requires a "tasks" entry`) {
		t.Errorf("watch without tasks: got %v", err)
	}

	// A task named "watch" in the map form keeps working
	writeConfig(t, `{"scripts": {"dev": {"watch": "npm run watch"}}}`)
	config, err = loadConfig()
	if err != nil {
		t.Fatalf("task named watch: loadConfig failed: %v", err)
	}
	tasks := config.Scripts["dev"].Tasks
	if len(tasks) != 1 || tasks[0].Name != "watch" || tasks[0].Command != "npm run watch" {
		t.Errorf("task named watch: got %+v", tasks)
	}
}

func TestLoadConfigTaskWatch(t *testing.T) {
	writeConfig(t, `{"scripts": {"dev": {"tasks": {
		"frontend": {"command": "npm run dev", "watch": true},
		"backend": "go run ."
	}}}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	tasks := config.Scripts["dev"].Tasks
	if !tasks[0].Watch || tasks[1].Watch {
		t.Errorf("watch flags: got %v / %v, want true / false", tasks[0].Watch, tasks[1].Watch)
	}

	// Restarting individual tasks contradicts sequential order
	writeConfig(t, `{"scripts": {"dev": {"mode": "sequential", "tasks": {
		"frontend": {"command": "npm run dev", "watch": true}
	}}}}`)
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "parallel mode") {
		t.Errorf("sequential + task watch: got %v", err)
	}
}

func TestScriptGroupMarshalTaskWatch(t *testing.T) {
	group := ScriptGroup{Tasks: []Task{{Name: "frontend", Dir: "frontend", Command: "npm run dev", Watch: true}}}
	data, err := json.Marshal(group)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"frontend":{"dir":"frontend","command":"npm run dev","watch":true}}`; string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
	var back ScriptGroup
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("roundtrip failed on %s: %v", data, err)
	}
	if len(back.Tasks) != 1 || !back.Tasks[0].Watch {
		t.Errorf("roundtrip: got %+v", back.Tasks)
	}
}

func TestScriptGroupMarshalWatch(t *testing.T) {
	group := ScriptGroup{Watch: true, Tasks: []Task{{Name: "frontend", Dir: "frontend", Command: "echo dev"}}}
	data, err := json.Marshal(group)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"watch":true,"tasks":{"frontend":"echo dev"}}`; string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
	var back ScriptGroup
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("roundtrip failed on %s: %v", data, err)
	}
	if !back.Watch || len(back.Tasks) != 1 || back.Tasks[0].Name != "frontend" {
		t.Errorf("roundtrip: got %+v", back)
	}
}

func TestScriptGroupMarshalNamedTasks(t *testing.T) {
	group := ScriptGroup{Tasks: []Task{
		{Name: "lint", Dir: "frontend", Command: "npm run lint"},
		{Name: "frontend", Dir: "frontend", Command: "npm run dev"},
	}}
	data, err := json.Marshal(group)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"lint":{"dir":"frontend","command":"npm run lint"},"frontend":"npm run dev"}`; string(data) != want {
		t.Errorf("got %s\nwant %s", data, want)
	}
	var back ScriptGroup
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("roundtrip failed on %s: %v", data, err)
	}
	if !reflect.DeepEqual(back.Tasks, group.Tasks) {
		t.Errorf("roundtrip: got %+v, want %+v", back.Tasks, group.Tasks)
	}
}

func TestLoadConfigScriptRunEverywhere(t *testing.T) {
	writeConfig(t, `{
		"directories": {"b": "./b", "a": "./a"},
		"scripts": {"lint": "golangci-lint run"}
	}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	tasks := config.Scripts["lint"].Tasks
	if len(tasks) != 2 || tasks[0].Dir != "a" || tasks[1].Dir != "b" {
		t.Errorf("tasks: got %+v", tasks)
	}
	for _, task := range tasks {
		if task.Command != "golangci-lint run" {
			t.Errorf("command: got %q", task.Command)
		}
	}
}

func TestLoadConfigScriptRunEverywhereNeedsDirectories(t *testing.T) {
	writeConfig(t, `{"scripts": {"lint": "golangci-lint run"}}`)
	if _, err := loadConfig(); err == nil {
		t.Error("expected error for run-everywhere script without directories")
	}
}

func TestLoadConfigScriptFormErrors(t *testing.T) {
	cases := map[string]string{
		"reserved field without tasks": `{"scripts": {"dev": {"mode": "parallel", "frontend": "npm run dev"}}}`,
		"unknown script field":         `{"scripts": {"dev": {"tasks": [], "bogus": 1}}}`,
		"tasks as string":              `{"scripts": {"dev": {"tasks": "npm run dev"}}}`,
		"tasks as number":              `{"scripts": {"dev": {"tasks": 123}}}`,
		"empty command string":         `{"scripts": {"lint": ""}}`,
		"blank command string":         `{"scripts": {"lint": "   "}}`,
		"null script":                  `{"scripts": {"lint": null}}`,
		"empty task name":              `{"scripts": {"dev": {"tasks": {"": "echo hi"}}}}`,
		"blank task name":              `{"scripts": {"dev": {"  ": "echo hi"}}}`,
		"duplicate task name":          `{"scripts": {"dev": {"tasks": {"a": "echo 1", "a": "echo 2"}}}}`,
	}
	for name, content := range cases {
		writeConfig(t, content)
		if _, err := loadConfig(); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestLoadConfigDirectoryMissingPath(t *testing.T) {
	for _, dirJSON := range []string{`{"alias": "p"}`, `""`, `null`, `"  "`} {
		writeConfig(t, `{"directories": {"production": `+dirJSON+`}, "scripts": {}}`)
		if _, err := loadConfig(); err == nil {
			t.Errorf("expected error for directory value %s", dirJSON)
		}
	}
}

func TestScriptGroupMarshalRoundtrip(t *testing.T) {
	// Simple groups marshal back to the directory-to-command form
	simple := ScriptGroup{Tasks: []Task{{Dir: "frontend", Command: "npm run dev"}}}
	data, err := json.Marshal(simple)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"frontend":"npm run dev"}` {
		t.Errorf("simple: got %s", data)
	}

	// Groups with extras keep the detailed form
	detailed := ScriptGroup{Mode: "sequential", Tasks: []Task{{Dir: "backend", Command: "go test ./...", Retries: 1}}}
	data, err = json.Marshal(detailed)
	if err != nil {
		t.Fatal(err)
	}
	var back ScriptGroup
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("roundtrip failed on %s: %v", data, err)
	}
	if back.Mode != "sequential" || len(back.Tasks) != 1 || back.Tasks[0].Retries != 1 || back.Tasks[0].Dir != "backend" {
		t.Errorf("roundtrip: got %+v from %s", back, data)
	}

	// Run-everywhere shorthand marshals back to a string
	all := ScriptGroup{AllCommand: "npm test"}
	data, err = json.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"npm test"` {
		t.Errorf("all: got %s", data)
	}
}

func TestScriptGroupMarshalReservedDirNames(t *testing.T) {
	// A directory named like a script field must not be emitted in the map
	// form, or it would be misread on reload (tasks → empty, mode → error)
	for _, dir := range []string{"tasks", "mode", "dependsOn", "maxParallel", "watch"} {
		group := ScriptGroup{Tasks: []Task{{Dir: dir, Command: "npm run dev"}}}
		data, err := json.Marshal(group)
		if err != nil {
			t.Fatal(err)
		}
		var back ScriptGroup
		if err := json.Unmarshal(data, &back); err != nil {
			t.Fatalf("dir %q: roundtrip failed on %s: %v", dir, data, err)
		}
		if len(back.Tasks) != 1 || back.Tasks[0].Dir != dir || back.Tasks[0].Command != "npm run dev" {
			t.Errorf("dir %q: roundtrip lost the task: %s → %+v", dir, data, back.Tasks)
		}
	}
}

func TestLoadConfigKeepsSchemaField(t *testing.T) {
	writeConfig(t, `{"$schema": "https://example.com/schema.json", "scripts": {}}`)
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}
	if config.Schema != "https://example.com/schema.json" {
		t.Errorf("Schema: got %q", config.Schema)
	}
}
