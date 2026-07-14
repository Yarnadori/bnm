package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncPreservesScriptsVerbatim(t *testing.T) {
	root := setupProjectDirs(t, "frontend", "backend")
	// OS-specific commands and key order must survive a sync rewrite, even
	// though loading collapses the command to the current OS
	content := `{
  "name": "my-app",
  "directories": {
    "frontend": "./frontend"
  },
  "scripts": {
    "build": {
      "zeta-first": {
        "windows": "build.cmd",
        "linux": "make",
        "default": "make"
      },
      "alpha-second": "echo hi"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(root, configFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	syncProject()

	data, err := os.ReadFile(configFileName)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{`"windows": "build.cmd"`, `"linux": "make"`, `"backend"`} {
		if !strings.Contains(got, want) {
			t.Errorf("synced config lost %s:\n%s", want, got)
		}
	}
	if strings.Index(got, "zeta-first") > strings.Index(got, "alpha-second") {
		t.Errorf("script key order was not preserved:\n%s", got)
	}
}
