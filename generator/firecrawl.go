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
	"fmt"
	"log/slog"

	firecrawl "github.com/mendableai/firecrawl-go/v2"
)

const firecrawlAPIURL = "https://api.firecrawl.dev"

type firecrawlClient struct {
	client *firecrawl.FirecrawlApp
	logger *slog.Logger
}

// NewFirecrawlClient initializes a new [*firecrawl.FirecrawlApp] given an API key.
func NewFirecrawlClient(apiKey string) (FirecrawlClient, error) {
	logger := slog.Default()

	client, err := firecrawl.NewFirecrawlApp(apiKey, firecrawlAPIURL)
	if err != nil {
		logger.Error("Failed to initialize FirecrawlApp", "error", err)
		return nil, fmt.Errorf("initialize FirecrawlApp: %w", err)
	}

	return &firecrawlClient{
		client: client,
		logger: logger,
	}, nil
}

func (f *firecrawlClient) MapWebsite(ctx context.Context, url string, limit int, options FirecrawlOptions) ([]string, error) {
	f.logger.InfoContext(ctx, "Mapping website", "url", url, "limit", limit)

	mapParams := &firecrawl.MapParams{
		Limit:             &limit,
		IncludeSubdomains: &options.IncludeSubdomains,
		IgnoreSitemap:     &options.IgnoreSitemap,
	}

	mapResponse, err := f.client.MapURL(url, mapParams)
	if err != nil {
		f.logger.ErrorContext(ctx, "Failed to map website", "url", url, "error", err)
		return nil, fmt.Errorf("map website %s: %w", url, err)
	}

	if !mapResponse.Success {
		f.logger.ErrorContext(ctx, "Map request was not successful", "url", url)
		return nil, fmt.Errorf("map request failed for %s", url)
	}

	urls := mapResponse.Links
	f.logger.InfoContext(ctx, "Found URLs", "count", len(urls))
	return urls, nil
}

func (f *firecrawlClient) ScrapeURL(ctx context.Context, url string, options FirecrawlOptions) (*ScrapedData, error) {
	f.logger.DebugContext(ctx, "Scraping URL", "url", url)

	scrapeParams := &firecrawl.ScrapeParams{
		Formats:         options.Formats,
		OnlyMainContent: &options.OnlyMainContent,
		Timeout:         &options.Timeout,
	}

	scrapeResponse, err := f.client.ScrapeURL(url, scrapeParams)
	if err != nil {
		f.logger.ErrorContext(ctx, "Failed to scrape URL", "url", url, "error", err)
		return nil, fmt.Errorf("scrape %s: %w", url, err)
	}

	if scrapeResponse.Markdown == "" {
		f.logger.ErrorContext(ctx, "No markdown content returned from scrape", "url", url)
		return nil, fmt.Errorf("no markdown content returned for %s", url)
	}

	metadata := make(map[string]string)
	if scrapeResponse.Metadata != nil {
		if scrapeResponse.Metadata.Title != nil {
			metadata["title"] = *scrapeResponse.Metadata.Title
		}
		if scrapeResponse.Metadata.Description != nil {
			// Handle StringOrStringSlice type - take first value if it's a slice
			desc := *scrapeResponse.Metadata.Description
			if len(desc) > 0 {
				metadata["description"] = desc[0]
			}
		}
	}

	return &ScrapedData{
		URL:      url,
		Markdown: scrapeResponse.Markdown,
		Metadata: metadata,
	}, nil
}
