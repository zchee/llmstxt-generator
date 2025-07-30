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

// Package config provides configuration management for the llmstxt-generator.
//
// It handles:
//   - Environment variable loading
//   - Default value configuration
//   - Configuration validation
//   - API key management
//   - Processing parameters (timeouts, batch sizes, etc.)
//   - Firecrawl and OpenAI client configuration options
package config

import (
	"fmt"
	"os"
	"time"

	openai "github.com/openai/openai-go"

	"github.com/zchee/llmstxt-generator/generator"
)

// Config represents the configuration for the llmstxt-generator.
type Config struct {
	FirecrawlAPIKey  string
	OpenAIAPIKey     string
	OpenAIModel      string
	MaxURLs          int
	OutputDir        string
	NoFullText       bool
	Verbose          bool
	BatchSize        int
	MaxWorkers       int
	BatchDelay       time.Duration
	Timeout          time.Duration
	MaxContentLength int
	FirecrawlOptions generator.FirecrawlOptions
}

// New returns the default configuration for the llmstxt-generator.
func New() *Config {
	return &Config{
		FirecrawlAPIKey: os.Getenv("FIRECRAWL_API_KEY"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:     openai.ChatModelGPT4_1Mini,
		MaxURLs:         20,
		OutputDir:       ".",
		NoFullText:      false,
		Verbose:         false,
		BatchSize:       10,
		MaxWorkers:      5,
		BatchDelay:      time.Second,
		Timeout:         30 * time.Second,
		// TODO(zchee): `4000` default value is the same as [mendableai/create-llmstxt-py](https://github.com/mendableai/create-llmstxt-py) for the moment.
		// See https://github.com/mendableai/create-llmstxt-py/blob/c015913a7e71/generate-llmstxt.py#L133
		MaxContentLength: 4000,
		FirecrawlOptions: generator.FirecrawlOptions{
			OnlyMainContent:   true,                 // Default to previous hard-coded value
			Timeout:           30000,                // Default to previous hard-coded value (30 seconds in ms)
			Formats:           []string{"markdown"}, // Default to previous hard-coded value
			IncludeSubdomains: false,                // Default conservative setting
			IgnoreSitemap:     false,                // Default conservative setting
		},
	}
}

// Validate validates for each [Config] field value.
func (c *Config) Validate() error {
	if c.FirecrawlAPIKey == "" {
		return fmt.Errorf("Firecrawl API key not provided. Set FIRECRAWL_API_KEY environment variable or use --firecrawl-api-key flag")
	}

	if c.OpenAIAPIKey == "" {
		return fmt.Errorf("OpenAI API key not provided. Set OPENAI_API_KEY environment variable or use --openai-api-key flag")
	}

	if c.MaxURLs <= 0 {
		return fmt.Errorf("max-urls must be greater than 0")
	}

	if c.BatchSize <= 0 {
		return fmt.Errorf("batch-size must be greater than 0")
	}

	if c.MaxWorkers <= 0 {
		return fmt.Errorf("max-workers must be greater than 0")
	}

	if c.MaxContentLength < 0 {
		return fmt.Errorf("max-content-length must be greater than or equal to 0")
	}

	return nil
}
