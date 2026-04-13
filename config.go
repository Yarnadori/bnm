package main

import (
	"encoding/json"
	"runtime"
)

type Config struct {
	Name        string                 `json:"name,omitempty"`
	Version     string                 `json:"version,omitempty"`
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
	Name    string  `json:"-"`
	Dir     string  `json:"dir"`
	Command Command `json:"command"`
}

type Command string

func (c *Command) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*c = Command(s)
		return nil
	}

	var m map[string]string
	if err := json.Unmarshal(data, &m); err == nil {
		osName := runtime.GOOS
		if cmd, ok := m[osName]; ok {
			*c = Command(cmd)
		} else if osName == "darwin" && m["mac"] != "" {
			*c = Command(m["mac"])
		} else if m["default"] != "" {
			*c = Command(m["default"])
		} else {
			*c = ""
		}
		return nil
	}

	return nil
}

func (c Command) String() string {
	return string(c)
}
