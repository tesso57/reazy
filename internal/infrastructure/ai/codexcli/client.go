// Package codexcli provides a Codex CLI based AI client.
package codexcli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultCommand = "codex"
	defaultSandbox = "read-only"
	defaultTimeout = 30 * time.Second
)

// Config controls Codex CLI subprocess invocation.
type Config struct {
	Command          string
	Model            string
	WebSearch        string
	ReasoningEffort  string
	ReasoningSummary string
	Verbosity        string
	Sandbox          string
	Timeout          time.Duration
}

// Runner executes Codex and returns stdout/stderr text.
type Runner func(ctx context.Context, command string, args []string, stdin string) (string, string, error)

// Client implements ai.Client by invoking Codex CLI as a subprocess.
type Client struct {
	config Config
	run    Runner
}

// NewClient creates a Codex CLI client.
func NewClient(cfg Config) Client {
	return Client{
		config: normalizeConfig(cfg),
		run:    defaultRunner,
	}
}

// NewClientWithRunner creates a client with a custom runner for tests.
func NewClientWithRunner(cfg Config, runner Runner) Client {
	normalized := normalizeConfig(cfg)
	if runner == nil {
		runner = defaultRunner
	}
	return Client{
		config: normalized,
		run:    runner,
	}
}

// Generate executes Codex and returns raw text output.
func (c Client) Generate(ctx context.Context, prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("prompt is empty")
	}

	runCtx := ctx
	cancel := func() {}
	if c.config.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, c.config.Timeout)
	}
	defer cancel()

	stdout, stderr, err := c.run(runCtx, c.config.Command, c.args(), prompt)
	if err != nil {
		reason := strings.TrimSpace(stderr)
		if reason == "" {
			reason = strings.TrimSpace(stdout)
		}
		if reason == "" {
			return "", fmt.Errorf("codex exec failed: %w", err)
		}
		return "", fmt.Errorf("codex exec failed: %w: %s", err, reason)
	}
	return stdout, nil
}

func normalizeConfig(cfg Config) Config {
	normalized := cfg
	if strings.TrimSpace(normalized.Command) == "" {
		normalized.Command = defaultCommand
	}
	if strings.TrimSpace(normalized.Sandbox) == "" {
		normalized.Sandbox = defaultSandbox
	}
	if normalized.Timeout <= 0 {
		normalized.Timeout = defaultTimeout
	}
	return normalized
}

func (c Client) args() []string {
	args := []string{
		"exec",
		"--skip-git-repo-check",
		"--sandbox", c.config.Sandbox,
		"--color", "never",
	}
	if strings.TrimSpace(c.config.Model) != "" {
		args = append(args, "-m", c.config.Model)
	}
	if strings.TrimSpace(c.config.WebSearch) != "" {
		args = append(args, "-c", fmt.Sprintf("web_search=%q", c.config.WebSearch))
	}
	if strings.TrimSpace(c.config.ReasoningEffort) != "" {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", c.config.ReasoningEffort))
	}
	if strings.TrimSpace(c.config.ReasoningSummary) != "" {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_summary=%q", c.config.ReasoningSummary))
	}
	if strings.TrimSpace(c.config.Verbosity) != "" {
		args = append(args, "-c", fmt.Sprintf("model_verbosity=%q", c.config.Verbosity))
	}
	args = append(args, "-")
	return args
}

func defaultRunner(ctx context.Context, command string, args []string, stdin string) (string, string, error) {
	cmd := exec.CommandContext(ctx, command, args...) //nolint:gosec
	cmd.Stdin = strings.NewReader(stdin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
