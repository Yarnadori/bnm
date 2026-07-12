package main

import (
	"fmt"
	"os"
)

var builtinCommands = []string{"init", "sync", "list", "exec", "completion", "help", "version"}

const bashCompletion = `_bnm_completions() {
  local cur=${COMP_WORDS[COMP_CWORD]}
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "$(bnm __complete commands 2>/dev/null)" -- "$cur") )
  elif [ "${COMP_WORDS[1]}" = "exec" ] && [ "$COMP_CWORD" -eq 2 ]; then
    COMPREPLY=( $(compgen -W "$(bnm __complete dirs 2>/dev/null)" -- "$cur") )
  fi
}
complete -F _bnm_completions bnm
`

const zshCompletion = `#compdef bnm
_bnm() {
  local -a items
  if (( CURRENT == 2 )); then
    items=(${(f)"$(bnm __complete commands 2>/dev/null)"})
    _describe 'command' items
  elif [[ ${words[2]} == exec ]] && (( CURRENT == 3 )); then
    items=(${(f)"$(bnm __complete dirs 2>/dev/null)"})
    _describe 'directory' items
  fi
}
compdef _bnm bnm
`

const fishCompletion = `complete -c bnm -f
complete -c bnm -n "test (count (commandline -opc)) -eq 1" -a "(bnm __complete commands 2>/dev/null)"
complete -c bnm -n "test (count (commandline -opc)) -eq 2; and test (commandline -opc)[2] = exec" -a "(bnm __complete dirs 2>/dev/null)"
`

// printCompletion prints the completion script for the given shell.
// Install with e.g. `source <(bnm completion bash)` in ~/.bashrc.
func printCompletion(shell string) {
	switch shell {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		fmt.Printf("Error: Unsupported shell '%s' (expected bash, zsh, or fish).\n", shell)
		os.Exit(1)
	}
}

// runComplete backs the shell completion scripts: it prints candidate words
// one per line and stays silent when bnm.json is missing or broken.
func runComplete(what string) {
	config, err := loadConfig()
	switch what {
	case "commands":
		for _, c := range builtinCommands {
			fmt.Println(c)
		}
		if err == nil {
			for _, name := range sortedKeys(config.Scripts) {
				fmt.Println(name)
			}
		}
	case "scripts":
		if err == nil {
			for _, name := range sortedKeys(config.Scripts) {
				fmt.Println(name)
			}
		}
	case "dirs":
		if err == nil {
			fmt.Println(".")
			for _, key := range sortedKeys(config.Directories) {
				fmt.Println(key)
				if alias := config.Directories[key].Alias; alias != "" {
					fmt.Println("-" + alias)
				}
			}
		}
	}
}
