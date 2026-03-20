package plugin

import (
	"context"
	"fmt"
	"strings"
	"text/template"
)

type DockerPlugin struct{}

func (p *DockerPlugin) Name() string              { return "docker" }
func (p *DockerPlugin) Description() string       { return "Dockerfile and docker-compose.yml" }
func (p *DockerPlugin) SupportedStacks() []string { return []string{"go", "python", "node", "rust"} }
func (p *DockerPlugin) Files() []string           { return []string{"Dockerfile", "docker-compose.yml"} }

func (p *DockerPlugin) Questions() []*Question {
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
			Name:        "image",
			Description: "Base Docker image",
			Type:        "text",
			Default:     "alpine",
			Required:    false,
		},
		{
			Name:        "port",
			Description: "Exposed port",
			Type:        "text",
			Default:     "8080",
			Required:    false,
		},
		{
			Name:        "serviceName",
			Description: "Service name for docker-compose",
			Type:        "text",
			Default:     "app",
			Required:    false,
		},
	}
}

func (p *DockerPlugin) Generate(ctx context.Context, answers map[string]interface{}) (string, error) {
	stack, _ := answers["stack"].(string)
	if stack == "" {
		stack = "go"
	}

	image, _ := answers["image"].(string)
	if image == "" {
		image = "alpine"
	}

	port, _ := answers["port"].(string)
	if port == "" {
		port = "8080"
	}

	serviceName, _ := answers["serviceName"].(string)
	if serviceName == "" {
		serviceName = "app"
	}

	dockerfile, err := p.generateDockerfile(stack, image, port)
	if err != nil {
		return "", err
	}

	compose, err := p.generateCompose(stack, serviceName, port)
	if err != nil {
		return "", err
	}

	return dockerfile + "\n---\n" + compose, nil
}

func (p *DockerPlugin) generateDockerfile(stack, image, port string) (string, error) {
	var baseImage string
	var runCmds string
	var copyCmds string
	var cmd string

	switch stack {
	case "go":
		baseImage = "golang:" + image
		runCmds = "RUN go mod download"
		copyCmds = "COPY . .\nRUN CGO_ENABLED=0 GOOS=linux go build -o /app main.go"
		cmd = "CMD [\"/app\"]"
	case "python":
		baseImage = "python:3.11-slim"
		runCmds = "RUN pip install --no-cache-dir -r requirements.txt"
		copyCmds = "COPY . ."
		cmd = "CMD [\"python\", \"main.py\"]"
	case "node":
		baseImage = "node:20-alpine"
		runCmds = "RUN npm ci --only=production"
		copyCmds = "COPY . ."
		cmd = "CMD [\"node\", \"index.js\"]"
	case "rust":
		baseImage = "rust:alpine"
		runCmds = "RUN cargo build --release"
		copyCmds = "COPY . ."
		cmd = "CMD [\"/app/target/release/app\"]"
	default:
		baseImage = "alpine:latest"
		copyCmds = "COPY . ."
		cmd = "CMD [\"sh\"]"
	}

	tmpl := `FROM {{.BaseImage}}

WORKDIR /app

{{.RunCmds}}

{{.CopyCmds}}

EXPOSE {{.Port}}

{{.Cmd}}
`

	t, err := template.New("dockerfile").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = t.Execute(&buf, map[string]string{
		"BaseImage": baseImage,
		"RunCmds":   runCmds,
		"CopyCmds":  copyCmds,
		"Port":      port,
		"Cmd":       cmd,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (p *DockerPlugin) generateCompose(stack, serviceName, port string) (string, error) {
	tmpl := `version: '3.8'

services:
  {{.ServiceName}}:
    build: .
    ports:
      - "{{.Port}}:{{.Port}}"
    environment:
      - NODE_ENV=production
    restart: unless-stopped
`

	t, err := template.New("compose").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = t.Execute(&buf, map[string]string{
		"ServiceName": serviceName,
		"Port":        port,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (p *DockerPlugin) Validate(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("empty dockerfile content")
	}
	return nil
}
