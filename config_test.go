package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
