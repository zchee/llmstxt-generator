// Copyright 2025 The llmstxt-generator Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package gollm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/go-json-experiment/json"
	"github.com/kaptinlin/jsonrepair"
)

// Anthropic models from https://github.com/anthropics/anthropic-sdk-go/blob/v1.9.1/message.go#L1859-L1872.
// "claude-3-7-sonnet-latest"
// "claude-3-7-sonnet-20250219"
// "claude-3-5-haiku-latest"
// "claude-3-5-haiku-20241022"
// "claude-sonnet-4-20250514"
// "claude-sonnet-4-0"
// "claude-4-sonnet-20250514"
// "claude-3-5-sonnet-latest"
// "claude-3-5-sonnet-20241022"
// "claude-3-5-sonnet-20240620"
// "claude-opus-4-0"
// "claude-opus-4-20250514"
// "claude-4-opus-20250514"
// "claude-opus-4-1-20250805"

// AnthropicConfig contains the configuration for the Anthropic client.
type AnthropicConfig struct {
	Config
}

type anthropicClient struct {
	client           anthropic.Client
	model            string
	maxContentLength int
	logger           *slog.Logger
}

var _ SummarizerClient = (*anthropicClient)(nil)

// NewAnthropicClient creates a new instance of [SummarizerClient] given the API key, model, maximum content length and request options.
func NewAnthropicClient(apiKey, model string, maxContentLength int, opts ...option.RequestOption) *anthropicClient {
	cOpts := []option.RequestOption{
		option.WithEnvironmentProduction(),
	}
	if val, ok := os.LookupEnv("ANTHROPIC_BASE_URL"); ok {
		cOpts = append(cOpts, option.WithBaseURL(val))
	}
	if val, ok := os.LookupEnv("ANTHROPIC_API_KEY"); ok {
		cOpts = append(cOpts, option.WithAPIKey(val))
	}
	if val, ok := os.LookupEnv("ANTHROPIC_AUTH_TOKEN"); ok {
		cOpts = append(cOpts, option.WithAuthToken(val))
	}
	// [option.RequestOption] are last win
	cOpts = append(cOpts, opts...)

	client := anthropic.Client{
		Options:     cOpts,
		Completions: anthropic.NewCompletionService(cOpts...),
		Messages:    anthropic.NewMessageService(cOpts...),
		Models:      anthropic.NewModelService(cOpts...),
		Beta:        anthropic.NewBetaService(cOpts...),
	}

	return &anthropicClient{
		client:           client,
		model:            model,
		maxContentLength: maxContentLength,
		logger:           slog.Default().WithGroup("anthropic"),
	}
}

// SummarizeContent summarizes and generates a title and description for the given uri and content using Anthropic LLM model.
//
// SummarizeContent implements [SummarizerClient].
func (c *anthropicClient) SummarizeContent(ctx context.Context, prompt Prompt, content string) (title, description string, err error) {
	c.logger.DebugContext(ctx, "Summarizes description",
		slog.String("model", c.model),
		slog.Group("prompt",
			slog.String("system", prompt.System),
			slog.String("user", prompt.User),
		),
	)

	if c.maxContentLength > 0 && len(content) > c.maxContentLength {
		content = content[:c.maxContentLength]
	}

	params := anthropic.BetaMessageNewParams{
		Model: anthropic.Model(c.model),
		System: []anthropic.BetaTextBlockParam{
			{
				Text: prompt.System,
			},
		},
		Messages: []anthropic.BetaMessageParam{
			{
				Content: []anthropic.BetaContentBlockParamUnion{
					anthropic.NewBetaTextBlock(prompt.User),
					anthropic.NewBetaTextBlock(fmt.Sprintf("Page content:\n%s", content)),
				},
				Role: anthropic.BetaMessageParamRoleUser,
			},
		},
	}

	switch {
	case strings.HasPrefix(c.model, "claude-opus-"):
		params.MaxTokens = 32000
		params.Thinking = anthropic.BetaThinkingConfigParamOfEnabled(25600)

	case strings.HasPrefix(c.model, "claude-sonnet-"):
		params.MaxTokens = 320000
		params.Thinking = anthropic.BetaThinkingConfigParamOfEnabled(256000)
	}

	stream := c.client.Beta.Messages.NewStreaming(ctx, params)

	for stream.Next() {
		data := stream.Current()
		for _, content := range data.Message.Content {
			if content.Text == "" {
				c.logger.ErrorContext(ctx, "Empty content returned from OpenAI")
				return "", "", fmt.Errorf("empty content returned for")
			}

			data, err := jsonrepair.JSONRepair(content.Text)
			if err != nil {
				c.logger.ErrorContext(ctx, "Repair JSON payload dailed", slog.Any("error", err))
			}

			fmt.Printf("data: %#v\n", data)

			var result DescriptionRequest
			jsonOpts := json.JoinOptions(
				json.DiscardUnknownMembers(true), // strictly parsing
			)
			if err := json.UnmarshalRead(strings.NewReader(data), &result, jsonOpts); err != nil {
				c.logger.ErrorContext(ctx, "Failed to parse JSON response", slog.String("content", data), slog.Any("error", err))
				return "", "", fmt.Errorf("parse JSON response: %w", err)
			}

			title = result.Title
			if title == "" {
				title = "Page"
			}
			description = result.Description
			if description == "" {
				description = "No description available"
			}
		}
	}

	if stream.Err() != nil {
		c.logger.ErrorContext(ctx, "Failed to get message with stream", slog.Any("error", stream.Err()))
		return "", "", fmt.Errorf("get message with stream: %w", stream.Err())
	}

	return title, description, nil
}
