package main

import (
	"os"
	"sort"
	"strings"
)

// Dependency and build-output directories that are never task targets
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"out":          true,
	"target":       true,
	"__pycache__":  true,
}

// scanDirectories finds subdirectories of root and returns them keyed by
// their name. Entries from existing that already point at a found path are
// kept as-is (key and alias included), so sync preserves customizations.
func scanDirectories(root string, existing map[string]Directory) (map[string]Directory, error) {
	pathToKey := map[string]string{}
	for key, dir := range existing {
		pathToKey[dir.Path] = key
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	dirs := map[string]Directory{}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || skipDirs[entry.Name()] {
			continue
		}
		name := entry.Name()
		path := "./" + name
		if key, ok := pathToKey[path]; ok {
			dirs[key] = existing[key]
			continue
		}
		dirs[name] = Directory{Path: path}
	}

	return dirs, nil
}

func diffDirectories(oldDirs, newDirs map[string]Directory) ([]string, []string, []string) {
	var added []string
	var removed []string
	var changed []string

	for key, newDir := range newDirs {
		oldDir, ok := oldDirs[key]
		if !ok {
			added = append(added, key)
		} else if oldDir != newDir {
			changed = append(changed, key)
		}
	}
	for key := range oldDirs {
		if _, ok := newDirs[key]; !ok {
			removed = append(removed, key)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)

	return added, removed, changed
}
