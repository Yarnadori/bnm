package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

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

func TestWatchTaskDirsRoutesToTasks(t *testing.T) {
	root := t.TempDir()
	front := filepath.Join(root, "frontend")
	back := filepath.Join(root, "backend")
	for _, d := range []string{front, back} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Two watched tasks share the frontend directory; the backend task is
	// not watched and must never be signaled.
	tasks := []Task{
		{Name: "front-lint", Dir: front, Watch: true},
		{Name: "front-dev", Dir: front, Watch: true},
		{Name: "back-dev", Dir: back},
	}

	changes, count, err := watchTaskDirs(t.Context(), tasks, "")
	if err != nil {
		t.Fatalf("watchTaskDirs failed: %v", err)
	}
	if count != 1 {
		t.Errorf("watched %d directories, want 1 (frontend only)", count)
	}

	if err := os.WriteFile(filepath.Join(front, "app.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case idxs := <-changes:
		if want := []int{0, 1}; !reflect.DeepEqual(idxs, want) {
			t.Errorf("signaled tasks %v, want %v", idxs, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no change signal for a write under the watched directory")
	}

	// A change in the unwatched directory produces no signal
	if err := os.WriteFile(filepath.Join(back, "main.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case idxs := <-changes:
		t.Errorf("unexpected signal %v for an unwatched directory", idxs)
	case <-time.After(2 * watchDebounce):
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
