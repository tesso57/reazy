// Package ai provides abstractions for AI provider integrations.
package ai

import "context"

// Client is an abstraction over concrete AI providers.
type Client interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
