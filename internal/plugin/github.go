package plugin

import (
	"context"
	"fmt"
	"strings"
	"text/template"
)

type GitHubActionsPlugin struct{}

func (p *GitHubActionsPlugin) Name() string        { return "github-actions" }
func (p *GitHubActionsPlugin) Description() string { return "GitHub Actions CI/CD workflows" }
func (p *GitHubActionsPlugin) SupportedStacks() []string {
	return []string{"go", "python", "node", "rust"}
}
func (p *GitHubActionsPlugin) Files() []string { return []string{".github/workflows/ci.yml"} }

func (p *GitHubActionsPlugin) Questions() []*Question {
	return []*Question{
		{
			Name:        "stack",
			Description: "What is your project stack?",
			Type:        "select",
			Options:     []string{"go", "python", "node", "rust"},
			Default:     "go",
			Required:    true,
		},
		{
			Name:        "goVersion",
			Description: "Go version (if using Go)",
			Type:        "text",
			Default:     "1.21",
			Required:    false,
		},
		{
			Name:        "nodeVersion",
			Description: "Node version (if using Node)",
			Type:        "text",
			Default:     "20",
			Required:    false,
		},
		{
			Name:        "pythonVersion",
			Description: "Python version (if using Python)",
			Type:        "text",
			Default:     "3.11",
			Required:    false,
		},
		{
			Name:        "runTests",
			Description: "Run tests?",
			Type:        "confirm",
			Default:     "true",
			Required:    false,
		},
	}
}

func (p *GitHubActionsPlugin) Generate(ctx context.Context, answers map[string]interface{}) (string, error) {
	stack, _ := answers["stack"].(string)
	if stack == "" {
		stack = "go"
	}

	goVersion, _ := answers["goVersion"].(string)
	if goVersion == "" {
		goVersion = "1.21"
	}

	nodeVersion, _ := answers["nodeVersion"].(string)
	if nodeVersion == "" {
		nodeVersion = "20"
	}

	pythonVersion, _ := answers["pythonVersion"].(string)
	if pythonVersion == "" {
		pythonVersion = "3.11"
	}

	runTests, _ := answers["runTests"].(string)
	if runTests == "" {
		runTests = "true"
	}

	tmpl := `name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up {{.SetupName}}
        uses: actions/setup-{{.SetupAction}}@v4
        with:
          {{.VersionKey}}: {{.Version}}

      - name: Get {{.CacheName}} dependencies
        run: {{.InstallCmd}}

      - name: Run tests
        if: {{.RunTests}}
        run: {{.TestCmd}}
`

	t, err := template.New("github-actions").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var setupName, setupAction, versionKey, version, installCmd, testCmd string

	switch stack {
	case "go":
		setupName = "Go"
		setupAction = "go"
		versionKey = "go-version"
		version = goVersion
		installCmd = "go mod download"
		testCmd = "go test -v ./..."
	case "python":
		setupName = "Python"
		setupAction = "python"
		versionKey = "python-version"
		version = pythonVersion
		installCmd = "pip install -r requirements.txt"
		testCmd = "pytest"
	case "node":
		setupName = "Node"
		setupAction = "node"
		versionKey = "node-version"
		version = nodeVersion
		installCmd = "npm ci"
		testCmd = "npm test"
	case "rust":
		setupName = "Rust"
		setupAction = "rust"
		versionKey = "rust-version"
		version = "stable"
		installCmd = "cargo fetch"
		testCmd = "cargo test"
	default:
		setupName = "Go"
		setupAction = "go"
		versionKey = "go-version"
		version = goVersion
		installCmd = "go mod download"
		testCmd = "go test ./..."
	}

	var buf strings.Builder
	err = t.Execute(&buf, map[string]string{
		"SetupName":   setupName,
		"SetupAction": setupAction,
		"VersionKey":  versionKey,
		"Version":     version,
		"InstallCmd":  installCmd,
		"TestCmd":     testCmd,
		"RunTests":    runTests,
		"CacheName":   setupName,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (p *GitHubActionsPlugin) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("empty workflow content")
	}
	return nil
}
