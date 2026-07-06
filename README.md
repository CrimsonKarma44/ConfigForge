# ConfigForge

An agentic CLI tool for generating configuration files for different project types. Features a plugin system for extensibility and optional LLM-powered enhancement.

## Motivation

ConfigForge was created to empower developers who actually build—developers who understand their craft, not those who are entirely dependent on AI. The goal is simple: **eliminate boilerplate configuration overhead** so you can spend less time setting up and more time building.

When you work across multiple languages and frameworks, you shouldn't have to mentally context-switch between different project structures, config formats, and best practices every single time. ConfigForge dispatches that cognitive load and lets you move from "new project" to "actually coding" in seconds. Whether you're spinning up a Go microservice, a Python CLI, or a Node.js package, the same familiar tool guides you through setup while respecting language-specific conventions.

This is for developers who want to focus on what matters: the logic, the architecture, the solving of real problems—not remembering whether it's `pyproject.toml` or `setup.py`.

## Features

- **Interactive CLI** - Prompts for configuration options
- **Plugin System** - Extensible architecture for adding new config generators
- **Hybrid Mode** - Optional LLM enhancement for smarter configs
- **Multi-stack Support** - Go, Python, Node.js, Rust, and more

## Quick Start

### Installation

```bash
go install github.com/deus/configforge@latest
```

Or build from source:

```bash
git clone https://github.com/deus/configforge.git
cd configforge
go build -o configforge ./cmd/main.go
```

### First Steps

```bash
# Interactive mode - guided setup
configforge

# List available plugins
configforge --list

# Generate a specific config
configforge docker
configforge python
configforge go
```

## Usage

### List Available Plugins

```bash
configforge --list
```

### Filter by Stack

```bash
configforge --list --stack go
```

### Generate Configs

```bash
# Interactive mode
configforge

# Generate specific config
configforge docker
configforge python
configforge go
configforge node
configforge github-actions
```

### Output Directory

```bash
configforge --output ./configs docker
```

### LLM Enhancement (Hybrid Mode)

```bash
# Requires OPENAI_API_KEY
export OPENAI_API_KEY=your_key
configforge --llm python
```

## Built-in Plugins

| Plugin | Description | Files |
|--------|-------------|-------|
| `docker` | Dockerfile & docker-compose | Dockerfile, docker-compose.yml |
| `github-actions` | CI/CD workflows | .github/workflows/ci.yml |
| `python` | Python project config | pyproject.toml |
| `go` | Go module config | go.mod |
| `node` | Node.js package.json | package.json |

## Contributing

### Plugin Development

#### Plugin Interface

```go
type ConfigPlugin interface {
    Name() string
    Description() string
    SupportedStacks() []string
    Questions() []*Question
    Generate(ctx context.Context, answers map[string]interface{}) (string, error)
    Validate(content string) error
    Files() []string
}
```

#### Creating a Plugin

1. Create a new file in `internal/plugin/`:

```go
package plugin

type MyPlugin struct{}

func (p *MyPlugin) Name() string          { return "myplugin" }
func (p *MyPlugin) Description() string  { return "My custom config" }
func (p *MyPlugin) SupportedStacks() []string { return []string{"go"} }
func (p *MyPlugin) Files() []string        { return []string{"myconfig.yaml"} }

func (p *MyPlugin) Questions() []*Question {
    return []*Question{
        {
            Name:        "name",
            Description: "Config name",
            Type:        "text",
            Required:    true,
        },
    }
}

func (p *MyPlugin) Generate(ctx context.Context, answers map[string]interface{}) (string, error) {
    // Generate config based on answers
    return "config content", nil
}

func (p *MyPlugin) Validate(content string) error {
    return nil
}
```

2. Register in loader.go

### Configuration

Plugin discovery paths:
- `./plugins` (project local)
- `~/.config/configforge/plugins` (user directory)

## License

MIT
