package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// rawSyncConfig mirrors Config but keeps every section except directories as
// raw JSON, so rewriting the file cannot alter scripts (OS-specific command
// maps, key order, and any detail loadConfig normalizes away).
type rawSyncConfig struct {
	Schema      json.RawMessage      `json:"$schema,omitempty"`
	Name        json.RawMessage      `json:"name,omitempty"`
	Version     json.RawMessage      `json:"version,omitempty"`
	Directories map[string]Directory `json:"directories,omitempty"`
	Scripts     json.RawMessage      `json:"scripts,omitempty"`
}

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

	original, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Failed to read %s: %v\n", configFile, err)
		os.Exit(1)
	}
	var raw rawSyncConfig
	if err := json.Unmarshal(original, &raw); err != nil {
		fmt.Printf("Failed to parse %s: %v\n", configFile, err)
		os.Exit(1)
	}
	raw.Directories = currentDirectories

	data, err := json.MarshalIndent(raw, "", "  ")
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
