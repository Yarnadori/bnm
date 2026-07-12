package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInitProjectScansDirectories(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	for _, d := range []string{"frontend", "backend", ".git", "node_modules", "dist"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	initProject()

	data, err := os.ReadFile(configFileName)
	if err != nil {
		t.Fatalf("bnm.json was not created: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("generated bnm.json is invalid: %v", err)
	}

	if _, ok := config.Directories["FRONTEND"]; !ok {
		t.Error("FRONTEND entry missing")
	}
	if _, ok := config.Directories["BACKEND"]; !ok {
		t.Error("BACKEND entry missing")
	}
	for _, excluded := range []string{".GIT", "NODE_MODULES", "DIST"} {
		if _, ok := config.Directories[excluded]; ok {
			t.Errorf("%s should be excluded from scan", excluded)
		}
	}

	if config.Directories["FRONTEND"].Path != "./frontend" {
		t.Errorf("FRONTEND path: got %q, want %q", config.Directories["FRONTEND"].Path, "./frontend")
	}

	// Aliases must be unique
	seen := map[string]bool{}
	for key, d := range config.Directories {
		if d.Alias == "" {
			t.Errorf("%s has empty alias", key)
		}
		if seen[d.Alias] {
			t.Errorf("duplicate alias %q", d.Alias)
		}
		seen[d.Alias] = true
	}
}

func TestInitProjectAliasCollision(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Both start with "FRONT" — aliases must not collide
	for _, d := range []string{"frontend", "frontoffice"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	initProject()

	data, err := os.ReadFile(configFileName)
	if err != nil {
		t.Fatal(err)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}

	a1 := config.Directories["FRONTEND"].Alias
	a2 := config.Directories["FRONTOFFICE"].Alias
	if a1 == a2 {
		t.Errorf("aliases collide: %q", a1)
	}
}
