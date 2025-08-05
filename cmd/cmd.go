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

// Package cmd provides the command-line interface for the llmstxt-generator tool.
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zchee/llmstxt-generator/config"
	"github.com/zchee/llmstxt-generator/generator"
	"github.com/zchee/llmstxt-generator/gollm"
)

var llmstxtGeneratorCmd = &cobra.Command{
	Use:   "llmstxt-generator <url>",
	Short: "Generate llms.txt and llms-full.txt files for websites using Firecrawl",
	Long: `Go implementation of the llms.txt generator that uses Firecrawl to map and scrape websites,
and OpenAI to generate titles and descriptions for creating structured llms.txt files.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return generate(cmd, args)
	},
}

// Execute executes the [llmstxtGeneratorCmd] root command.
func Execute() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return llmstxtGeneratorCmd.ExecuteContext(ctx)
}

func setupLogger(verbose bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	if verbose {
		opts.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, opts))
	slog.SetDefault(logger)

	return logger
}

func maskVal(s string) (string, func() string) {
	sz := len(s)>>5 + len(s)>>4

	restore := func() string { return s }
	start := s[:sz]
	end := s[len(s)-sz:]

	c := len(s) - len(start) - len(end)
	if c <= 0 {
		c = 8
	}
	masked := start + strings.Repeat("*", c) + end

	return masked, restore
}

var cfg *config.Config

func init() {
	cfg = config.New()

	// masking sensitive API values
	fireCrawlAPIKey, restoreFC := maskVal(cfg.FirecrawlAPIKey)
	defer func() { cfg.FirecrawlAPIKey = restoreFC() }()

	var apiKey string
	if cfg.APIKey != "" {
		llmAPIKey, restoreAPIKey := maskVal(cfg.APIKey)
		apiKey = llmAPIKey
		defer func() { cfg.APIKey = restoreAPIKey() }()
	}

	llmstxtGeneratorCmd.Flags().StringVar(&cfg.Model, "model", cfg.Model, "LLM model for summaries and generating concise titles and descriptions")
	llmstxtGeneratorCmd.Flags().IntVar(&cfg.MaxURLs, "max-urls", cfg.MaxURLs, "Maximum number of URLs to process")
	llmstxtGeneratorCmd.Flags().StringVar(&cfg.OutputDir, "output-dir", cfg.OutputDir, "Directory to save output files")
	llmstxtGeneratorCmd.Flags().StringVar(&cfg.FirecrawlAPIKey, "firecrawl-api-key", fireCrawlAPIKey, "Firecrawl API key")
	llmstxtGeneratorCmd.Flags().StringVar(&cfg.APIKey, "api-key", apiKey, "LLM client API key")
	llmstxtGeneratorCmd.Flags().BoolVar(&cfg.NoFullText, "no-full-text", cfg.NoFullText, "Don't generate llms-full.txt file")
	llmstxtGeneratorCmd.Flags().BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "Enable verbose logging")
	llmstxtGeneratorCmd.Flags().IntVar(&cfg.BatchSize, "batch-size", cfg.BatchSize, "Number of URLs to process in each batch")
	llmstxtGeneratorCmd.Flags().IntVar(&cfg.MaxWorkers, "max-workers", cfg.MaxWorkers, "Maximum number of concurrent workers")
	llmstxtGeneratorCmd.Flags().DurationVar(&cfg.BatchDelay, "batch-delay", cfg.BatchDelay, "Delay between batches")
	llmstxtGeneratorCmd.Flags().DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Timeout for individual URL processing")
	llmstxtGeneratorCmd.Flags().IntVar(&cfg.MaxContentLength, "max-content-length", cfg.MaxContentLength, "Maximum content length for OpenAI processing (0 for unlimited)")
}

// OpenAI:
// - "chatgpt-4o-latest"
// - "codex-mini-latest"
// - "gpt-3.5-turbo"
// - "gpt-3.5-turbo-0125"
// - "gpt-3.5-turbo-0301"
// - "gpt-3.5-turbo-0613"
// - "gpt-3.5-turbo-1106"
// - "gpt-3.5-turbo-16k"
// - "gpt-3.5-turbo-16k-0613"
// - "gpt-4"
// - "gpt-4-0125-preview"
// - "gpt-4-0314"
// - "gpt-4-0613"
// - "gpt-4.1"
// - "gpt-4-1106-preview"
// - "gpt-4.1-2025-04-14"
// - "gpt-4.1-mini"
// - "gpt-4.1-mini-2025-04-14"
// - "gpt-4.1-nano"
// - "gpt-4.1-nano-2025-04-14"
// - "gpt-4-32k"
// - "gpt-4-32k-0314"
// - "gpt-4-32k-0613"
// - "gpt-4o"
// - "gpt-4o-2024-05-13"
// - "gpt-4o-2024-08-06"
// - "gpt-4o-2024-11-20"
// - "gpt-4o-audio-preview"
// - "gpt-4o-audio-preview-2024-10-01"
// - "gpt-4o-audio-preview-2024-12-17"
// - "gpt-4o-audio-preview-2025-06-03"
// - "gpt-4o-mini"
// - "gpt-4o-mini-2024-07-18"
// - "gpt-4o-mini-audio-preview"
// - "gpt-4o-mini-audio-preview-2024-12-17"
// - "gpt-4o-mini-search-preview"
// - "gpt-4o-mini-search-preview-2025-03-11"
// - "gpt-4o-search-preview"
// - "gpt-4o-search-preview-2025-03-11"
// - "gpt-4-turbo"
// - "gpt-4-turbo-2024-04-09"
// - "gpt-4-turbo-preview"
// - "gpt-4-vision-preview"
// - "o1"
// - "o1-2024-12-17"
// - "o1-mini"
// - "o1-mini-2024-09-12"
// - "o1-preview"
// - "o1-preview-2024-09-12"
// - "o3"
// - "o3-2025-04-16"
// - "o3-mini"
// - "o3-mini-2025-01-31"
// - "o4-mini"
// - "o4-mini-2025-04-16"
//
// Anthropic:
// - "claude-3-5-haiku-20241022"
// - "claude-3-5-haiku-latest"
// - "claude-3-5-sonnet-20241022"
// - "claude-3-5-sonnet-latest"
// - "claude-3-7-sonnet-20250219"
// - "claude-3-7-sonnet-latest"
// - "claude-4-opus-20250514"
// - "claude-4-sonnet-20250514"
// - "claude-3-5-sonnet-20240620"
// - "claude-opus-4-0"
// - "claude-opus-4-1-20250805"
// - "claude-opus-4-20250514"
// - "claude-sonnet-4-0"
// - "claude-sonnet-4-20250514"

func detectClientFromModel(cfg *config.Config) gollm.SummarizerClient {
	var summarizerFunc func(apiKey string) gollm.SummarizerClient

	isOpenAI := func(model string) bool {
		openAIModelPrefixes := []string{
			"chatgpt-",
			"codex-",
			"gpt-",
			"o1",
			"o3",
			"o4",
		}
		for _, prefix := range openAIModelPrefixes {
			if strings.HasPrefix(model, prefix) {
				return true
			}
		}
		return false
	}

	switch {
	case strings.HasPrefix(cfg.Model, "claude-"):
		if cfg.APIKey == "" {
			cfg.APIKey = cfg.AnthropicOption.APIKey
		}
		summarizerFunc = func(apiKey string) gollm.SummarizerClient {
			return gollm.NewAnthropicClient(apiKey, cfg.Model, cfg.MaxContentLength)
		}

	case isOpenAI(cfg.Model):
		if cfg.APIKey == "" {
			cfg.APIKey = cfg.OpenAIOption.APIKey
		}
		summarizerFunc = func(apiKey string) gollm.SummarizerClient {
			return gollm.NewOpenAIClient(apiKey, cfg.Model, cfg.MaxContentLength)
		}

	default:
		panic(fmt.Errorf("unkonwn model: %v", cfg.Model))
	}

	return summarizerFunc(cfg.APIKey)
}

func generate(cmd *cobra.Command, args []string) (err error) {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	targetURL := args[0]
	targetURL, err = normalizeURL(targetURL)
	if err != nil {
		return fmt.Errorf("normalize URL: %w", err)
	}

	stat, err := os.Stat(cfg.OutputDir)
	if err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}
	if !stat.IsDir() {
		return fmt.Errorf("output-dir exist but not directory: %w", err)
	}

	logger := setupLogger(cfg.Verbose)

	firecrawlClient, err := generator.NewFirecrawlClient(cfg.FirecrawlAPIKey)
	if err != nil {
		return err
	}

	client := detectClientFromModel(cfg)
	options := generator.GenerationOptions{
		Model:            cfg.Model,
		MaxURLs:          cfg.MaxURLs,
		OutputDir:        cfg.OutputDir,
		NoFullText:       cfg.NoFullText,
		Verbose:          cfg.Verbose,
		BatchSize:        cfg.BatchSize,
		MaxWorkers:       cfg.MaxWorkers,
		BatchDelay:       cfg.BatchDelay,
		Timeout:          cfg.Timeout,
		MaxContentLength: cfg.MaxContentLength,
		FirecrawlOptions: cfg.FirecrawlOptions,
	}

	gen := generator.NewLLMsTxtGenerator(firecrawlClient, client, options)

	result, err := gen.GenerateLLMsTXT(cmd.Context(), targetURL)
	if err != nil {
		return fmt.Errorf("generate llms.txt: %w", err)
	}

	domain, err := generator.ParseDomainFromURL(targetURL)
	if err != nil {
		return fmt.Errorf("extract domain from URL: %w", err)
	}

	llmsTxtPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("%s-llms.txt", domain))
	if err := os.WriteFile(llmsTxtPath, []byte(result.LLMsTxt), 0644); err != nil {
		return fmt.Errorf("write llms.txt file: %w", err)
	}
	logger.InfoContext(cmd.Context(), "Saved llms.txt", "path", llmsTxtPath)

	if !cfg.NoFullText {
		llmsFullTxtPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("%s-llms-full.txt", domain))
		if err := os.WriteFile(llmsFullTxtPath, []byte(result.LLMsFullTxt), 0644); err != nil {
			return fmt.Errorf("write llms-full.txt file: %w", err)
		}

		logger.InfoContext(cmd.Context(), "Saved llms-full.txt", "path", llmsFullTxtPath)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nSuccess! Processed %d out of %d URLs\n", result.ProcessedCount, result.TotalCount)
	fmt.Fprintf(cmd.OutOrStdout(), "Files saved to %s/\n", cfg.OutputDir)

	return nil
}

func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength]
}

func CleanText(text string) string {
	text = strings.TrimSpace(text)

	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	return text
}

func RemovePageSeparators(text string) string {
	re := regexp.MustCompile(`<\|firecrawl-page-\d+-lllmstxt\|>\n`)
	return re.ReplaceAllString(text, "")
}

func normalizeURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}

	if parsedURL.Scheme == "" {
		return "", fmt.Errorf("URL must include a scheme (http:// or https://)")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("URL scheme must be http or https")
	}

	if parsedURL.Host == "" {
		return "", fmt.Errorf("URL must include a host")
	}

	return parsedURL.String(), nil
}
