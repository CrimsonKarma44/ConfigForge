package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Provider interface {
	Name() string
	Enhance(ctx context.Context, pluginName, content string) (string, error)
}

type LLM struct {
	provider Provider
}

func New(model string) (*LLM, error) {
	provider, err := detectProvider(model)
	if err != nil {
		return nil, err
	}
	return &LLM{provider: provider}, nil
}

func detectProvider(model string) (Provider, error) {
	lowerModel := strings.ToLower(model)

	if strings.HasPrefix(lowerModel, "anthropic/") || strings.HasPrefix(lowerModel, "claude") {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set for Claude model")
		}
		actualModel := strings.TrimPrefix(model, "anthropic/")
		return &AnthropicProvider{model: actualModel}, nil
	}

	if strings.HasPrefix(lowerModel, "google/") || strings.HasPrefix(lowerModel, "gemini") || strings.HasPrefix(lowerModel, "gemma") {
		apiKey := os.Getenv("GOOGLE_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GOOGLE_API_KEY not set for Gemini model")
		}
		actualModel := strings.TrimPrefix(model, "google/")
		return &GoogleProvider{model: actualModel}, nil
	}

	if strings.HasPrefix(lowerModel, "ollama/") || strings.HasPrefix(lowerModel, "local/") || model == "ollama" {
		actualModel := strings.TrimPrefix(strings.TrimPrefix(model, "ollama/"), "local/")
		if actualModel == "" {
			actualModel = "llama2"
		}
		return &OllamaProvider{model: actualModel}, nil
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}
	return &OpenAIProvider{model: model}, nil
}

func (l *LLM) Enhance(ctx context.Context, pluginName, content string) (string, error) {
	return l.provider.Enhance(ctx, pluginName, content)
}

func (l *LLM) Name() string {
	return l.provider.Name()
}

type OpenAIProvider struct {
	model string
}

func (p *OpenAIProvider) Name() string { return "OpenAI" }

func (p *OpenAIProvider) Enhance(ctx context.Context, pluginName, content string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	prompt := buildPrompt(pluginName, content)

	payload := map[string]interface{}{
		"model":       p.model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  2000,
		"temperature": 0.7,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return "", fmt.Errorf("rate limited (429) - wait a moment or check your OpenAI plan limits")
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return parseOpenAIResponse(resp.Body)
}

type AnthropicProvider struct {
	model string
}

func (p *AnthropicProvider) Name() string { return "Anthropic Claude" }

func (p *AnthropicProvider) Enhance(ctx context.Context, pluginName, content string) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	prompt := buildPrompt(pluginName, content)

	payload := map[string]interface{}{
		"model":      p.model,
		"max_tokens": 2000,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return "", fmt.Errorf("rate limited (429) - wait a moment or check your Anthropic plan limits")
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	contentArr, ok := result["content"].([]interface{})
	if !ok || len(contentArr) == 0 {
		return "", fmt.Errorf("no content returned")
	}

	text, ok := contentArr[0].(map[string]interface{})["text"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	return strings.TrimSpace(text), nil
}

type GoogleProvider struct {
	model string
}

func (p *GoogleProvider) Name() string { return "Google Gemini" }

func (p *GoogleProvider) Enhance(ctx context.Context, pluginName, content string) (string, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY not set")
	}

	prompt := buildPrompt(pluginName, content)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 2000,
		},
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.model, apiKey)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return "", fmt.Errorf("rate limited (429) - wait a moment or check your Google Cloud quota")
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("no candidates returned")
	}

	candidate := candidates[0].(map[string]interface{})
	contentMap := candidate["content"].(map[string]interface{})
	parts := contentMap["parts"].([]interface{})
	if len(parts) == 0 {
		return "", fmt.Errorf("no parts returned")
	}

	text := parts[0].(map[string]interface{})["text"].(string)

	return strings.TrimSpace(text), nil
}

type OllamaProvider struct {
	model string
}

func (p *OllamaProvider) Name() string { return "Ollama (Local)" }

func (p *OllamaProvider) Enhance(ctx context.Context, pluginName, content string) (string, error) {
	model := p.model
	if model == "ollama" || model == "" {
		model = "llama2"
	}

	prompt := buildPrompt(pluginName, content)

	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/generate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("connect to Ollama: %w (make sure Ollama is running: ollama serve)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Ollama error: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	response, ok := result["response"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	return strings.TrimSpace(response), nil
}

func buildPrompt(pluginName, content string) string {
	return fmt.Sprintf(`You are a configuration expert. Review and improve this generated config for a %s plugin.

Current config:
%s

Improve it by:
1. Adding any missing best practices
2. Fixing any potential issues
3. Adding helpful comments where needed

Return ONLY the improved config, no explanations.`, pluginName, content)
}

func parseOpenAIResponse(body io.Reader) (string, error) {
	var result map[string]interface{}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return "", err
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid choice")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("no message")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("no content")
	}

	return strings.TrimSpace(content), nil
}
