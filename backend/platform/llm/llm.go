// Package llm is a narrow client for a text model.
//
// Narrow on purpose: one call, a system prompt and a user prompt, text
// back. Everything this product asks a model to do is "read these
// figures and write two sentences", and an interface that can only do
// that is an interface nobody can quietly use for something riskier.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrNotConfigured means no API key was supplied. Distinct from a
// transport failure: "nobody set this up" and "the provider is down"
// want different responses, and a feature that is simply switched off
// should not look like an outage.
var ErrNotConfigured = errors.New("llm: not configured")

type Client interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

// Anthropic talks to the Messages API.
type Anthropic struct {
	APIKey string
	Model  string
	// MaxTokens caps the reply. These are two-sentence explanations, so
	// it is small deliberately: a cap is the cheapest guard against a
	// model that decides to write an essay, and against the bill that
	// would come with it.
	MaxTokens int
	Timeout   time.Duration
	HTTP      *http.Client
	BaseURL   string
}

func (a Anthropic) Configured() bool { return a.APIKey != "" }

func (a Anthropic) client() *http.Client {
	if a.HTTP != nil {
		return a.HTTP
	}
	return &http.Client{Timeout: a.timeout()}
}

func (a Anthropic) timeout() time.Duration {
	if a.Timeout > 0 {
		return a.Timeout
	}
	return 20 * time.Second
}

type msgRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []msgContent `json:"messages"`
}

type msgContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type msgResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (a Anthropic) Complete(ctx context.Context, system, user string) (string, error) {
	if !a.Configured() {
		return "", ErrNotConfigured
	}
	model := a.Model
	if model == "" {
		model = "claude-sonnet-5"
	}
	maxTokens := a.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 400
	}
	base := a.BaseURL
	if base == "" {
		base = "https://api.anthropic.com"
	}

	body, err := json.Marshal(msgRequest{
		Model: model, MaxTokens: maxTokens, System: system,
		Messages: []msgContent{{Role: "user", Content: user}},
	})
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, a.timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var out msgResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("llm: malformed response: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("llm: %s: %s", out.Error.Type, out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: provider returned %s", resp.Status)
	}
	var text string
	for _, c := range out.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}
	if text == "" {
		return "", errors.New("llm: provider returned no text")
	}
	return text, nil
}
