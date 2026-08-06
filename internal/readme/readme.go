package readme

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lwlee2608/aiwire"
	"github.com/openai/openai-go/v3"
)

const model = "z-ai/glm-4.7"

//go:embed system_prompt.md
var systemPrompt string

type Config struct {
	AppName     string
	ModuleName  string
	Description string
	FullStack   bool
	OutputDir   string
}

func Generate(ctx context.Context, cfg Config) error {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENROUTER_API_KEY not set")
	}

	service := aiwire.NewOpenAIService(apiKey, "https://openrouter.ai/api/v1")
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(strings.TrimSpace(systemPrompt)),
		openai.UserMessage(prompt(cfg)),
	}

	resp, err := service.Completions(ctx, messages, nil, aiwire.CompletionOption{
		Model:       model,
		Temperature: 0.3,
		Provider:    &aiwire.ProviderOption{AllowFallbacks: true},
	})
	if err != nil {
		return err
	}

	content := strings.TrimSpace(resp.Message.Content)
	if content == "" {
		return fmt.Errorf("model returned an empty README")
	}

	return os.WriteFile(filepath.Join(cfg.OutputDir, "README.md"), []byte(content+"\n"), 0o644)
}

func prompt(cfg Config) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Project: %s\nGo module: %s\n", cfg.AppName, cfg.ModuleName)
	if cfg.Description != "" {
		fmt.Fprintf(&b, "Author's description: %s\n", cfg.Description)
	}
	if cfg.FullStack {
		fmt.Fprintf(&b, "Top-level structure: services/%s-server (Go backend and API), services/%s-web (React + TypeScript web UI)\n",
			cfg.AppName, cfg.AppName)
	} else {
		b.WriteString("Top-level structure: cmd/ (entrypoint), internal/ (application code)\n")
	}
	b.WriteString("\nWrite the README.md for this project.")
	return b.String()
}
