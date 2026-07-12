package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

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
