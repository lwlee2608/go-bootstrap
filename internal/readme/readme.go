package readme

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lwlee2608/aiwire"
	"github.com/openai/openai-go/v3"
)

const model = "openai/gpt-5.6-luna"

var baseURL = "https://openrouter.ai/api/v1"

//go:embed system_prompt.md
var systemPrompt string

type Config struct {
	AppName     string
	ModuleName  string
	Description string
	FullStack   bool
	OutputDir   string
	APIKey      string
}

func EnvKey() string {
	return os.Getenv("OPENROUTER_API_KEY")
}

func ValidateKey(ctx context.Context, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("no API key")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/key", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach openrouter: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("key rejected by openrouter")
	default:
		return fmt.Errorf("openrouter returned %s", resp.Status)
	}

	var body struct {
		Data struct {
			Limit          *float64 `json:"limit"`
			LimitRemaining *float64 `json:"limit_remaining"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}
	if body.Data.Limit != nil && body.Data.LimitRemaining != nil && *body.Data.LimitRemaining <= 0 {
		return fmt.Errorf("key has no remaining credit")
	}
	return nil
}

func Generate(ctx context.Context, cfg Config) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("no OpenRouter API key")
	}

	service := aiwire.NewOpenAIService(cfg.APIKey, baseURL)
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
