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
	"strings"

	"github.com/go-json-experiment/json"
	"github.com/kaptinlin/jsonrepair"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

// Prompt defines the structure of the prompt used for summarizing content.
type Prompt struct {
	System string
	User   string
}

// OpenAIClient defines the interface for interacting with OpenAI's API to summarize and generate titles and descriptions contents.
type OpenAIClient interface {
	SummarizeContent(ctx context.Context, prompt Prompt, content string) (title, description string, err error)
}

type openaiClient struct {
	client           *openai.Client
	model            string
	maxContentLength int
	logger           *slog.Logger
}

var _ OpenAIClient = (*openaiClient)(nil)

// NewOpenAIClient creates a new instance of [OpenAIClient] given the API key, model, maximum content length and request options.
func NewOpenAIClient(apiKey, model string, maxContentLength int, opts ...option.RequestOption) *openaiClient {
	client := openai.NewClient(option.WithAPIKey(apiKey))

	return &openaiClient{
		client:           &client,
		model:            model,
		maxContentLength: maxContentLength,
		logger:           slog.Default(),
	}
}

// SummarizeContent summarizes and generates a title and description for the given uri and content using OpenAI LLM model.
func (o *openaiClient) SummarizeContent(ctx context.Context, prompt Prompt, content string) (title, description string, err error) {
	o.logger.DebugContext(ctx, "Summarizes description",
		slog.Group("prompt",
			slog.String("system", prompt.System),
			slog.String("user", prompt.User),
		),
	)

	if o.maxContentLength > 0 && len(content) > o.maxContentLength {
		content = content[:o.maxContentLength]
	}

	params := openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(prompt.System),
			openai.UserMessage(fmt.Sprintf("%s\n\nPage content:\n%s", prompt.User, content)),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfText: openai.Ptr(shared.NewResponseFormatTextParam()),
		},
	}
	switch {
	case strings.HasPrefix(o.model, "gpt"):
		params.MaxTokens = openai.Int(25000)
	case strings.HasPrefix(o.model, "o"):
		params.MaxCompletionTokens = openai.Int(25000)
		params.ReasoningEffort = openai.ReasoningEffortHigh
	}

	chatCompletion, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		o.logger.ErrorContext(ctx, "Failed to generate description", slog.Any("error", err))
		return "", "", fmt.Errorf("generate description: %w", err)
	}
	if len(chatCompletion.Choices) == 0 {
		o.logger.ErrorContext(ctx, "No choices returned from OpenAI")
		return "", "", fmt.Errorf("no choices returned")
	}

	content = chatCompletion.Choices[0].Message.Content
	if content == "" {
		o.logger.ErrorContext(ctx, "Empty content returned from OpenAI")
		return "", "", fmt.Errorf("empty content returned for")
	}

	content, err = jsonrepair.JSONRepair(content)
	if err != nil {
		o.logger.ErrorContext(ctx, "Repair JSON payload dailed", slog.Any("error", err))
	}

	var result DescriptionRequest
	opts := json.JoinOptions(
		json.DiscardUnknownMembers(true), // strictly parsing
	)
	if err := json.UnmarshalRead(strings.NewReader(content), &result, opts); err != nil {
		o.logger.ErrorContext(ctx, "Failed to parse JSON response", slog.String("content", content), slog.Any("error", err))
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

type DescriptionRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}
