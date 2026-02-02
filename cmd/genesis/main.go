package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lwlee2608/go-bootstrap/internal/git"
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
		OutputDir:  outputDir,
	}

	if err := scaffold.Generate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating project: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nProject created!")
	fmt.Println("\nNext steps:")
	fmt.Println("  make build")
	fmt.Printf("  ./bin/%s\n", result.AppName)
}
