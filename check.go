package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// runCheck validates bnm.json beyond what loading already enforces:
// directory paths must exist, aliases must be unique, task dirs must
// resolve, and every task must have a command for this OS.
func runCheck() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	problems := checkConfig(config)
	if len(problems) == 0 {
		fmt.Printf("[bnm] %s is valid.\n", configFileName)
		return
	}
	fmt.Printf("[bnm] Found %d problem(s) in %s:\n", len(problems), configFileName)
	for _, p := range problems {
		fmt.Printf("  - %s\n", p)
	}
	os.Exit(1)
}

func checkConfig(config *Config) []string {
	var problems []string

	keys := map[string]string{}
	paths := map[string]string{}
	for _, key := range sortedKeys(config.Directories) {
		keys[strings.ToLower(key)] = key
		paths[strings.ToLower(strings.TrimPrefix(config.Directories[key].Path, "./"))] = key
	}
	aliases := map[string]string{}
	for _, key := range sortedKeys(config.Directories) {
		d := config.Directories[key]
		if fi, err := os.Stat(d.Path); err != nil || !fi.IsDir() {
			problems = append(problems, fmt.Sprintf("directory '%s': path '%s' does not exist", key, d.Path))
		}
		if d.Alias == "" {
			continue
		}
		lower := strings.ToLower(d.Alias)
		if other, dup := aliases[lower]; dup {
			problems = append(problems, fmt.Sprintf("directories '%s' and '%s' share alias '%s'", other, key, d.Alias))
		} else {
			aliases[lower] = key
		}
		if other, clash := keys[lower]; clash && !strings.EqualFold(key, d.Alias) {
			problems = append(problems, fmt.Sprintf("alias '%s' of directory '%s' collides with directory '%s' (names take priority)", d.Alias, key, other))
		}
		if other, clash := paths[lower]; clash && other != key {
			problems = append(problems, fmt.Sprintf("alias '%s' of directory '%s' collides with the path of directory '%s' (paths take priority)", d.Alias, key, other))
		}
	}

	for _, name := range sortedKeys(config.Scripts) {
		for i, task := range config.Scripts[name].Tasks {
			if strings.TrimSpace(task.Command.String()) == "" {
				problems = append(problems, fmt.Sprintf("script '%s' task %d: no command for this OS (%s)", name, i+1, runtime.GOOS))
			}
			dir := task.Dir
			if dir == "" || dir == "." {
				continue
			}
			if d, ok := config.Directories[dir]; ok {
				dir = d.Path
			}
			if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
				problems = append(problems, fmt.Sprintf("script '%s' task %d: directory '%s' not found", name, i+1, task.Dir))
			}
		}
	}

	return problems
}
