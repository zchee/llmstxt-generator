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

// OpenAI models from https://github.com/openai/openai-go/blob/v2.0.2/shared/shared.go#L23-L86.
// "gpt-5"
// "gpt-5-mini"
// "gpt-5-nano"
// "gpt-5-2025-08-07"
// "gpt-5-mini-2025-08-07"
// "gpt-5-nano-2025-08-07"
// "gpt-5-chat-latest"
// "gpt-4.1"
// "gpt-4.1-mini"
// "gpt-4.1-nano"
// "gpt-4.1-2025-04-14"
// "gpt-4.1-mini-2025-04-14"
// "gpt-4.1-nano-2025-04-14"
// "o4-mini"
// "o4-mini-2025-04-16"
// "o3"
// "o3-2025-04-16"
// "o3-mini"
// "o3-mini-2025-01-31"
// "o1"
// "o1-2024-12-17"
// "o1-preview"
// "o1-preview-2024-09-12"
// "o1-mini"
// "o1-mini-2024-09-12"
// "gpt-4o"
// "gpt-4o-2024-11-20"
// "gpt-4o-2024-08-06"
// "gpt-4o-2024-05-13"
// "gpt-4o-audio-preview"
// "gpt-4o-audio-preview-2024-10-01"
// "gpt-4o-audio-preview-2024-12-17"
// "gpt-4o-audio-preview-2025-06-03"
// "gpt-4o-mini-audio-preview"
// "gpt-4o-mini-audio-preview-2024-12-17"
// "gpt-4o-search-preview"
// "gpt-4o-mini-search-preview"
// "gpt-4o-search-preview-2025-03-11"
// "gpt-4o-mini-search-preview-2025-03-11"
// "chatgpt-4o-latest"
// "codex-mini-latest"
// "gpt-4o-mini"
// "gpt-4o-mini-2024-07-18"
// "gpt-4-turbo"
// "gpt-4-turbo-2024-04-09"
// "gpt-4-0125-preview"
// "gpt-4-turbo-preview"
// "gpt-4-1106-preview"
// "gpt-4-vision-preview"
// "gpt-4"
// "gpt-4-0314"
// "gpt-4-0613"
// "gpt-4-32k"
// "gpt-4-32k-0314"
// "gpt-4-32k-0613"
// "gpt-3.5-turbo"
// "gpt-3.5-turbo-16k"
// "gpt-3.5-turbo-0301"
// "gpt-3.5-turbo-0613"
// "gpt-3.5-turbo-1106"
// "gpt-3.5-turbo-0125"
// "gpt-3.5-turbo-16k-0613"

package gollm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-json-experiment/json"
	"github.com/kaptinlin/jsonrepair"
	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/shared"
)

// OpenAIConfig contains the configuration for the OpenAI client.
type OpenAIConfig struct {
	Config
}

type openaiClient struct {
	client           *openai.Client
	model            string
	maxContentLength int
	logger           *slog.Logger
}

var _ SummarizerClient = (*openaiClient)(nil)

// NewOpenAIClient creates a new instance of [SummarizerClient] given the API key, model, maximum content length and request options.
func NewOpenAIClient(apiKey, model string, maxContentLength int, opts ...option.RequestOption) *openaiClient {
	client := openai.NewClient(option.WithAPIKey(apiKey))

	return &openaiClient{
		client:           &client,
		model:            model,
		maxContentLength: maxContentLength,
		logger:           slog.Default().WithGroup("openai"),
	}
}

type DescriptionRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// SummarizeContent summarizes and generates a title and description for the given uri and content using OpenAI LLM model.
//
// SummarizeContent implements [SummarizerClient].
func (c *openaiClient) SummarizeContent(ctx context.Context, prompt Prompt, content string) (title, description string, err error) {
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

	params := openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt.System),
			openai.UserMessage(fmt.Sprintf("%s\n\nPage content:\n%s", prompt.User, content)),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfText: openai.Ptr(shared.NewResponseFormatTextParam()),
		},
		Verbosity:           openai.ChatCompletionNewParamsVerbosityHigh,
		MaxCompletionTokens: openai.Int(25000), // https://platform.openai.com/docs/guides/reasoning#allocating-space-for-reasoning
	}
	switch {
	case strings.HasPrefix(c.model, "gpt-5"):
		params.ReasoningEffort = openai.ReasoningEffortLow

	case strings.HasPrefix(c.model, "gpt"):
		// nothing to do

	case strings.HasPrefix(c.model, "o"):
		params.ReasoningEffort = openai.ReasoningEffortHigh
	}

	chatCompletion, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		c.logger.ErrorContext(ctx, "Failed to generate description", slog.Any("error", err))
		return "", "", fmt.Errorf("generate description: %w", err)
	}
	if len(chatCompletion.Choices) == 0 {
		c.logger.ErrorContext(ctx, "No choices returned from OpenAI")
		return "", "", fmt.Errorf("no choices returned")
	}

	content = chatCompletion.Choices[0].Message.Content
	if content == "" {
		c.logger.ErrorContext(ctx, "Empty content returned from OpenAI")
		return "", "", fmt.Errorf("empty content returned for")
	}

	content, err = jsonrepair.JSONRepair(content)
	if err != nil {
		c.logger.ErrorContext(ctx, "Repair JSON payload dailed", slog.Any("error", err))
	}

	var result DescriptionRequest
	opts := json.JoinOptions(
		json.DiscardUnknownMembers(true), // strictly parsing
	)
	if err := json.UnmarshalRead(strings.NewReader(content), &result, opts); err != nil {
		c.logger.ErrorContext(ctx, "Failed to parse JSON response", slog.String("content", content), slog.Any("error", err))
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

	return title, description, nil
}
