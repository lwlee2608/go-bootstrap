package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGeneratedProjectBuilds(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		AppName:    "myapp",
		ModuleName: "github.com/user/myapp",
		OutputDir:  tmpDir,
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Change to generated project and run make build
	cmd := exec.Command("make", "build")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("make build failed: %v", err)
	}

	// Verify binary exists
	binaryPath := filepath.Join(tmpDir, "bin", "myapp")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Errorf("binary not created: %s", binaryPath)
	}

	// Run the generated binary
	runCmd := exec.Command(binaryPath)
	output, err := runCmd.Output()
	if err != nil {
		t.Fatalf("running binary failed: %v", err)
	}

	expected := "myapp v0.1.0\n"
	if string(output) != expected {
		t.Errorf("unexpected output: got %q, want %q", string(output), expected)
	}
}

func TestGeneratedProjectWithHttpBuilds(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		AppName:    "myapp",
		ModuleName: "github.com/user/myapp",
		AddHTTP:    true,
		OutputDir:  tmpDir,
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Change to generated project and run make build
	cmd := exec.Command("make", "build")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("make build failed: %v", err)
	}

	// Verify binary exists
	binaryPath := filepath.Join(tmpDir, "bin", "myapp")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Errorf("binary not created: %s", binaryPath)
	}
}
