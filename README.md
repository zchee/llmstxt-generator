# llmstxt-generator

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-00ADD8.svg)](https://go.dev/)

A high-performance Go implementation of the llms.txt generator that uses [Firecrawl](https://www.firecrawl.dev/) to map and scrape websites, and any LLM providers (Currently only supports [OpenAI](https://openai.com/)) to generate concise titles and descriptions for creating structured llms.txt files.

> [!IMPORTANT]
> This project is in the alpha stage.
>
> Flags, configuration, behavior, and design may change significantly.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [What is llms.txt?](#what-is-llmstxt)
- [Installation](#installation)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Output Format](#output-format)
- [Performance](#performance)
- [API Documentation](#api-documentation)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [Acknowledgments](#acknowledgments)
- [License](#license)

## Overview

`llmstxt-generator` is a command-line tool that automatically generates `llms.txt` and `llms-full.txt` files from any website.

It intelligently crawls websites, extracts content, and uses AI to create meaningful summaries that help LLMs understand and navigate your site's structure.

### Key Benefits

- **Automated Discovery**: Automatically maps your entire website structure
- **AI-Powered Summaries**: Uses OpenAI to generate concise, meaningful descriptions
- **Performance Optimized**: Concurrent processing with configurable batching and rate limiting
- **Flexible Output**: Generates both summary (`llms.txt`) and full content (`llms-full.txt`) versions

## Features

- 🚀 **High-Performance Concurrent Processing**: Process multiple URLs simultaneously with configurable worker pools
- 🤖 **Multiple AI Model Support**: Compatible with GPT-4, GPT-4 Turbo, and other OpenAI models
- 📊 **Intelligent Batching**: Process URLs in configurable batches with automatic rate limiting
- 🔧 **Highly Configurable**: Extensive CLI flags and environment variable support
- 📝 **Dual Output Formats**: Generate both concise summaries and full-text versions
- 🛡️ **Robust Error Handling**: Graceful failure recovery and comprehensive error reporting
- 🔍 **Smart Content Extraction**: Focuses on main content while filtering out navigation and boilerplate
- ⏱️ **Timeout Management**: Configurable timeouts for reliable processing of large sites
- 📈 **Progress Tracking**: Real-time progress updates with detailed logging options

## What is llms.txt?

The `llms.txt` format is a structured way to help Large Language Models (LLMs) understand and navigate websites more effectively. It provides:

- **llms.txt**: A concise index with titles, URLs, and brief descriptions
- **llms-full.txt**: Complete content from all pages for comprehensive context

This standardized format enables LLMs to quickly understand site structure, find relevant information, and provide better assistance to users asking about your website.

- [The /llms.txt file – llms-txt](https://llmstxt.org/)
    - Official llms.txt specification
    - [AnswerDotAI/llms-txt: The /llms.txt file, helping language models use your website](https://github.com/AnswerDotAI/llms-txt)

## Installation

### From Source

```bash
# Requires Go 1.24 or higher
go install github.com/zchee/llmstxt-generator@latest
```

### Build from Repository

```bash
git clone https://github.com/zchee/llmstxt-generator.git
cd llmstxt-generator
go build -o llmstxt-generator
```

## Prerequisites

Before using llmstxt-generator, you'll need:

1. **Firecrawl API Key**: Sign up at [firecrawl.dev](https://www.firecrawl.dev/) to get your API key
2. **OpenAI API Key**: Get your API key from [OpenAI Platform](https://platform.openai.com/)
3. **Go 1.24+**: Required if building from source

### Setting up API Keys

Set your API keys as environment variables:

```bash
export FIRECRAWL_API_KEY="your-firecrawl-api-key"
export OPENAI_API_KEY="your-openai-api-key"
```

Or pass them directly via command-line flags.

## Quick Start

Generate llms.txt files for a website:

```bash
# Basic usage
llmstxt-generator https://example.com

# With custom output directory
llmstxt-generator https://example.com --output-dir ./output

# Process more URLs with higher concurrency
llmstxt-generator https://example.com --max-urls 100 --max-workers 10

# Use a specific OpenAI model
llmstxt-generator https://example.com --model gpt-4-turbo-preview
```

## Configuration

### Command-Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--model` | OpenAI model for generating summaries | `gpt-4.1-mini` |
| `--max-urls` | Maximum number of URLs to process | `20` |
| `--output-dir` | Directory to save output files | `.` (current) |
| `--firecrawl-api-key` | Firecrawl API key | `$FIRECRAWL_API_KEY` |
| `--openai-api-key` | OpenAI API key | `$OPENAI_API_KEY` |
| `--no-full-text` | Skip generating llms-full.txt | `false` |
| `--verbose` | Enable verbose logging | `false` |
| `--batch-size` | Number of URLs per batch | `10` |
| `--max-workers` | Maximum concurrent workers | `5` |
| `--batch-delay` | Delay between batches | `1s` |
| `--timeout` | Timeout for URL processing | `30s` |
| `--max-content-length` | Max content length for OpenAI | `4000` |

### Environment Variables

- `FIRECRAWL_API_KEY`: Your Firecrawl API key
- `OPENAI_API_KEY`: Your OpenAI API key

## Usage Examples

### Basic Website Processing

```bash
# Generate files for a simple website
llmstxt-generator https://myblog.com
```

### Large Website with Custom Settings

```bash
# Process up to 500 URLs with increased concurrency
llmstxt-generator https://docs.example.com \
  --max-urls 500 \
  --max-workers 20 \
  --batch-size 50 \
  --output-dir ./documentation \
  --verbose
```

### Production Deployment

```bash
# Production settings with timeouts and rate limiting
llmstxt-generator https://enterprise.example.com \
  --model gpt-4-turbo-preview \
  --max-urls 1000 \
  --max-workers 10 \
  --batch-size 25 \
  --batch-delay 2s \
  --timeout 45s \
  --max-content-length 8000 \
  --output-dir /var/www/llms-files \
  --verbose
```

<!-- ### CI/CD Integration -->
<!---->
<!-- ```yaml -->
<!-- # Example GitHub Actions workflow -->
<!-- - name: Generate llms.txt -->
<!--   env: -->
<!--     FIRECRAWL_API_KEY: ${{ secrets.FIRECRAWL_API_KEY }} -->
<!--     OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }} -->
<!--   run: | -->
<!--     llmstxt-generator https://mysite.com \ -->
<!--       --output-dir ./public \ -->
<!--       --max-urls 100 -->
<!-- ``` -->

## Output Format

### llms.txt Example

```
# https://example.com llms.txt

- [Homepage](https://example.com): Welcome to Example.com - Your trusted source for examples
- [About Us](https://example.com/about): Learn about our mission, team, and company history
- [Products](https://example.com/products): Browse our complete catalog of innovative products
- [Contact](https://example.com/contact): Get in touch with our support team today
```

### llms-full.txt Example

```
# https://example.com llms-full.txt

<|firecrawl-page-1-lllmstxt|>
## Homepage
Welcome to Example.com! We are the leading provider of example services...
[Full page content]

<|firecrawl-page-2-lllmstxt|>
## About Us
Founded in 2020, Example.com has grown to become...
[Full page content]
```

### Key Components

1. **CLI Layer** (`cmd/`): Handles command-line parsing and user interaction
2. **Configuration** (`config/`): Manages settings, validation, and defaults
3. **Generator** (`generator/`): Core business logic for content generation
4. **API Clients**: Abstracted interfaces for Firecrawl and OpenAI services

## Performance

### Optimization Strategies

- **Concurrent Processing**: Utilizes Go's goroutines for parallel URL processing
- **Intelligent Batching**: Reduces API overhead by processing URLs in batches
- **Rate Limiting**: Prevents API throttling with configurable delays
- **Memory Efficiency**: Pre-allocated buffers and efficient string building
- **Context Cancellation**: Proper cleanup and resource management

### Benchmarks

Processing performance varies based on website size and API response times:

- Small sites (< 50 pages): ~1-2 minutes
- Medium sites (50-200 pages): ~5-10 minutes  
- Large sites (200-1000 pages): ~15-30 minutes

*Note: Actual performance depends on API rate limits and network conditions*

## API Documentation

### Generator Package

The main generator provides a simple API for programmatic use:

```go
package main

import (
	"github.com/zchee/llmstxt-generator/generator"
)

func main() {
    // Create firecrawlClient, openaiClient and options...
    // .
    // .
    // .
	// Create a new generator
	gen := generator.NewLLMsTxtGenerator(
		firecrawlClient,
		openaiClient,
		options,
	)
	
	// Generate llms.txt files
	result, err := gen.GenerateLLMsTXT(ctx, "https://example.com")
	if err != nil {
		log.Fatal(err)
	}
	
	// Access generated content
	fmt.Println(result.LLMsTxt)
	fmt.Println(result.LLMsFullTxt)
}
```

<!-- ### Custom Clients -->
<!---->
<!-- Implement custom clients by satisfying the interfaces: -->
<!---->
<!-- ```go -->
<!-- type FirecrawlClient interface { -->
<!-- 	MapWebsite(ctx context.Context, url string, limit int, options FirecrawlOptions) ([]string, error) -->
<!-- 	ScrapeURL(ctx context.Context, url string, options FirecrawlOptions) (*ScrapedData, error) -->
<!-- } -->
<!---->
<!-- type OpenAIClient interface { -->
<!-- 	SummarizeContent(ctx context.Context, prompt Prompt, content string) (title, description string, err error) -->
<!-- } -->
<!-- ``` -->

## Troubleshooting

### Common Issues

#### API Key Errors

```
Error: Firecrawl API key not provided
```

##### **Solution**

Ensure your API keys are set correctly:

```bash
export FIRECRAWL_API_KEY="your-key"
export OPENAI_API_KEY="your-key"
```

#### Rate Limiting

```
Error: API rate limit exceeded
```

##### **Solution**
Increase batch delay or reduce worker count:

```bash
llmstxt-generator https://example.com --batch-delay 5s --max-workers 3
```

#### Timeout Errors
```
Error: Context deadline exceeded
```

##### **Solution**

Increase timeout duration:

```bash
llmstxt-generator https://example.com --timeout 60s
```

#### Memory Issues

For very large sites, consider:
- Processing in smaller batches with `--max-urls`
- Reducing concurrent workers with `--max-workers`
- Increasing `--max-content-length` for better summaries

### Debug Mode

Enable verbose logging for detailed troubleshooting:

```bash
llmstxt-generator https://example.com --verbose
```

## Contributing

We welcome contributions! Please follow these guidelines:

1. **Fork the repository** and create your feature branch
2. **Write tests** for new functionality
3. **Follow Go conventions** and run `go fmt`
4. **Update documentation** for user-facing changes
5. **Submit a pull request** with a clear description

### Development Setup

```bash
# Clone the repository
git clone https://github.com/zchee/llmstxt-generator.git
cd llmstxt-generator

# Install dependencies
go mod download

# Run tests
go test ./...

# Build and run locally
go build -o llmstxt-generator
./llmstxt-generator https://example.com
```

## Acknowledgments

- [mendableai/create-llmstxt-py](https://github.com/mendableai/create-llmstxt-py)
    - This library is a Go port of this repository by [Firecrawl by Mendable](https://github.com/mendableai). Special thanks to the original author for creating such a useful tool.
- The Go community for excellent libraries and tools

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
