package scaffold

import (
	"embed"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates
var templates embed.FS

type Config struct {
	AppName    string
	ModuleName string
	AddHTTP    bool
	OutputDir  string
}

func Generate(cfg Config) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return err
	}

	// Create pkg directory explicitly
	if err := os.MkdirAll(filepath.Join(cfg.OutputDir, "pkg"), 0755); err != nil {
		return err
	}

	err := fs.WalkDir(templates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Skip files not ending in .tmpl (if any)
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Calculate relative path from "templates" directory
		relPath, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}

		// Special handling for HTTP files if AddHTTP is false
		if !cfg.AddHTTP {
			if strings.HasPrefix(relPath, "internal/api/http") ||
				relPath == "cmd/config.go.tmpl" ||
				relPath == "cmd/logger.go.tmpl" ||
				relPath == "application.yml.tmpl" {
				return nil
			}
		}

		// Determine destination path
		destRelPath := strings.TrimSuffix(relPath, ".tmpl")
		if destRelPath == "gitignore" {
			destRelPath = ".gitignore"
		}
		if strings.HasPrefix(destRelPath, "cmd/") {
			// Map cmd/main.go -> cmd/{appName}/main.go
			destRelPath = filepath.Join("cmd", cfg.AppName, strings.TrimPrefix(destRelPath, "cmd/"))
		}

		destPath := filepath.Join(cfg.OutputDir, destRelPath)

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return generateFile(path, destPath, cfg)
	})

	if err != nil {
		return err
	}

	// Run go mod tidy at the destination
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = cfg.OutputDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return err
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
