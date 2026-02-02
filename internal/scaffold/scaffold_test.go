package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		AppName:    "testapp",
		ModuleName: "github.com/test/testapp",
		OutputDir:  tmpDir,
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check directories exist
	dirs := []string{
		filepath.Join(tmpDir, "cmd", "testapp"),
		filepath.Join(tmpDir, "pkg"),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("directory not created: %s", dir)
		}
	}

	// Check files exist and have correct content
	gomod, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	if !strings.Contains(string(gomod), "github.com/test/testapp") {
		t.Error("go.mod does not contain module name")
	}

	makefile, err := os.ReadFile(filepath.Join(tmpDir, "Makefile"))
	if err != nil {
		t.Fatalf("failed to read Makefile: %v", err)
	}
	if !strings.Contains(string(makefile), "APP             := testapp") {
		t.Error("Makefile does not contain app name")
	}

	mainGo, err := os.ReadFile(filepath.Join(tmpDir, "cmd", "testapp", "main.go"))
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}
	if !strings.Contains(string(mainGo), "testapp") {
		t.Error("main.go does not contain app name")
	}

	gitignore, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignore), "bin/") {
		t.Error(".gitignore does not contain bin/")
	}
}
