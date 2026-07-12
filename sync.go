package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func syncProject() {
	configFile := configFileName
	config := mustLoadConfig()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	currentDirectories, err := scanDirectories(cwd, config.Directories)
	if err != nil {
		fmt.Printf("Failed to read directory: %v\n", err)
		os.Exit(1)
	}

	added, removed, changed := diffDirectories(config.Directories, currentDirectories)
	if len(added) == 0 && len(removed) == 0 && len(changed) == 0 {
		fmt.Println("[bnm] bnm.json is already in sync.")
		return
	}

	config.Directories = currentDirectories
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("Failed to update %s: %v\n", configFile, err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		fmt.Printf("Failed to update %s: %v\n", configFile, err)
		os.Exit(1)
	}

	for _, key := range added {
		fmt.Printf("[bnm] Added directory: %s (%s)\n", key, currentDirectories[key].Path)
	}
	for _, key := range removed {
		fmt.Printf("[bnm] Removed directory: %s\n", key)
	}
	for _, key := range changed {
		fmt.Printf("[bnm] Updated directory: %s (%s)\n", key, currentDirectories[key].Path)
	}
	fmt.Printf("[bnm] Updated %s.\n", configFile)
}
