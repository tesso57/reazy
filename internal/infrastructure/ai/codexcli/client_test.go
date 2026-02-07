package codexcli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestClient_Generate(t *testing.T) {
	var gotCommand string
	var gotArgs []string
	var gotInput string

	client := NewClientWithRunner(Config{
		Command:          "codex-bin",
		Model:            "gpt-5",
		WebSearch:        "live",
		ReasoningEffort:  "medium",
		ReasoningSummary: "concise",
		Verbosity:        "low",
		Sandbox:          "read-only",
		Timeout:          5 * time.Second,
	}, func(_ context.Context, command string, args []string, stdin string) (string, string, error) {
		gotCommand = command
		gotArgs = append([]string(nil), args...)
		gotInput = stdin
		return "assistant output", "", nil
	})

	got, err := client.Generate(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got != "assistant output" {
		t.Fatalf("output = %q, want %q", got, "assistant output")
	}

	if gotCommand != "codex-bin" {
		t.Fatalf("command = %q, want %q", gotCommand, "codex-bin")
	}
	if gotInput != "test prompt" {
		t.Fatalf("stdin = %q, want %q", gotInput, "test prompt")
	}

	if !containsArgPair(gotArgs, "-m", "gpt-5") {
		t.Fatalf("args missing model: %#v", gotArgs)
	}
	if !containsArgPair(gotArgs, "--sandbox", "read-only") {
		t.Fatalf("args missing sandbox: %#v", gotArgs)
	}
	if !containsArgPair(gotArgs, "-c", `web_search="live"`) {
		t.Fatalf("args missing web_search: %#v", gotArgs)
	}
	if !containsArgPair(gotArgs, "-c", `model_reasoning_effort="medium"`) {
		t.Fatalf("args missing reasoning effort: %#v", gotArgs)
	}
	if !containsArgPair(gotArgs, "-c", `model_reasoning_summary="concise"`) {
		t.Fatalf("args missing reasoning summary: %#v", gotArgs)
	}
	if !containsArgPair(gotArgs, "-c", `model_verbosity="low"`) {
		t.Fatalf("args missing verbosity: %#v", gotArgs)
	}
	if len(gotArgs) == 0 || gotArgs[len(gotArgs)-1] != "-" {
		t.Fatalf("last arg should be '-', got %#v", gotArgs)
	}
}

func TestClient_GenerateErrors(t *testing.T) {
	client := NewClientWithRunner(Config{}, func(_ context.Context, _ string, _ []string, _ string) (string, string, error) {
		return "", "auth required", errors.New("exit status 1")
	})

	_, err := client.Generate(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth required") {
		t.Fatalf("error = %v, expected stderr in message", err)
	}
}

func TestClient_EmptyPrompt(t *testing.T) {
	client := NewClientWithRunner(Config{}, nil)
	_, err := client.Generate(context.Background(), "")
	if err == nil {
		t.Fatal("expected empty prompt error")
	}
}

func containsArgPair(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}
