package main

import (
	"fmt"
	"os"
)

var version = "dev"

func printUsage() {
	fmt.Println("Usage: bnm <command>")
	fmt.Println("  init                         : Initialize (Creates bnm.json)")
	fmt.Println("  list                         : List directories and scripts defined in bnm.json")
	fmt.Println("  exec <dir or alias> <cmd...> : Execute a command in target (use '.' for current directory)")
	fmt.Println("  <script>                     : Execute a script defined in bnm.json (e.g., dev)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help      Show this help")
	fmt.Println("  -v, --version   Show version")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: bnm <command>")
		fmt.Println("Run 'bnm --help' for more information.")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "help", "--help", "-h":
		printUsage()

	case "version", "--version", "-v":
		fmt.Println("bnm", version)

	case "init":
		initProject()

	case "list", "ls":
		runList()

	case "exec":
		if len(os.Args) < 4 {
			fmt.Println("Usage: bnm exec <dir or alias> <command...>")
			fmt.Println("Example: bnm exec -B pnpm add something")
			os.Exit(1)
		}
		taskName := os.Args[2]
		cmdArgs := os.Args[3:]
		runExec(taskName, cmdArgs)

	default:
		// Otherwise, treat it as a script execution
		runScript(command)
	}
}
