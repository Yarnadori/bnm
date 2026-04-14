package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: bnm <command>")
		fmt.Println("Run 'bnm --help' for more information.")
		os.Exit(1)
	}

	command := os.Args[1]

	// Handle the "help" command
	if command == "help" || command == "--help" || command == "-h" {
		fmt.Println("Usage: bnm <command>")
		fmt.Println("  init                      : Initialize (Creates bnm.json)")
		fmt.Println("  exec <dir or alias> <cmd...> : Execute a command in target (use '.' for current directory)")
		fmt.Println("  <script>                  : Execute a script defined in bnm.json (e.g., dev)")
		return
	}

	// Handle the "version" command
	if command == "version" || command == "--version" || command == "-v" {
		fmt.Println("bnm", version)
		return
	}

	// Handle the "init" command
	if command == "init" {
		initProject()
		return
	}

	// Handle the "exec" command
	if command == "exec" {
		if len(os.Args) < 4 {
			fmt.Println("Usage: bnm exec <dir or alias> <command...>")
			fmt.Println("Example: bnm exec -B pnpm add something")
			os.Exit(1)
		}
		taskName := os.Args[2]
		cmdArgs := os.Args[3:]
		runExec(taskName, cmdArgs)
		return
	}

	// Otherwise, treat it as a script execution
	runScript(command)
}
