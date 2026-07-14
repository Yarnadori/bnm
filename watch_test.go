package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestUnderPath(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		path string
		want bool
	}{
		{dir, true},
		{filepath.Join(dir, "sub"), true},
		{filepath.Join(dir, "sub", "deep"), true},
		{filepath.Dir(dir), false},
		{dir + "-sibling", false},
	}
	for _, c := range cases {
		if got := underPath(c.path, dir); got != c.want {
			t.Errorf("underPath(%q, %q) = %v, want %v", c.path, dir, got, c.want)
		}
	}
	if underPath(dir, "") {
		t.Error("empty dir must match nothing")
	}
}

func TestWatchTreeSkipsIgnoredAndLogDir(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"src", "node_modules", ".git", "logs", filepath.Join("logs", "dev")} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	logAbs, _ := filepath.Abs(filepath.Join(root, "logs"))
	// Expected: root and src only — node_modules and .git are always
	// ignored, and the log directory subtree is excluded.
	if count := watchTree(watcher, root, logAbs); count != 2 {
		t.Errorf("watched %d directories, want 2 (root and src)", count)
	}
}
