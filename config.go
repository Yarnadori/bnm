package main

type Config struct {
	Directories map[string]Directory `json:"directories,omitempty"`
	Scripts     map[string][]Task    `json:"scripts"`
}

type Directory struct {
	Alias string `json:"alias"`
	Path  string `json:"path"`
}

type Task struct {
	Name    string `json:"-"`
	Dir     string `json:"dir"`
	Command string `json:"command"`
}