# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.2] - 2026-04-12

### Changed

- Updated Defuddle submodule to [`66049af`](https://github.com/kepano/defuddle/commit/66049af2aab6bd16bc38d24176f3dd19a86a086d) (0.16.0).
- Rebuilt JS bundle with latest upstream changes.
## [0.1.1] - 2026-04-12

### Changed

- Updated Defuddle submodule to [`3cb89a8`](https://github.com/kepano/defuddle/commit/3cb89a8ca8dfdc29a109ec33ae0efd089970f180) (v0.13.0-162).
- Rebuilt JS bundle with latest upstream changes including:
  - YouTube transcript extraction improvements (CJK ranges, multi-speaker support, fallback behaviors)
  - Configurable fetch option support
  - HTML parsing enhancements

## [0.1.0] - 2026-04-03

### Added

- Go library (`defuddle.NewParser`, `Parser.Parse`, `Parser.Close`) for extracting main content from web pages as clean HTML or Markdown.
- CLI tool (`cmd/defuddle`) with flags for markdown output (`-m`), JSON output (`-j`), property extraction (`-p`), and file output (`-o`).
- Sandboxed QuickJS (Wazero WASM) runtime — no CGO, no Node.js, no browser.
- Custom webpack bundle inlining linkedom and Defuddle source with QuickJS polyfills (`Buffer`, `URL`, `atob`, `performance.now`).
- Go-native Markdown conversion via [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown).
- Fixture tests against the upstream Defuddle test suite.
- CI workflow (bundle-check + lint + test) and release workflow (cross-compile for linux/darwin/windows amd64/arm64).
- mise tasks for bundle, sync, build, test, lint, and CI.

### Dependencies

- Defuddle submodule at [`b19bc0e`](https://github.com/kepano/defuddle/commit/b19bc0e) (v0.15.0+)
- [fastschema/qjs](https://github.com/fastschema/qjs) — QuickJS via Wazero
- [html-to-markdown v2](https://github.com/JohannesKaufmann/html-to-markdown) — HTML → Markdown

[0.1.1]: https://github.com/vaayne/go-defuddle/releases/tag/v0.1.1
[0.1.0]: https://github.com/vaayne/go-defuddle/releases/tag/v0.1.0
[0.1.2]: https://github.com/vaayne/go-defuddle/releases/tag/v0.1.2
