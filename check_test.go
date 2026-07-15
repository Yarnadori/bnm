package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckConfigValid(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir(filepath.Join(dir, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := &Config{
		Directories: map[string]Directory{
			"FRONTEND": {Alias: "F", Path: "./frontend"},
		},
		Scripts: map[string]ScriptGroup{
			"dev": {Tasks: []Task{
				{Dir: "FRONTEND", Command: "echo dev"},
				{Command: "echo root"},
			}},
		},
	}
	if problems := checkConfig(config); len(problems) != 0 {
		t.Errorf("expected no problems, got %v", problems)
	}
}

func TestCheckConfigMissingDirectoryPath(t *testing.T) {
	t.Chdir(t.TempDir())
	config := &Config{
		Directories: map[string]Directory{
			"FRONTEND": {Alias: "F", Path: "./frontend"},
		},
		Scripts: map[string]ScriptGroup{},
	}
	problems := checkConfig(config)
	if len(problems) != 1 || !strings.Contains(problems[0], "./frontend") {
		t.Errorf("got %v, want missing path problem", problems)
	}
}

func TestCheckConfigDuplicateAlias(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	for _, d := range []string{"a", "b"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	config := &Config{
		Directories: map[string]Directory{
			"A": {Alias: "x", Path: "./a"},
			"B": {Alias: "X", Path: "./b"},
		},
		Scripts: map[string]ScriptGroup{},
	}
	problems := checkConfig(config)
	if len(problems) != 1 || !strings.Contains(problems[0], "alias") {
		t.Errorf("got %v, want duplicate alias problem", problems)
	}
}

func TestCheckConfigAliasKeyCollision(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	for _, d := range []string{"a", "b"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	config := &Config{
		Directories: map[string]Directory{
			"z":     {Path: "./a"},
			"OTHER": {Alias: "Z", Path: "./b"},
		},
		Scripts: map[string]ScriptGroup{},
	}
	problems := checkConfig(config)
	if len(problems) != 1 || !strings.Contains(problems[0], "collides") {
		t.Errorf("got %v, want alias-name collision problem", problems)
	}

	// An alias equal to its own key is harmless
	config.Directories = map[string]Directory{"web": {Alias: "WEB", Path: "./a"}}
	if problems := checkConfig(config); len(problems) != 0 {
		t.Errorf("self alias: got %v, want none", problems)
	}

	// An alias colliding with another directory's path is reported
	config.Directories = map[string]Directory{
		"X": {Alias: "b", Path: "./a"},
		"Y": {Path: "./b"},
	}
	problems = checkConfig(config)
	if len(problems) != 1 || !strings.Contains(problems[0], "path") {
		t.Errorf("got %v, want alias-path collision problem", problems)
	}
}

func TestHasCheckScript(t *testing.T) {
	// A "check" script routes "bnm check" to the script runner
	writeConfig(t, `{"scripts": {"check": {"frontend": "echo hi"}}}`)
	if !hasCheckScript() {
		t.Error("expected true when a check script is defined")
	}

	// No check script, a broken config, and a missing config all fall
	// through to the validator
	writeConfig(t, `{"scripts": {"dev": {"frontend": "echo hi"}}}`)
	if hasCheckScript() {
		t.Error("expected false without a check script")
	}
	writeConfig(t, `{broken`)
	if hasCheckScript() {
		t.Error("expected false for a broken config")
	}
	t.Chdir(t.TempDir())
	if hasCheckScript() {
		t.Error("expected false for a missing config")
	}
}

func TestCheckConfigNamedTasks(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir(filepath.Join(dir, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := &Config{
		Directories: map[string]Directory{
			"FRONTEND": {Alias: "F", Path: "./frontend"},
		},
		Scripts: map[string]ScriptGroup{
			"check": {Tasks: []Task{
				// A name differing from the dir is fine, and the dir may be
				// a key, an alias, or a path
				{Name: "lint", Dir: "FRONTEND", Command: "echo lint"},
				{Name: "typecheck", Dir: "F", Command: "echo typecheck"},
				{Name: "test", Dir: "./frontend", Command: "echo test"},
			}},
		},
	}
	if problems := checkConfig(config); len(problems) != 0 {
		t.Errorf("expected no problems, got %v", problems)
	}
}

func TestCheckConfigDuplicateTaskName(t *testing.T) {
	t.Chdir(t.TempDir())
	config := &Config{
		Scripts: map[string]ScriptGroup{
			"check": {Tasks: []Task{
				{Name: "lint", Command: "echo 1"},
				{Name: "lint", Command: "echo 2"},
			}},
		},
	}
	problems := checkConfig(config)
	if len(problems) != 1 || !strings.Contains(problems[0], "share the name 'lint'") {
		t.Errorf("got %v, want duplicate task name problem", problems)
	}

	// Names differing only in case are also duplicates: task filters match
	// case-insensitively, so --task would select both
	config.Scripts["check"] = ScriptGroup{Tasks: []Task{
		{Name: "Lint", Command: "echo 1"},
		{Name: "lint", Command: "echo 2"},
	}}
	problems = checkConfig(config)
	if len(problems) != 1 || !strings.Contains(problems[0], "share a name up to case") {
		t.Errorf("case-only duplicate: got %v", problems)
	}

	// Unnamed legacy tasks may share a directory without complaint
	config.Scripts["check"] = ScriptGroup{Tasks: []Task{
		{Command: "echo 1"},
		{Command: "echo 2"},
	}}
	if problems := checkConfig(config); len(problems) != 0 {
		t.Errorf("legacy tasks: got %v, want none", problems)
	}
}

func TestCheckConfigAliasDirResolves(t *testing.T) {
	t.Chdir(t.TempDir())
	config := &Config{
		Directories: map[string]Directory{
			"FRONTEND": {Alias: "F", Path: "./frontend"},
		},
		Scripts: map[string]ScriptGroup{
			"dev": {Tasks: []Task{{Name: "lint", Dir: "F", Command: "echo hi"}}},
		},
	}
	// ./frontend does not exist: both the directory entry and the task's
	// alias-resolved dir must be reported
	problems := checkConfig(config)
	if len(problems) != 2 {
		t.Errorf("got %v, want 2 problems", problems)
	}
}

func TestCheckConfigTaskProblems(t *testing.T) {
	t.Chdir(t.TempDir())
	config := &Config{
		Scripts: map[string]ScriptGroup{
			"dev": {Tasks: []Task{
				{Command: ""},                          // no command for this OS
				{Dir: "./missing", Command: "echo hi"}, // unresolvable dir
			}},
		},
	}
	problems := checkConfig(config)
	if len(problems) != 2 {
		t.Fatalf("got %v, want 2 problems", problems)
	}
	if !strings.Contains(problems[0], "no command") {
		t.Errorf("problems[0]: got %q", problems[0])
	}
	if !strings.Contains(problems[1], "./missing") {
		t.Errorf("problems[1]: got %q", problems[1])
	}
}
