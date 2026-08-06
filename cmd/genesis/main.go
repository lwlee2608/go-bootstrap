package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lwlee2608/go-bootstrap/internal/git"
	"github.com/lwlee2608/go-bootstrap/internal/readme"
	"github.com/lwlee2608/go-bootstrap/internal/scaffold"
	"github.com/lwlee2608/go-bootstrap/internal/tui"
)

var AppVersion = "dev"

func main() {
	version := flag.Bool("version", false, "print version")
	flag.Parse()

	if *version {
		fmt.Printf("genesis %s\n", AppVersion)
		os.Exit(0)
	}

	outputDir := "."

	suggestedApp := git.DetectAppName()
	suggestedModule := git.DetectModuleName()
	model := tui.New(tui.Options{
		SuggestedAppName:    suggestedApp,
		SuggestedModuleName: suggestedModule,
	})
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result, err := finalModel.(tui.Model).Result()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := scaffold.Config{
		AppName:    result.AppName,
		ModuleName: result.ModuleName,
		AddHTTP:    result.AddHTTP,
		FullStack:  result.FullStack,
		OutputDir:  outputDir,
	}

	if err := scaffold.Generate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating project: %v\n", err)
		os.Exit(1)
	}

	if result.GenReadme {
		fmt.Println("\nGenerating README.md with AI...")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := readme.Generate(ctx, readme.Config{
			AppName:     result.AppName,
			ModuleName:  result.ModuleName,
			Description: result.Description,
			FullStack:   result.FullStack,
			OutputDir:   outputDir,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping AI README: %v\n", err)
		}
	}

	fmt.Println("\nProject created!")
	fmt.Println("\nNext steps:")
	if result.FullStack {
		fmt.Printf("  cd services/%s-server && make build\n", result.AppName)
		fmt.Printf("  cd services/%s-web && pnpm install\n", result.AppName)
	} else {
		fmt.Println("  make build")
		fmt.Printf("  ./bin/%s\n", result.AppName)
	}
}
