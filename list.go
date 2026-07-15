package main

import (
	"fmt"
	"sort"
	"strings"
)

// runList prints the directories and scripts defined in bnm.json
func runList() {
	config := mustLoadConfig()

	if config.Name != "" {
		fmt.Printf("%s", config.Name)
		if config.Version != "" {
			fmt.Printf(" v%s", config.Version)
		}
		fmt.Println()
		fmt.Println()
	}

	fmt.Println("Directories:")
	if len(config.Directories) == 0 {
		fmt.Println("  (none)")
	} else {
		keys := sortedKeys(config.Directories)
		for _, key := range keys {
			dir := config.Directories[key]
			fmt.Printf("  %-12s %-4s %s\n", key, dir.Alias, dir.Path)
		}
	}

	fmt.Println()
	fmt.Println("Scripts:")
	if len(config.Scripts) == 0 {
		fmt.Println("  (none)")
	} else {
		names := sortedKeys(config.Scripts)
		for _, name := range names {
			group := config.Scripts[name]
			mode := group.Mode
			if mode == "" {
				mode = "parallel"
			}
			attrs := mode
			if group.MaxParallel > 0 {
				attrs += fmt.Sprintf(", max %d", group.MaxParallel)
			}
			if len(group.DependsOn) > 0 {
				attrs += ", depends on: " + strings.Join(group.DependsOn, ", ")
			}
			fmt.Printf("  %s (%s)\n", name, attrs)
			nameWidth, dirWidth := 12, 4
			for _, task := range group.Tasks {
				nameWidth = max(nameWidth, len(taskName(task, config)))
				dirWidth = max(dirWidth, len(taskDirLabel(task)))
			}
			for _, task := range group.Tasks {
				fmt.Printf("    %-*s %-*s %s\n", nameWidth, taskName(task, config), dirWidth, taskDirLabel(task), task.Command)
			}
		}
	}
}

// printAvailableScripts lists script names as a hint after a lookup miss
func printAvailableScripts(config *Config) {
	if len(config.Scripts) == 0 {
		fmt.Println("No scripts are defined in bnm.json.")
		return
	}
	fmt.Println("Available scripts:")
	for _, name := range sortedKeys(config.Scripts) {
		fmt.Printf("  %s\n", name)
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
