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

// Package generator provides functionality for generating llms.txt files
// from website content using Firecrawl and OpenAI APIs.
//
// The package supports:
//   - Website mapping and URL discovery via Firecrawl
//   - Content scraping with configurable parameters
//   - AI-powered title and description generation via OpenAI
//   - Concurrent processing with rate limiting
//   - Generation of both summary (llms.txt) and full content (llms-full.txt) files
package generator

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/zchee/llmstxt-generator/gollm"
)

// NewLLMsTxtGenerator creates a new instance of LLMsTxtGenerator with the provided clients and options.
//
// Parameters:
//   - firecrawlClient: Client for website mapping and content scraping
//   - openaiClient: Client for AI-powered content analysis and description generation
//   - options: Configuration options for generation behavior, timeouts, and processing limits
//
// Returns a configured generator ready to process websites and generate llms.txt files.
func NewLLMsTxtGenerator(firecrawlClient FirecrawlClient, openaiClient gollm.OpenAIClient, options GenerationOptions) *LLMsTxtGenerator {
	return &LLMsTxtGenerator{
		firecrawlClient: firecrawlClient,
		openaiClient:    openaiClient,
		options:         options,
	}
}

const (
	systemPrompt  = `You are a helpful assistant that generates concise titles and descriptions for web pages.`
	userPromptFmt = `Generate a 9-10 word description and a 3-4 word title of the entire page based on ALL the content one will find on the page for this url: %s. This will help in a user finding the page for its intended purpose.

Return the response in JSON format:
{
    "title": "3-4 word title",
    "description": "9-10 word description"
}`
)

func (g *LLMsTxtGenerator) SystemPrompt() string {
	return systemPrompt
}

func (g *LLMsTxtGenerator) UserPrompt(uri string) string {
	return fmt.Sprintf(userPromptFmt, uri)
}

// GenerateLLMsTXT generates both llms.txt and llms-full.txt files from a target URL.
//
// The process includes:
//  1. Mapping the website to discover all available URLs
//  2. Processing URLs in configurable batches with rate limiting
//  3. Scraping content from each URL using Firecrawl
//  4. Generating AI-powered titles and descriptions using OpenAI
//  5. Building structured output files
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - targetURL: The base URL of the website to process
//   - logger: Structured logger for progress tracking and debugging
//
// Returns GenerationResult containing the generated content and processing statistics,
// or an error if the generation process fails.
func (g *LLMsTxtGenerator) GenerateLLMsTXT(ctx context.Context, targetURL string) (*GenerationResult, error) {
	logger := slog.Default()
	logger.InfoContext(ctx, "Generating llms.txt", "url", targetURL)

	urls, err := g.firecrawlClient.MapWebsite(ctx, targetURL, g.options.MaxURLs, g.options.FirecrawlOptions)
	if err != nil {
		return nil, fmt.Errorf("map website: %w", err)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs found for the website")
	}

	if len(urls) > g.options.MaxURLs {
		urls = urls[:g.options.MaxURLs]
	}

	var allResults []ProcessedURL
	var mu sync.Mutex

	batchSize := g.options.BatchSize
	for i := 0; i < len(urls); i += batchSize {
		end := min(i+batchSize, len(urls))
		batch := urls[i:end]

		logger.InfoContext(ctx, "Processing batch", "batch", i/batchSize+1, "total_batches", (len(urls)+batchSize-1)/batchSize)

		batchResults, err := g.processBatch(ctx, batch, i, logger)
		if err != nil {
			logger.ErrorContext(ctx, "Batch processing failed", "batch", i/batchSize+1, "error", err)
		}

		mu.Lock()
		allResults = append(allResults, batchResults...)
		mu.Unlock()

		if i+batchSize < len(urls) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(g.options.BatchDelay):
			}
		}
	}

	slices.SortFunc(allResults, func(url1, url2 ProcessedURL) int {
		return cmp.Compare(url1.Index, url2.Index)
	})

	llmsTxt := g.buildLLMsTxt(targetURL, allResults)
	llmsFullTxt := g.buildLLMsFullTxt(targetURL, allResults)

	return &GenerationResult{
		LLMsTxt:        llmsTxt,
		LLMsFullTxt:    llmsFullTxt,
		ProcessedCount: len(allResults),
		TotalCount:     len(urls),
	}, nil
}

func (g *LLMsTxtGenerator) processBatch(ctx context.Context, urls []string, startIndex int, logger *slog.Logger) ([]ProcessedURL, error) {
	results := make([]ProcessedURL, 0, len(urls))
	var mu sync.Mutex
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(g.options.MaxWorkers)

	for i, url := range urls {
		eg.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			result, err := g.processURL(ctx, url, startIndex+i, logger)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to process URL", "url", url, "error", err)
				return err
			}

			if result != nil {
				mu.Lock()
				results = append(results, *result)
				mu.Unlock()
			}

			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

func (g *LLMsTxtGenerator) processURL(ctx context.Context, uri string, index int, logger *slog.Logger) (*ProcessedURL, error) {
	// Check context before expensive operations
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ctx, cancel := context.WithTimeout(ctx, g.options.Timeout)
	defer cancel()

	scrapedData, err := g.firecrawlClient.ScrapeURL(ctx, uri, g.options.FirecrawlOptions)
	if err != nil || scrapedData == nil || scrapedData.Markdown == "" {
		return nil, fmt.Errorf("scrape URL %s: %w", uri, err)
	}

	// Check context again before OpenAI call
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	prompt := gollm.Prompt{
		System: g.SystemPrompt(),
		User:   g.UserPrompt(uri),
	}
	title, description, err := g.openaiClient.SummarizeContent(ctx, prompt, scrapedData.Markdown)
	if err != nil {
		logger.WarnContext(ctx, "Failed to generate description, using defaults", "url", uri, "error", err)
		title = "Page"
		description = "No description available"
	}

	return &ProcessedURL{
		URL:         uri,
		Title:       title,
		Description: description,
		Markdown:    scrapedData.Markdown,
		Index:       index,
	}, nil
}

func (g *LLMsTxtGenerator) buildLLMsTxt(targetURL string, results []ProcessedURL) string {
	// Pre-calculate capacity to avoid reallocations
	estimatedSize := len(targetURL) + 20 // header size
	for _, result := range results {
		estimatedSize += len(result.Title) + len(result.URL) + len(result.Description) + 10 // format overhead
	}

	var sb strings.Builder
	sb.Grow(estimatedSize)
	sb.WriteString(fmt.Sprintf("# %s llms.txt\n\n", targetURL))

	for _, result := range results {
		sb.WriteString(fmt.Sprintf("- [%s](%s): %s\n", result.Title, result.URL, result.Description))
	}

	return sb.String()
}

func (g *LLMsTxtGenerator) buildLLMsFullTxt(targetURL string, results []ProcessedURL) string {
	// Pre-calculate capacity to avoid reallocations
	estimatedSize := len(targetURL) + 25 // header size
	for i, result := range results {
		estimatedSize += len(result.Title) + len(result.Markdown) + 100 // separator and format overhead
		estimatedSize += len(fmt.Sprintf("%d", i+1))                    // page number length
	}

	var sb strings.Builder
	sb.Grow(estimatedSize)
	sb.WriteString(fmt.Sprintf("# %s llms-full.txt\n\n", targetURL))

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("<|firecrawl-page-%d-lllmstxt|>\n## %s\n%s\n\n", i+1, result.Title, result.Markdown))
	}

	content := sb.String()
	// When NoFullText is true, we want clean content without page separators
	// When NoFullText is false, we want content with page separators for full text version
	if g.options.NoFullText {
		return g.removePageSeparators(content)
	}
	return content
}

func (g *LLMsTxtGenerator) removePageSeparators(text string) string {
	re := regexp.MustCompile(`<\|firecrawl-page-\d+-lllmstxt\|>\n`)
	return re.ReplaceAllString(text, "")
}

func ParseDomainFromURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}

	domain := parsedURL.Hostname()
	domain = strings.TrimPrefix(domain, "www.")
	return domain, nil
}
