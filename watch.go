package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watchDebounce is how long to wait after the last file event before
// restarting, so bursts of writes (saves, generators) trigger one restart.
const watchDebounce = 300 * time.Millisecond

// watchIgnoreDirs are directory names that are never watched: build output
// and dependency trees change constantly and would restart tasks in a loop.
var watchIgnoreDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	"out":          true,
	"target":       true,
	"vendor":       true,
	"coverage":     true,
	"tmp":          true,
	"__pycache__":  true,
}

// runScriptWatch runs the script, then reruns it whenever a file under any
// task directory changes. It returns when ctx is canceled (Ctrl+C).
func runScriptWatch(ctx context.Context, config *Config, order []string, tasksByScript map[string][]Task, targetScript string, sharedEnv []string, opts scriptOptions) {
	roots := watchRoots(tasksByScript)
	changes, count, err := watchChanges(ctx, roots)
	if err != nil {
		fmt.Printf("Error: failed to start file watcher: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[bnm] Watch mode: watching %d directories. Press Ctrl+C to stop.\n", count)

	for {
		runCtx, cancelRun := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			runScriptOnce(runCtx, config, order, tasksByScript, targetScript, sharedEnv, opts)
			close(done)
		}()

		select {
		case <-changes:
			fmt.Println("[bnm] Change detected. Restarting...")
			cancelRun()
			<-done
		case <-done:
			cancelRun()
			if ctx.Err() != nil {
				return
			}
			fmt.Println("[bnm] Waiting for file changes...")
			select {
			case <-changes:
				fmt.Println("[bnm] Change detected. Restarting...")
			case <-ctx.Done():
				return
			}
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// watchRoots returns the unique directories of every task in the run.
func watchRoots(tasksByScript map[string][]Task) []string {
	seen := map[string]bool{}
	var roots []string
	for _, tasks := range tasksByScript {
		for _, t := range tasks {
			dir := t.Dir
			if dir == "" {
				dir = "."
			}
			if !seen[dir] {
				seen[dir] = true
				roots = append(roots, dir)
			}
		}
	}
	return roots
}

// watchChanges watches the given roots recursively and delivers one signal
// per debounced burst of file events. It also reports how many directories
// are being watched.
func watchChanges(ctx context.Context, roots []string) (<-chan struct{}, int, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, 0, err
	}

	count := 0
	for _, root := range roots {
		count += watchTree(watcher, root)
	}

	changes := make(chan struct{}, 1)
	go func() {
		defer watcher.Close()
		var timer *time.Timer
		var timerC <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 {
					continue
				}
				// Watch directories created while running
				if ev.Op&fsnotify.Create != 0 {
					if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
						watchTree(watcher, ev.Name)
					}
				}
				if timer == nil {
					timer = time.NewTimer(watchDebounce)
					timerC = timer.C
				} else {
					timer.Reset(watchDebounce)
				}
			case <-timerC:
				timer, timerC = nil, nil
				select {
				case changes <- struct{}{}:
				default:
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return changes, count, nil
}

// watchTree adds root and its subdirectories to the watcher, skipping
// ignored and hidden directories. It returns how many were added.
func watchTree(watcher *fsnotify.Watcher, root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if path != root && (watchIgnoreDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		if watcher.Add(path) == nil {
			count++
		}
		return nil
	})
	return count
}
