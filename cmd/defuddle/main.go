package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	defuddle "github.com/vaayne/go-defuddle"
)

const version = "0.1.0"

const defaultUA = "Mozilla/5.0 (compatible; go-defuddle/1.0; +https://github.com/vaayne/go-defuddle)"

func main() {
	var (
		markdown bool
		jsonOut  bool
		prop     string
		output   string
		debug    bool
		showVer  bool
	)

	flag.BoolVar(&markdown, "markdown", false, "Convert content to markdown format")
	flag.BoolVar(&markdown, "m", false, "Convert content to markdown format (shorthand)")
	flag.BoolVar(&jsonOut, "json", false, "Output as JSON with metadata and content")
	flag.BoolVar(&jsonOut, "j", false, "Output as JSON with metadata (shorthand)")
	flag.StringVar(&prop, "property", "", "Extract a specific property (e.g. title, author, domain)")
	flag.StringVar(&prop, "p", "", "Extract a specific property (shorthand)")
	flag.StringVar(&output, "output", "", "Output file path (default: stdout)")
	flag.StringVar(&output, "o", "", "Output file path (shorthand)")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&showVer, "version", false, "Print version")
	flag.BoolVar(&showVer, "v", false, "Print version (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `go-defuddle — extract article content from web pages

Usage:
  go-defuddle [flags] <source>

Source can be a URL (http/https) or a file path.

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  go-defuddle https://example.com/article
  go-defuddle -m https://example.com/article
  go-defuddle -j page.html
  go-defuddle -p title https://example.com/article
  go-defuddle -m -o output.md https://example.com/article
`)
	}

	flag.Parse()

	if showVer {
		fmt.Printf("go-defuddle %s\n", version)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	source := flag.Arg(0)

	// Fetch or read HTML
	var html string
	var url string

	isURL := strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
	if isURL {
		url = source
		body, err := fetchPage(source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		html = body
	} else {
		data, err := os.ReadFile(source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		html = string(data)
	}

	// Parse
	parser, err := defuddle.NewParser()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer parser.Close()

	opts := &defuddle.Options{
		Markdown: markdown || jsonOut,
		Debug:    debug,
	}

	result, err := parser.Parse(html, url, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check for empty content
	stripped := strings.TrimSpace(stripTags(result.Content))
	if stripped == "" {
		fmt.Fprintf(os.Stderr, "Error: No content could be extracted from %s\n", source)
		os.Exit(1)
	}

	// Format output
	var out string

	if prop != "" {
		out = getProperty(result, prop)
		if out == "" {
			fmt.Fprintf(os.Stderr, "Error: Property %q not found or empty\n", prop)
			os.Exit(1)
		}
	} else if jsonOut {
		data, _ := json.MarshalIndent(result, "", "  ")
		out = string(data)
	} else if markdown {
		out = result.Markdown
	} else {
		out = result.Content
	}

	// Write output
	if output != "" {
		if err := os.WriteFile(output, []byte(out+"\n"), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Output written to %s\n", output)
	} else {
		fmt.Println(out)
	}
}

func fetchPage(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // 5MB limit
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	return string(body), nil
}

func stripTags(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func getProperty(r *defuddle.Result, name string) string {
	switch name {
	case "title":
		return r.Title
	case "author":
		return r.Author
	case "description":
		return r.Description
	case "domain":
		return r.Domain
	case "favicon":
		return r.Favicon
	case "image":
		return r.Image
	case "language":
		return r.Language
	case "published":
		return r.Published
	case "site":
		return r.Site
	case "content":
		return r.Content
	case "markdown":
		return r.Markdown
	case "wordCount":
		return fmt.Sprintf("%d", r.WordCount)
	default:
		return ""
	}
}
