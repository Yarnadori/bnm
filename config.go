package main

type Config struct {
	Directories map[string]Directory   `json:"directories,omitempty"`
	Scripts     map[string]ScriptGroup `json:"scripts"`
}

type ScriptGroup struct {
	Mode  string `json:"mode,omitempty"`
	Tasks []Task `json:"tasks"`
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