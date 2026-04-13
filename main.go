package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: bnm <command>")
		fmt.Println("  init                      : Initialize (Creates bnm.json)")
		fmt.Println("  exec <dir or alias> <cmd...> : Execute a command in target (use '.' for current directory)")
		fmt.Println("  <script>                  : Execute a script defined in bnm.json (e.g., dev)")
		os.Exit(1)
	}

	command := os.Args[1]

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
