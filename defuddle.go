// Package defuddle extracts main content from web pages as clean HTML or Markdown.
//
// It runs the Defuddle (https://github.com/kepano/defuddle) JavaScript library inside
// a sandboxed QuickJS runtime (via WebAssembly), with Markdown conversion handled
// natively in Go via html-to-markdown.
//
// Basic usage:
//
//	parser, err := defuddle.NewParser()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer parser.Close()
//
//	result, err := parser.Parse(html, "https://example.com/page", nil)
package defuddle

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/fastschema/qjs"
)

//go:embed internal/js/polyfills.js
var polyfillsJS string

//go:embed internal/js/defuddle-bundle.js
var defuddleJS string

// Result holds the parsed output from defuddle.
type Result struct {
	// Content is the extracted main content as clean HTML.
	Content string `json:"content"`
	// Title is the page title.
	Title string `json:"title"`
	// Description is the meta description.
	Description string `json:"description"`
	// Domain is the hostname (e.g. "example.com").
	Domain string `json:"domain"`
	// Favicon is the favicon URL.
	Favicon string `json:"favicon"`
	// Image is the Open Graph or lead image URL.
	Image string `json:"image"`
	// Language is the content language (e.g. "en").
	Language string `json:"language"`
	// Published is the publish date (ISO 8601 when available).
	Published string `json:"published"`
	// Author is the author name.
	Author string `json:"author"`
	// Site is the site name.
	Site string `json:"site"`
	// WordCount is the word count of extracted content.
	WordCount int `json:"wordCount"`
	// ParseTime is the JS-side parse time in milliseconds.
	ParseTime int `json:"parseTime"`
	// MetaTags contains all meta tags from <head>.
	MetaTags []MetaTag `json:"metaTags,omitempty"`
	// SchemaOrgData contains parsed JSON-LD schema.org data.
	SchemaOrgData json.RawMessage `json:"schemaOrgData,omitempty"`
	// Markdown is the content converted to Markdown.
	// Only populated when Options.Markdown is true.
	Markdown string `json:"markdown,omitempty"`
}

// MetaTag represents a single HTML meta tag.
type MetaTag struct {
	Name     *string `json:"name"`
	Property *string `json:"property"`
	Content  string  `json:"content"`
}

// Options controls parsing behavior.
type Options struct {
	// Markdown converts the extracted HTML content to Markdown (Go-side).
	Markdown bool `json:"-"`

	// RemoveSmallImages toggles removal of small/tracking images.
	RemoveSmallImages *bool `json:"removeSmallImages,omitempty"`
	// RemoveHiddenElements toggles removal of hidden DOM elements.
	RemoveHiddenElements *bool `json:"removeHiddenElements,omitempty"`
	// RemoveLowScoring toggles removal of low-scoring content blocks.
	RemoveLowScoring *bool `json:"removeLowScoring,omitempty"`
	// RemoveExactSelectors toggles removal via exact CSS selectors.
	RemoveExactSelectors *bool `json:"removeExactSelectors,omitempty"`
	// RemovePartialSelectors toggles removal via partial class/id matching.
	RemovePartialSelectors *bool `json:"removePartialSelectors,omitempty"`
	// RemoveContentPatterns toggles content-pattern-based removal.
	RemoveContentPatterns *bool `json:"removeContentPatterns,omitempty"`
	// Standardize toggles HTML normalization (headings, code blocks, etc.).
	Standardize *bool `json:"standardize,omitempty"`
	// Debug enables debug output from the defuddle pipeline.
	Debug bool `json:"debug,omitempty"`
}

// Parser wraps a QuickJS runtime with the defuddle bundle pre-loaded.
//
// A Parser is safe for sequential use but NOT for concurrent use from
// multiple goroutines. For concurrent workloads, create one Parser per
// goroutine or use a sync.Pool.
type Parser struct {
	rt *qjs.Runtime
	mu sync.Mutex
}

// NewParser creates a new Parser instance. This loads the QuickJS WebAssembly
// runtime and evaluates the defuddle JS bundle (~450ms cold start). Reuse the
// parser across multiple Parse calls to amortize this cost.
func NewParser() (*Parser, error) {
	rt, err := qjs.New()
	if err != nil {
		return nil, fmt.Errorf("qjs.New: %w", err)
	}

	ctx := rt.Context()

	if _, err := ctx.Eval("polyfills.js", qjs.Code(polyfillsJS)); err != nil {
		rt.Close()
		return nil, fmt.Errorf("load polyfills: %w", err)
	}

	if _, err := ctx.Eval("defuddle-bundle.js", qjs.Code(defuddleJS)); err != nil {
		rt.Close()
		return nil, fmt.Errorf("load defuddle bundle: %w", err)
	}

	return &Parser{rt: rt}, nil
}

// jsResult is used to detect JS-side errors returned in the JSON response.
type jsResult struct {
	Error string `json:"error,omitempty"`
	Result
}

// Parse extracts main content from a raw HTML string.
//
// The url parameter is used for resolving relative links and matching
// site-specific extractors. Pass an empty string if unknown.
func (p *Parser) Parse(html, url string, opts *Options) (*Result, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if opts == nil {
		opts = &Options{}
	}

	optsJSON, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("marshal options: %w", err)
	}

	ctx := p.rt.Context()

	htmlJSON, _ := json.Marshal(html)
	urlJSON, _ := json.Marshal(url)
	optsJSONStr, _ := json.Marshal(string(optsJSON))

	script := fmt.Sprintf(
		`defuddleParse(%s, %s, %s);`,
		string(htmlJSON), string(urlJSON), string(optsJSONStr),
	)

	val, err := ctx.Eval("parse.js", qjs.Code(script))
	if err != nil {
		return nil, fmt.Errorf("eval defuddleParse: %w", err)
	}
	defer val.Free()

	var jr jsResult
	if err := json.Unmarshal([]byte(val.String()), &jr); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	if jr.Error != "" {
		return nil, fmt.Errorf("defuddle: %s", jr.Error)
	}

	result := &jr.Result

	if opts.Markdown && result.Content != "" {
		md, err := htmltomarkdown.ConvertString(result.Content)
		if err != nil {
			return nil, fmt.Errorf("html-to-markdown: %w", err)
		}
		result.Markdown = strings.TrimSpace(md)
	}

	return result, nil
}

// Close releases the underlying QuickJS runtime. Always defer this after NewParser.
func (p *Parser) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rt != nil {
		p.rt.Close()
		p.rt = nil
	}
}
