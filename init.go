package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type initOptions struct {
	Yes     bool
	Force   bool
	DryRun  bool
	Include map[string]bool // lowercased directory names; nil means all
	Exclude map[string]bool
}

// commandHints maps a marker file to suggested dev/test commands.
var commandHints = []struct {
	Marker string
	Dev    string
	Test   string
}{
	{"package.json", "npm run dev", "npm test"},
	{"go.mod", "go run .", "go test ./..."},
	{"Cargo.toml", "cargo run", "cargo test"},
	{"pyproject.toml", "", "pytest"},
}

// initProject generates bnm.json: it scans subdirectories, asks which to
// include and which detected commands to use (interactive on a terminal,
// automatic with --yes or without one), and refuses to overwrite an
// existing config unless --force is given.
func initProject(args []string) {
	opts, err := parseInitArgs(args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Usage: bnm init [--yes] [--force] [--dry-run] [--include a,b] [--exclude a,b]")
		os.Exit(1)
	}

	if fileExists(configFileName) && !opts.Force && !opts.DryRun {
		fmt.Printf("%s already exists.\n", configFileName)
		fmt.Println("Use --force to overwrite or 'bnm sync' to update directories.")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	found, err := scanDirectories(cwd, nil)
	if err != nil {
		fmt.Printf("Failed to read directory: %v\n", err)
		os.Exit(1)
	}

	interactive := !opts.Yes && isTerminal(os.Stdin)
	reader := bufio.NewReader(os.Stdin)

	var names []string
	for _, name := range sortedKeys(found) {
		lower := strings.ToLower(name)
		if opts.Include != nil && !opts.Include[lower] {
			continue
		}
		if opts.Exclude[lower] {
			continue
		}
		if interactive && !promptYesNo(reader, fmt.Sprintf("Include '%s'?", name), true) {
			continue
		}
		names = append(names, name)
	}

	config := Config{
		Schema:  schemaURL,
		Name:    filepath.Base(cwd),
		Version: "0.0.0",
		Scripts: map[string]ScriptGroup{},
	}
	if len(names) > 0 {
		config.Directories = map[string]Directory{}
	}

	devTasks := map[string]string{}
	testTasks := map[string]string{}
	for _, name := range names {
		config.Directories[name] = found[name]
		marker, dev, test := detectCommands(found[name].Path)
		if marker == "" {
			continue
		}
		if interactive {
			fmt.Printf("Detected %s in %s.\n", marker, name)
			if dev != "" && !promptYesNo(reader, fmt.Sprintf("Use %q for the dev task?", dev), true) {
				dev = ""
			}
			if test != "" && !promptYesNo(reader, fmt.Sprintf("Use %q for the test task?", test), true) {
				test = ""
			}
		}
		if dev != "" {
			devTasks[name] = dev
		}
		if test != "" {
			testTasks[name] = test
		}
	}
	for scriptName, tasks := range map[string]map[string]string{"dev": devTasks, "test": testTasks} {
		if len(tasks) == 0 {
			continue
		}
		group := ScriptGroup{}
		for _, dir := range sortedKeys(tasks) {
			group.Tasks = append(group.Tasks, Task{Dir: dir, Command: Command(tasks[dir])})
		}
		config.Scripts[scriptName] = group
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Printf("Failed to create %s: %v\n", configFileName, err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if opts.DryRun {
		fmt.Printf("%s", data)
		fmt.Printf("[bnm] Dry run: %s was not written.\n", configFileName)
		return
	}

	if opts.Force && fileExists(configFileName) {
		// Never overwrite without a confirmed backup
		if err := backupExistingConfig(); err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("Aborted; %s was not modified.\n", configFileName)
			os.Exit(1)
		}
		fmt.Printf("[bnm] Backed up existing config to %s.bak.\n", configFileName)
	}

	if err := os.WriteFile(configFileName, data, 0o644); err != nil {
		fmt.Printf("Failed to create %s: %v\n", configFileName, err)
		os.Exit(1)
	}
	fmt.Printf("Created %s with %d directories.\n", configFileName, len(names))
}

func parseInitArgs(args []string) (initOptions, error) {
	opts := initOptions{Exclude: map[string]bool{}}
	for i := 0; i < len(args); i++ {
		name, value, hasValue := strings.Cut(args[i], "=")
		takeValue := func() (string, error) {
			if hasValue {
				return value, nil
			}
			i++
			if i >= len(args) {
				return "", fmt.Errorf("%s requires a value", name)
			}
			return args[i], nil
		}
		switch name {
		case "--yes", "-y":
			opts.Yes = true
		case "--force":
			opts.Force = true
		case "--dry-run", "-n":
			opts.DryRun = true
		case "--include":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.Include = nameSet(v, opts.Include)
		case "--exclude":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.Exclude = nameSet(v, opts.Exclude)
		default:
			return opts, fmt.Errorf("unknown option '%s'", args[i])
		}
	}
	return opts, nil
}

// backupExistingConfig copies bnm.json to bnm.json.bak, reporting any
// failure so the caller can abort instead of overwriting the original.
func backupExistingConfig() error {
	old, err := os.ReadFile(configFileName)
	if err != nil {
		return fmt.Errorf("failed to read existing %s: %w", configFileName, err)
	}
	backup := configFileName + ".bak"
	if err := os.WriteFile(backup, old, 0o644); err != nil {
		return fmt.Errorf("failed to back up existing config to %s: %w", backup, err)
	}
	return nil
}

// nameSet adds the comma-separated names to the set, lowercased.
func nameSet(list string, set map[string]bool) map[string]bool {
	if set == nil {
		set = map[string]bool{}
	}
	for name := range strings.SplitSeq(list, ",") {
		if name = strings.TrimSpace(name); name != "" {
			set[strings.ToLower(name)] = true
		}
	}
	return set
}

// detectCommands looks for a project marker file in dir and returns it with
// the suggested dev and test commands.
func detectCommands(dir string) (marker, dev, test string) {
	for _, hint := range commandHints {
		if fileExists(filepath.Join(dir, hint.Marker)) {
			return hint.Marker, hint.Dev, hint.Test
		}
	}
	return "", "", ""
}

// promptYesNo asks a yes/no question and returns the answer, using def for
// an empty reply.
func promptYesNo(reader *bufio.Reader, question string, def bool) bool {
	suffix := "[Y/n]"
	if !def {
		suffix = "[y/N]"
	}
	fmt.Printf("%s %s: ", question, suffix)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return def
	}
}

// Helper function to check if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
