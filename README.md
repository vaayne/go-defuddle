# go-defuddle

Go port of [Defuddle](https://github.com/kepano/defuddle) — extract main content from web pages as clean HTML or Markdown.

Runs the real Defuddle JavaScript library inside a sandboxed [QuickJS](https://github.com/fastschema/qjs) (WebAssembly) runtime. Zero CGO. Pure Go. Single binary.

## Install

### As a library

```bash
go get github.com/vaayne/go-defuddle
```

### As a CLI

```bash
go install github.com/vaayne/go-defuddle/cmd/defuddle@latest
```

## CLI usage

```bash
# Extract as markdown
defuddle -m https://example.com/article

# Output as JSON with metadata
defuddle -j https://example.com/article

# Extract a specific property
defuddle -p title https://example.com/article

# Parse a local HTML file
defuddle -m page.html

# Save to file
defuddle -m -o output.md https://example.com/article
```

### Flags

```
-m, -markdown     Convert content to markdown format
-j, -json         Output as JSON with metadata and content
-p, -property     Extract a specific property (title, author, domain, etc.)
-o, -output       Output file path (default: stdout)
    -debug        Enable debug mode
-v, -version      Print version
```

## Library usage

```go
package main

import (
	"fmt"
	"log"

	defuddle "github.com/vaayne/go-defuddle"
)

func main() {
	parser, err := defuddle.NewParser()
	if err != nil {
		log.Fatal(err)
	}
	defer parser.Close()

	result, err := parser.Parse(
		`<html>
		<head><title>My Article</title></head>
		<body>
			<article>
				<h1>My Article</h1>
				<p>This is the main content.</p>
			</article>
			<footer>Copyright 2025</footer>
		</body>
		</html>`,
		"https://example.com/my-article",
		&defuddle.Options{Markdown: true},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Title:", result.Title)
	fmt.Println(result.Markdown)
}
```

## API

### `NewParser() (*Parser, error)`

Creates a parser instance. Loads the QuickJS WASM runtime and evaluates the JS bundle (~450ms cold start). Reuse across calls.

### `Parser.Parse(html, url string, opts *Options) (*Result, error)`

Extracts main content from raw HTML.

- `html` — full HTML source
- `url` — page URL for resolving relative links and site-specific extractors
- `opts` — parsing options (pass `nil` for defaults)

### `Parser.Close()`

Releases the QuickJS runtime.

### Types

```go
type Result struct {
	Content       string          // Clean HTML
	Title         string          // Page title
	Description   string          // Meta description
	Domain        string          // Hostname
	Favicon       string          // Favicon URL
	Image         string          // Lead image URL
	Language      string          // Content language
	Published     string          // Publish date
	Author        string          // Author name
	Site          string          // Site name
	WordCount     int             // Word count
	ParseTime     int             // JS parse time (ms)
	MetaTags      []MetaTag       // Meta tags from <head>
	SchemaOrgData json.RawMessage // JSON-LD schema.org data
	Markdown      string          // Markdown (when Options.Markdown is true)
}

type Options struct {
	Markdown               bool  // Convert to Markdown (Go-side)
	RemoveSmallImages      *bool // Toggle small image removal
	RemoveHiddenElements   *bool // Toggle hidden element removal
	RemoveLowScoring       *bool // Toggle low-scoring block removal
	RemoveExactSelectors   *bool // Toggle exact CSS selector removal
	RemovePartialSelectors *bool // Toggle partial class/id removal
	RemoveContentPatterns  *bool // Toggle content-pattern removal
	Standardize            *bool // Toggle HTML normalization
	Debug                  bool  // Enable debug output
}
```

### Concurrency

A `Parser` is **not safe for concurrent use**. For concurrent workloads, create one per goroutine:

```go
pool := make(chan *defuddle.Parser, numWorkers)
for range numWorkers {
	p, _ := defuddle.NewParser()
	pool <- p
}

// Per goroutine:
p := <-pool
defer func() { pool <- p }()
result, _ := p.Parse(html, url, nil)
```

## How it works

```
┌──────────────┐       ┌──────────────────────────┐       ┌────────────────────┐
│   Go app     │──────▶│   QuickJS (Wazero WASM)  │──────▶│  html-to-markdown  │
│  .Parse()    │ HTML  │   defuddle + linkedom     │ JSON  │  HTML → Markdown   │
└──────────────┘       └──────────────────────────┘       └────────────────────┘
```

1. **Content extraction** runs in JavaScript. Defuddle and [linkedom](https://github.com/WebReflection/linkedom) are bundled into a single ~430KB JS file executed in QuickJS via [Wazero](https://wazero.io/) (WebAssembly). No Node.js, no browser, no CGO.
2. **Markdown conversion** runs in Go via [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown), which uses [goldmark](https://github.com/yuin/goldmark) internally.

### Performance

| Metric | Time |
|--------|------|
| Init (cold start) | ~450ms |
| Parse + Markdown | ~95ms |

Init is one-time per `Parser` instance.

## Syncing with upstream Defuddle

[Defuddle](https://github.com/kepano/defuddle) is included as a git submodule. The JS bundle is a **custom webpack build** — not taken from Defuddle's `dist/` — because Defuddle's shipped bundles expect either a browser DOM or Node.js `require()`, neither of which exist in QuickJS.

Our custom bundle (`internal/js/bundle-entry.js`):
- Inlines linkedom directly (no runtime `require()`)
- Imports `Defuddle` from source (`defuddle/src/defuddle.ts`)
- Patches the DOM (`styleSheets`, `getComputedStyle`)
- Skips Turndown (Go handles Markdown)
- Uses `math.core.ts` (no temml/mathml-to-latex, saves ~450KB)

### To sync

```bash
# Update the submodule
cd defuddle
git pull origin main
cd ..

# Install JS deps and rebuild bundle
npm install
npx webpack

# Verify
go test ./...
go run ./cmd/defuddle/ -m https://stephango.com/saw
```

### What can break

| Upstream change | Fix |
|---|---|
| New browser/Node API used | Add polyfill to `internal/js/polyfills.js` |
| `Defuddle` constructor or `parse()` signature changes | Update `internal/js/bundle-entry.js` |
| `parse()` return type changes | Update `Result` struct in `defuddle.go` |
| New npm dep with native bindings | Check for pure-JS alternative |
| `math.core.ts` path changes | Update webpack alias in `webpack.config.js` |

## QuickJS polyfills

QuickJS is ES2023 compliant but has no Web/Node APIs. `internal/js/polyfills.js` provides:

| Polyfill | Reason |
|----------|--------|
| `self` | UMD bundle expects `self` on `globalThis` |
| `Buffer.from()` | htmlparser2 entity decoder uses Buffer for base64 |
| `URL` | Defuddle uses `new URL()` for domain extraction, link resolution |
| `atob()` | Base64 fallback for htmlparser2 |
| `performance.now()` | Defuddle profiling; shimmed to `Date.now()` |

## Project structure

```
go-defuddle/
├── defuddle.go              # Go library (Parser, Result, Options)
├── defuddle/                # git submodule → github.com/kepano/defuddle
├── cmd/defuddle/main.go     # CLI
├── internal/js/
│   ├── bundle-entry.js      # Webpack entry (wires linkedom + defuddle)
│   ├── polyfills.js         # QuickJS polyfills (Buffer, URL, atob, etc.)
│   └── defuddle-bundle.js   # Built bundle (~430KB, embedded via go:embed)
├── webpack.config.js        # Webpack config
├── tsconfig.json            # TypeScript config for webpack
├── package.json             # npm deps (linkedom, webpack, ts-loader)
└── go.mod
```

## Dependencies

### Go

| Package | Purpose |
|---------|---------|
| [fastschema/qjs](https://github.com/fastschema/qjs) | QuickJS via Wazero (WASM, no CGO) |
| [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) | HTML → Markdown (uses goldmark) |

### JavaScript (bundled into defuddle-bundle.js)

| Package | Purpose |
|---------|---------|
| [defuddle](https://github.com/kepano/defuddle) | Content extraction pipeline |
| [linkedom](https://github.com/WebReflection/linkedom) | DOM implementation |
| [htmlparser2](https://github.com/fb55/htmlparser2) | HTML parser |
| [cssom](https://github.com/NV/CSSOM) | CSS parsing |

## Limitations

- **No `getComputedStyle`**: linkedom doesn't compute CSS. Hidden-element removal uses inline styles and class heuristics.
- **No canvas**: Image dimensions use HTML attributes only.
- **URL polyfill is minimal**: Covers common cases. Edge cases with IPv6 or exotic schemes may not parse.
- **Single-threaded per Parser**: Create multiple instances for concurrency.
- **~450ms cold start**: First `NewParser()` loads WASM + JS. Subsequent `Parse` calls are ~95ms.

## Credits

- [Defuddle](https://github.com/kepano/defuddle) by Steph Ango — the content extraction engine
- [QJS](https://github.com/fastschema/qjs) — CGO-free QuickJS for Go via Wazero
- [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) — HTML to Markdown in Go
- [linkedom](https://github.com/WebReflection/linkedom) — lightweight DOM

## License

MIT
