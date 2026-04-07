package main

import (
	"fmt"
	"os"
)

// initProject generates the boilerplate configuration files
func initProject() {
	fmt.Println("[bnm] Initializing project...")

	// Create bnm.json
	jsonFile := "bnm.json"
	if fileExists(jsonFile) {
		fmt.Printf("%s already exists, skipping creation.\n", jsonFile)
	} else {
		defaultJSON :=
			`{
  "scripts": {
    "dev": [
      {
        "name": "BACKEND",
        "alias": "B",
        "dir": "./backend",
        "command": "echo 'Put your backend command here'"
      },
      {
        "name": "FRONTEND",
        "alias": "F",
        "dir": "./frontend",
        "command": "echo 'Put your frontend command here'"
      }
    ]
  }
}`
		if err := os.WriteFile(jsonFile, []byte(defaultJSON), 0644); err != nil {
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
