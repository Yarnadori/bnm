package main

import (
	"fmt"
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

func scanDirectories(root string, existing map[string]Directory) (map[string]Directory, error) {
	dirs := map[string]Directory{}
	usedAliases := map[string]bool{}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var newEntries []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || skipDirs[entry.Name()] {
			continue
		}

		name := entry.Name()
		key := strings.ToUpper(name)
		path := "./" + name
		if dir, ok := existing[key]; ok && dir.Path == path {
			dirs[key] = dir
			if dir.Alias != "" {
				usedAliases[strings.ToUpper(dir.Alias)] = true
			}
			continue
		}

		newEntries = append(newEntries, name)
	}

	for _, name := range newEntries {
		key := strings.ToUpper(name)
		dirs[key] = Directory{
			Alias: assignAlias(name, usedAliases),
			Path:  "./" + name,
		}
	}

	return dirs, nil
}

func assignAlias(name string, usedAliases map[string]bool) string {
	runes := []rune(strings.ToUpper(name))
	for i := 1; i <= len(runes); i++ {
		candidate := string(runes[:i])
		if !usedAliases[candidate] {
			usedAliases[candidate] = true
			return candidate
		}
	}

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", string(runes[0]), i)
		if !usedAliases[candidate] {
			usedAliases[candidate] = true
			return candidate
		}
	}
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
