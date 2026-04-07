package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: bnm <command>")
		fmt.Println("  init        : Initialize (Creates bnm.json and .env)")
		fmt.Println("  <script>    : Execute a script defined in bnm.json (e.g., dev)")
		os.Exit(1)
	}

	command := os.Args[1]

	// Handle the "init" command
	if command == "init" {
		initProject()
		return
	}

	// Otherwise, treat it as a script execution
	runScript(command)
}