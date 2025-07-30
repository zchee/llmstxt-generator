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

package generator

import (
	"context"
	"time"

	"github.com/zchee/llmstxt-generator/gollm"
)

type ScrapedData struct {
	URL      string            `json:"url"`
	Markdown string            `json:"markdown"`
	Metadata map[string]string `json:"metadata"`
}

type ProcessedURL struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Markdown    string `json:"markdown"`
	Index       int    `json:"index"`
}

type GenerationResult struct {
	LLMsTxt        string `json:"llms_txt"`
	LLMsFullTxt    string `json:"llms_full_txt"`
	ProcessedCount int    `json:"processed_count"`
	TotalCount     int    `json:"total_count"`
}

type FirecrawlOptions struct {
	OnlyMainContent   bool
	Timeout           int
	Formats           []string
	IncludeSubdomains bool
	IgnoreSitemap     bool
}

type GenerationOptions struct {
	Model            string
	MaxURLs          int
	OutputDir        string
	NoFullText       bool
	Verbose          bool
	BatchSize        int
	MaxWorkers       int
	BatchDelay       time.Duration
	Timeout          time.Duration
	MaxContentLength int
	FirecrawlOptions FirecrawlOptions
}

type FirecrawlClient interface {
	MapWebsite(ctx context.Context, url string, limit int, options FirecrawlOptions) ([]string, error)
	ScrapeURL(ctx context.Context, url string, options FirecrawlOptions) (*ScrapedData, error)
}

type LLMsTxtGenerator struct {
	firecrawlClient FirecrawlClient
	openaiClient    gollm.OpenAIClient
	options         GenerationOptions
}

type MapResponse struct {
	Success bool     `json:"success"`
	Links   []string `json:"links"`
}

type ScrapeResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Markdown string            `json:"markdown"`
		Metadata map[string]string `json:"metadata"`
	} `json:"data"`
}
