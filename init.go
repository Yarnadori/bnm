package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// initProject generates the boilerplate configuration files
func initProject() {
	fmt.Println("[bnm] Initializing project...")

	// Create bnm.json
	jsonFile := "bnm.json"
	if fileExists(jsonFile) {
		fmt.Printf("%s already exists. Overwrite? [y/N]: ", jsonFile)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
			fmt.Println("Aborted.")
			return
		}
	}

	{
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Failed to get current directory: %v\n", err)
			os.Exit(1)
		}
		dirName := filepath.Base(cwd)

		dirs := map[string]Directory{}
		usedAliases := map[string]bool{}

		// Generate a unique alias from a directory name
		assignAlias := func(name string) string {
			runes := []rune(strings.ToUpper(name))
			for i := 1; i <= len(runes); i++ {
				candidate := string(runes[:i])
				if !usedAliases[candidate] {
					usedAliases[candidate] = true
					return candidate
				}
			}
			// Fallback: append a number
			for i := 2; ; i++ {
				candidate := fmt.Sprintf("%s%d", string(runes[0]), i)
				if !usedAliases[candidate] {
					usedAliases[candidate] = true
					return candidate
				}
			}
		}

		// Scan and add subdirectories
		entries, err := os.ReadDir(cwd)
		if err != nil {
			fmt.Printf("Failed to read directory: %v\n", err)
			os.Exit(1)
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			name := entry.Name()
			dirs[strings.ToUpper(name)] = Directory{
				Alias: assignAlias(name),
				Path:  "./" + name,
			}
		}

		config := Config{
			Name:        dirName,
			Version:     "0.0.0",
			Directories: dirs,
			Scripts:     map[string]ScriptGroup{},
		}

		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			fmt.Printf("Failed to create %s: %v\n", jsonFile, err)
			os.Exit(1)
		}

		if err := os.WriteFile(jsonFile, data, 0644); err != nil {
			fmt.Printf("Failed to create %s: %v\n", jsonFile, err)
		} else {
			fmt.Printf("Created %s.\n", jsonFile)
		}
	}

	fmt.Println("\nInitialization complete.")
}

// Helper function to check if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
