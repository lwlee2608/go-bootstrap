package scaffold

import (
	"embed"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*
var templates embed.FS

type Config struct {
	AppName    string
	ModuleName string
	OutputDir  string
}

func Generate(cfg Config) error {
	// Create directories
	dirs := []string{
		cfg.OutputDir,
		filepath.Join(cfg.OutputDir, "cmd", cfg.AppName),
		filepath.Join(cfg.OutputDir, "pkg"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Generate files from templates
	files := []struct {
		tmpl string
		dest string
	}{
		{"templates/go.mod.tmpl", "go.mod"},
		{"templates/Makefile.tmpl", "Makefile"},
		{"templates/gitignore.tmpl", ".gitignore"},
		{"templates/main.go.tmpl", filepath.Join("cmd", cfg.AppName, "main.go")},
	}

	for _, f := range files {
		if err := generateFile(f.tmpl, filepath.Join(cfg.OutputDir, f.dest), cfg); err != nil {
			return err
		}
	}

	return nil
}

func generateFile(tmplPath, destPath string, data Config) error {
	content, err := templates.ReadFile(tmplPath)
	if err != nil {
		return err
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(content))
	if err != nil {
		return err
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}
