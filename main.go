package main

import (
	"fmt"
	"os"
	"strings"
)

var version = "dev"

func printUsage() {
	fmt.Println("Usage: bnm <command>")
	fmt.Println("  init                         : Initialize (Creates bnm.json)")
	fmt.Println("  sync                         : Sync directories in bnm.json with current subdirectories")
	fmt.Println("  list                         : List directories and scripts defined in bnm.json")
	fmt.Println("  check                        : Validate bnm.json (paths, aliases, commands, dependencies)")
	fmt.Println("  exec <dir or alias> <cmd...> : Execute a command in target (use '.' for current directory)")
	fmt.Println("  exec --all <cmd...>          : Execute a command in every configured directory")
	fmt.Println("  completion <bash|zsh|fish>   : Print a shell completion script")
	fmt.Println("  <script> [dir...] [-- args]  : Execute a script defined in bnm.json (e.g., dev)")
	fmt.Println()
	fmt.Println("Script options:")
	fmt.Println("  <script> -F FRONTEND         : Run only tasks in the given directories (alias, key, or path)")
	fmt.Println("  <script> --watch             : Rerun the script when files in task directories change")
	fmt.Println("  <script> --dry-run           : Show the execution plan without running anything")
	fmt.Println("  <script> --log-dir DIR       : Also write each task's output to DIR/<script>/<task>.log")
	fmt.Println("  <script> --summary json      : Print the run summary as JSON")
	fmt.Println("  <script> --no-color          : Disable colored output (NO_COLOR is also honored)")
	fmt.Println("  <script> -- --port 3000      : Pass extra arguments to every task command")
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

	case "sync":
		syncProject()

	case "list", "ls":
		runList()

	case "check":
		runCheck()

	case "exec":
		if len(os.Args) >= 4 && os.Args[2] == "--all" {
			runExecAll(os.Args[3:])
			return
		}
		if len(os.Args) < 4 {
			fmt.Println("Usage: bnm exec <dir or alias> <command...>")
			fmt.Println("       bnm exec --all <command...>")
			fmt.Println("Example: bnm exec -B pnpm add something")
			os.Exit(1)
		}
		taskName := os.Args[2]
		cmdArgs := os.Args[3:]
		runExec(taskName, cmdArgs)

	case "completion":
		if len(os.Args) < 3 {
			fmt.Println("Usage: bnm completion <bash|zsh|fish>")
			os.Exit(1)
		}
		printCompletion(os.Args[2])

	case "__complete":
		what := ""
		if len(os.Args) > 2 {
			what = os.Args[2]
		}
		runComplete(what)

	default:
		// Otherwise, treat it as a script execution
		filters, extraArgs, opts, err := splitScriptArgs(os.Args[2:])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		runScript(command, filters, extraArgs, opts)
	}
}

// splitScriptArgs separates script options and directory filters from
// pass-through arguments: everything before "--" is an option or a directory
// filter, everything after is appended to each task command.
func splitScriptArgs(args []string) (filters []string, extraArgs []string, opts scriptOptions, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--":
			return filters, args[i+1:], opts, nil
		case "--watch", "-w":
			opts.Watch = true
			continue
		case "--dry-run", "-n":
			opts.DryRun = true
			continue
		case "--no-color":
			opts.NoColor = true
			continue
		}
		name, value, hasValue := strings.Cut(a, "=")
		takeValue := func() (string, error) {
			if hasValue {
				return value, nil
			}
			i++
			if i >= len(args) || args[i] == "--" {
				return "", fmt.Errorf("%s requires a value", name)
			}
			return args[i], nil
		}
		switch name {
		case "--log-dir":
			if opts.LogDir, err = takeValue(); err != nil {
				return nil, nil, opts, err
			}
		case "--summary":
			v, err := takeValue()
			if err != nil {
				return nil, nil, opts, err
			}
			if v != "text" && v != "json" {
				return nil, nil, opts, fmt.Errorf("--summary must be 'text' or 'json', got '%s'", v)
			}
			opts.Summary = v
		default:
			filters = append(filters, a)
		}
	}
	return filters, nil, opts, nil
}
