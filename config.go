package main

type Config struct {
	Scripts map[string][]Task `json:"scripts"`
}

type Task struct {
	Name    string `json:"name"`
	Alias   string `json:"alias,omitempty"`
	Dir     string `json:"dir"`
	Command string `json:"command"`
}