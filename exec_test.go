package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestResolveExecTargetNamePriority(t *testing.T) {
	// "z" is both a formal name and another directory's alias; the formal
	// name must win regardless of sort order
	config := &Config{Directories: map[string]Directory{
		"other": {Alias: "z", Path: "./other"},
		"z":     {Path: "./real-z"},
	}}

	name, dir, found := resolveExecTarget(config, "z")
	if !found || name != "z" || dir != "./real-z" {
		t.Errorf("got %q / %q / %v, want the formal name to win", name, dir, found)
	}

	// The alias still resolves when nothing else matches
	delete(config.Directories, "z")
	name, dir, found = resolveExecTarget(config, "z")
	if !found || name != "other" || dir != "./other" {
		t.Errorf("alias fallback: got %q / %q / %v", name, dir, found)
	}

	if _, _, found := resolveExecTarget(config, "missing"); found {
		t.Error("expected no match for unknown name")
	}
}

func TestResolveExecTargetPathBeatsAlias(t *testing.T) {
	// "api" is one directory's alias and another's path; paths win, matching
	// the priority of script filters
	config := &Config{Directories: map[string]Directory{
		"X": {Alias: "api", Path: "./x"},
		"Y": {Path: "./api"},
	}}

	name, dir, found := resolveExecTarget(config, "api")
	if !found || name != "Y" || dir != "./api" {
		t.Errorf("got %q / %q / %v, want the path match to win", name, dir, found)
	}
}

func TestRunExecAllProcessesContinuesAfterFailureInSortedOrder(t *testing.T) {
	config := &Config{Directories: map[string]Directory{
		"ZETA":  {Path: "./zeta"},
		"ALPHA": {Path: "./alpha"},
		"MID":   {Path: "./mid"},
	}}

	var called []string
	failed := runExecAllProcesses(context.Background(), config, []string{"BASE=1"}, func(_ context.Context, task Task, env []string) error {
		called = append(called, task.Name)
		if task.Name == "ALPHA" {
			return errors.New("failed")
		}
		return nil
	}, "echo test")

	if want := []string{"ALPHA", "MID", "ZETA"}; !reflect.DeepEqual(called, want) {
		t.Errorf("called in order %v, want %v", called, want)
	}
	if want := []string{"ALPHA"}; !reflect.DeepEqual(failed, want) {
		t.Errorf("failed directories %v, want %v", failed, want)
	}
}

func TestRunExecAllProcessesStopsWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := &Config{Directories: map[string]Directory{
		"A": {Path: "./a"},
		"B": {Path: "./b"},
	}}

	var called []string
	failed := runExecAllProcesses(ctx, config, nil, func(_ context.Context, task Task, _ []string) error {
		called = append(called, task.Name)
		cancel()
		return context.Canceled
	}, "echo test")

	if want := []string{"A"}; !reflect.DeepEqual(called, want) {
		t.Errorf("called %v, want %v", called, want)
	}
	if len(failed) != 0 {
		t.Errorf("cancellation reported failures: %v", failed)
	}
}
