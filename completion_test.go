package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestCompletionCandidates(t *testing.T) {
	config := &Config{
		Directories: map[string]Directory{
			"FRONTEND": {Alias: "F", Path: "./frontend"},
			"BACKEND":  {Alias: "B", Path: "./backend"},
		},
		Scripts: map[string]ScriptGroup{"dev": {}, "build": {}},
	}

	if got, want := completionCandidates("scripts", config, nil), []string{"build", "dev"}; !reflect.DeepEqual(got, want) {
		t.Errorf("scripts: got %v, want %v", got, want)
	}
	if got, want := completionCandidates("dirs", config, nil), []string{".", "BACKEND", "-B", "FRONTEND", "-F"}; !reflect.DeepEqual(got, want) {
		t.Errorf("dirs: got %v, want %v", got, want)
	}
	commands := completionCandidates("commands", config, nil)
	if len(commands) != len(builtinCommands)+2 || commands[len(commands)-2] != "build" || commands[len(commands)-1] != "dev" {
		t.Errorf("commands: got %v", commands)
	}
}

func TestCompletionCandidatesIgnoresBrokenConfig(t *testing.T) {
	configErr := errors.New("broken config")
	if got := completionCandidates("dirs", nil, configErr); got != nil {
		t.Errorf("dirs: got %v, want no candidates", got)
	}
	if got := completionCandidates("commands", nil, configErr); !reflect.DeepEqual(got, builtinCommands) {
		t.Errorf("commands: got %v, want builtins", got)
	}
}
