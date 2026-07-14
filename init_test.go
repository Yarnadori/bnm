package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupProjectDirs(t *testing.T, dirs ...string) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	for _, d := range dirs {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func readGeneratedConfig(t *testing.T) *Config {
	t.Helper()
	data, err := os.ReadFile(configFileName)
	if err != nil {
		t.Fatalf("bnm.json was not created: %v", err)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("generated bnm.json is invalid: %v", err)
	}
	return &config
}

func TestInitProjectScansDirectories(t *testing.T) {
	root := setupProjectDirs(t, "frontend", "backend", ".git", "node_modules", "dist")
	if err := os.WriteFile(filepath.Join(root, "frontend", "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend", "go.mod"), []byte("module backend"), 0o644); err != nil {
		t.Fatal(err)
	}

	// stdin is not a terminal in tests, so init runs non-interactively
	initProject(nil)

	config := readGeneratedConfig(t)
	for _, key := range []string{"frontend", "backend"} {
		if config.Directories[key].Path != "./"+key {
			t.Errorf("%s path: got %q, want %q", key, config.Directories[key].Path, "./"+key)
		}
	}
	for _, excluded := range []string{".git", "node_modules", "dist"} {
		if _, ok := config.Directories[excluded]; ok {
			t.Errorf("%s should be excluded from scan", excluded)
		}
	}

	// Detected commands become dev and test scripts
	dev := config.Scripts["dev"]
	want := map[string]string{"backend": "go run .", "frontend": "npm run dev"}
	if len(dev.Tasks) != 2 {
		t.Fatalf("dev tasks: got %v", dev.Tasks)
	}
	for _, task := range dev.Tasks {
		if want[task.Dir] != task.Command.String() {
			t.Errorf("dev task %s: got %q, want %q", task.Dir, task.Command, want[task.Dir])
		}
	}
	if len(config.Scripts["test"].Tasks) != 2 {
		t.Errorf("test tasks: got %v", config.Scripts["test"].Tasks)
	}
}

func TestInitProjectIncludeExclude(t *testing.T) {
	setupProjectDirs(t, "frontend", "backend", "docs")

	initProject([]string{"--include", "frontend,backend", "--exclude", "backend"})

	config := readGeneratedConfig(t)
	if len(config.Directories) != 1 {
		t.Fatalf("directories: got %v", config.Directories)
	}
	if _, ok := config.Directories["frontend"]; !ok {
		t.Error("frontend entry missing")
	}
}

func TestInitProjectDryRun(t *testing.T) {
	setupProjectDirs(t, "frontend")

	initProject([]string{"--dry-run"})

	if fileExists(configFileName) {
		t.Error("dry run must not write bnm.json")
	}
}

func TestInitProjectForceBacksUp(t *testing.T) {
	setupProjectDirs(t, "frontend")
	if err := os.WriteFile(configFileName, []byte(`{"scripts": {}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	initProject([]string{"--force"})

	backup, err := os.ReadFile(configFileName + ".bak")
	if err != nil {
		t.Fatalf("backup was not created: %v", err)
	}
	if string(backup) != `{"scripts": {}}` {
		t.Errorf("backup content: got %q", backup)
	}
	config := readGeneratedConfig(t)
	if _, ok := config.Directories["frontend"]; !ok {
		t.Error("config was not regenerated")
	}
}

func TestBackupExistingConfigFailure(t *testing.T) {
	setupProjectDirs(t, "frontend")
	if err := os.WriteFile(configFileName, []byte(`{"scripts": {}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// A directory squatting on the backup path must make the backup fail
	if err := os.Mkdir(configFileName+".bak", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := backupExistingConfig(); err == nil {
		t.Error("expected error when the backup cannot be written")
	}
}

func TestParseInitArgs(t *testing.T) {
	opts, err := parseInitArgs([]string{"--yes", "--force", "--dry-run", "--include=a,b", "--exclude", "c"})
	if err != nil {
		t.Fatalf("parseInitArgs failed: %v", err)
	}
	if !opts.Yes || !opts.Force || !opts.DryRun {
		t.Errorf("flags: got %+v", opts)
	}
	if !opts.Include["a"] || !opts.Include["b"] || !opts.Exclude["c"] {
		t.Errorf("sets: got %+v", opts)
	}

	for _, args := range [][]string{{"--bogus"}, {"--include"}} {
		if _, err := parseInitArgs(args); err == nil {
			t.Errorf("parseInitArgs(%v): expected error", args)
		}
	}
}

func TestDetectCommands(t *testing.T) {
	dir := t.TempDir()
	if marker, _, _ := detectCommands(dir); marker != "" {
		t.Errorf("empty dir: got marker %q", marker)
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0o644); err != nil {
		t.Fatal(err)
	}
	marker, dev, test := detectCommands(dir)
	if marker != "go.mod" || dev != "go run ." || test != "go test ./..." {
		t.Errorf("got %q / %q / %q", marker, dev, test)
	}
}
