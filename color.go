package main

import (
	"os"
	"sync"
)

// ANSI colors cycled through task prefixes so parallel output is easy to tell apart
var taskColors = []string{
	"\033[36m", // cyan
	"\033[33m", // yellow
	"\033[32m", // green
	"\033[35m", // magenta
	"\033[34m", // blue
	"\033[31m", // red
}

const colorReset = "\033[0m"

var (
	colorMu      sync.Mutex
	colorIndex   int
	nameColors   = map[string]string{}
	colorEnabled = isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == ""
)

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// colorFor returns a stable color prefix/reset pair for a task name,
// or empty strings when colors are disabled (non-TTY or NO_COLOR).
func colorFor(name string) (string, string) {
	if !colorEnabled {
		return "", ""
	}
	colorMu.Lock()
	defer colorMu.Unlock()
	c, ok := nameColors[name]
	if !ok {
		c = taskColors[colorIndex%len(taskColors)]
		colorIndex++
		nameColors[name] = c
	}
	return c, colorReset
}
