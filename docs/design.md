# Design Document

## Overview

go-defuddle is a Go library and CLI for extracting main content from web pages. Rather than re-implementing the [Defuddle](https://github.com/kepano/defuddle) extraction algorithm in Go, it runs the real JavaScript library inside a sandboxed QuickJS runtime via WebAssembly.

## Goals

- **Fidelity** — produce identical extraction results to upstream Defuddle.
- **Zero CGO** — pure Go binary, easy to cross-compile and deploy.
- **Single binary** — no external runtime, no Node.js, no browser.
- **Low surface area** — thin Go wrapper; upstream JS does the heavy lifting.

## Non-goals

- Re-implementing Defuddle's extraction logic in Go.
- Supporting browser or Node.js as a runtime.
- Full Web API compatibility in QuickJS.

## Architecture

```
┌──────────────┐       ┌──────────────────────────┐       ┌────────────────────┐
│   Go caller  │──────▶│   QuickJS (Wazero WASM)  │──────▶│  html-to-markdown  │
│  .Parse()    │ HTML  │   defuddle + linkedom     │ JSON  │  HTML → Markdown   │
└──────────────┘       └──────────────────────────┘       └────────────────────┘
```

### Layers

1. **Go API** (`defuddle.go`) — `Parser` manages a QuickJS runtime. `Parse()` sends HTML in, gets structured JSON out. The JS bundle is embedded via `go:embed`.

2. **JS sandbox** (QuickJS via [Wazero](https://wazero.io/)) — a single-threaded ES2023 engine compiled to WebAssembly. No file system, no network, no CGO.

3. **JS bundle** (`internal/js/defuddle-bundle.js`) — a custom webpack build that inlines Defuddle source + [linkedom](https://github.com/WebReflection/linkedom) (DOM implementation) into one self-contained file.

4. **Markdown conversion** (Go-side) — [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) converts extracted HTML to Markdown natively in Go, bypassing Defuddle's built-in Turndown dependency.

### Why this split

| Concern | Where | Why |
|---------|-------|-----|
| Content extraction | JS (QuickJS) | Reuse upstream logic exactly, stay in sync |
| DOM parsing | JS (linkedom) | Defuddle expects a DOM; linkedom is lightweight and has no native deps |
| Markdown conversion | Go (html-to-markdown) | Avoids bundling Turndown; Go-native is faster and more controllable |
| Polyfills | JS (`polyfills.js`) | QuickJS lacks Web APIs (`URL`, `Buffer`, `atob`, `performance.now`) |

## JS Bundle

The bundle is **not** taken from Defuddle's `dist/` because upstream builds expect either a browser DOM or Node.js `require()`. Our custom webpack build:

- Imports `Defuddle` directly from TypeScript source.
- Inlines linkedom (no runtime `require()`).
- Patches missing DOM APIs (`styleSheets`, `getComputedStyle`).
- Excludes Turndown, temml, and mathml-to-latex (saves ~450KB).
- Uses `math.core.ts` alias to skip heavy math rendering deps.

Entry point: `internal/js/bundle-entry.js` exposes a single global function `defuddleParse(html, url, optionsJson)` that returns a JSON string.

## Polyfills

QuickJS is ES2023 compliant but has no Web/Node APIs. `internal/js/polyfills.js` provides minimal shims:

| Polyfill | Consumer |
|----------|----------|
| `self` | UMD bundles expect `self` on `globalThis` |
| `Buffer.from()` | htmlparser2 entity decoder (base64) |
| `URL` | Defuddle link resolution and domain extraction |
| `atob()` | Base64 fallback for htmlparser2 |
| `performance.now()` | Defuddle profiling (shimmed to `Date.now()`) |

Polyfills are loaded before the bundle and kept intentionally minimal — only add what breaks without them.

## Data flow

```
1. Go: Parser.Parse(html, url, opts)
2. Go: JSON-encode html, url, opts → JS eval string
3. JS:  defuddleParse() → linkedom parses HTML → Defuddle extracts content
4. JS:  return JSON.stringify(result)
5. Go: unmarshal JSON → Result struct
6. Go: if opts.Markdown, convert result.Content via html-to-markdown
7. Go: return *Result
```

## Concurrency

A `Parser` holds a single-threaded QuickJS runtime and is **not safe for concurrent use**. The `mu sync.Mutex` prevents data races but serializes calls. For parallel workloads, create one `Parser` per goroutine or use a pool.

## Performance

| Metric | Time |
|--------|------|
| `NewParser()` cold start | ~450ms (load WASM + eval JS bundle) |
| `Parse()` per page | ~95ms |

The cold start is dominated by Wazero compiling the QuickJS WASM module and evaluating ~430KB of JavaScript. Subsequent `Parse()` calls reuse the warm runtime.

## Syncing with upstream

Defuddle is pinned as a git submodule. To update:

```bash
mise run sync   # pulls latest, rebuilds bundle, runs tests
```

### What can break on update

| Upstream change | Required fix |
|---|---|
| New browser/Node API used | Add polyfill to `polyfills.js` |
| `Defuddle` constructor or `parse()` signature changes | Update `bundle-entry.js` |
| `parse()` return shape changes | Update `Result` struct in `defuddle.go` |
| New npm dep with native bindings | Find pure-JS alternative or polyfill |
| `math.core.ts` path moves | Update webpack alias |

## Trade-offs

**Chose JS-in-WASM over pure Go port:**
- ✅ Exact parity with upstream — no divergence risk.
- ✅ Easy to sync — `git submodule update` + rebuild.
- ❌ ~450ms cold start per Parser instance.
- ❌ Debugging JS issues inside QuickJS is harder.

**Chose Go-side Markdown over bundling Turndown:**
- ✅ Smaller bundle (~430KB vs ~880KB).
- ✅ Markdown output is customizable from Go.
- ❌ Minor formatting differences from Defuddle's native Markdown output.
