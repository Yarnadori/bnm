package main

import (
	"fmt"
	"os"
	"strings"
)

var version = "dev"

func printUsage() {
	fmt.Println("Usage: bnm <command>")
	fmt.Println("  init                         : Initialize (Creates bnm.json; see 'init --yes/--force/--dry-run/--include/--exclude')")
	fmt.Println("  sync                         : Sync directories in bnm.json with current subdirectories")
	fmt.Println("  list                         : List directories and scripts defined in bnm.json")
	fmt.Println("  doctor                       : Validate bnm.json (paths, aliases, commands, dependencies)")
	fmt.Println("  check                        : Same as doctor, unless a \"check\" script is defined — then it runs that script")
	fmt.Println("  exec <dir or alias> <cmd...> : Execute a command in target (use '.' for current directory)")
	fmt.Println("  exec --all <cmd...>          : Execute a command in every configured directory")
	fmt.Println("  completion <bash|zsh|fish>   : Print a shell completion script")
	fmt.Println("  <script> [dir...] [-- args]  : Execute a script defined in bnm.json (e.g., dev)")
	fmt.Println()
	fmt.Println("Script options:")
	fmt.Println("  <script> frontend backend    : Run only tasks in the given directories (name, alias, or path)")
	fmt.Println("  <script> --filter frontend   : Same as above; -F is the short form and can repeat")
	fmt.Println("  <script> --task lint         : Run only the task with this name; -T is the short form and can repeat")
	fmt.Println("  <script> --watch             : Rerun the script when files in task directories change")
	fmt.Println("  <script> --no-watch          : Run once, overriding \"watch\": true in bnm.json")
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
		initProject(os.Args[2:])

	case "sync":
		syncProject()

	case "list", "ls":
		runList()

	case "doctor":
		runCheck()

	case "check":
		// Historically the validator; a config that defines a "check"
		// script gets the script instead (the validator stays reachable
		// as "bnm doctor").
		runCheckCommand(os.Args[2:])

	case "exec":
		if len(os.Args) >= 4 && os.Args[2] == "--all" {
			runExecAll(os.Args[3:])
			return
		}
		if len(os.Args) < 4 {
			fmt.Println("Usage: bnm exec <dir or alias> <command...>")
			fmt.Println("       bnm exec --all <command...>")
			fmt.Println("Example: bnm exec backend pnpm add something")
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
		if a == "--" {
			extraArgs = args[i+1:]
			break
		}
		switch a {
		case "--watch", "-w":
			opts.Watch = true
			continue
		case "--no-watch":
			opts.NoWatch = true
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
		case "--filter", "-F":
			v, err := takeValue()
			if err != nil {
				return nil, nil, opts, err
			}
			filters = append(filters, v)
		case "--task", "-T":
			v, err := takeValue()
			if err != nil {
				return nil, nil, opts, err
			}
			opts.TaskFilters = append(opts.TaskFilters, v)
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
	if opts.Watch && opts.NoWatch {
		return nil, nil, opts, fmt.Errorf("--watch and --no-watch cannot be combined")
	}
	return filters, extraArgs, opts, nil
}
