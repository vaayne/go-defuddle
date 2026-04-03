# AGENTS.md

Go wrapper for [Defuddle](https://github.com/kepano/defuddle). Runs the JS library in sandboxed QuickJS (Wazero WASM). No CGO.

## Architecture

```
defuddle.go          — Go library (Parser, Result, Options)
cmd/defuddle/        — CLI
internal/js/         — JS bundle (polyfills.js, bundle-entry.js, defuddle-bundle.js)
defuddle/            — upstream submodule (github.com/kepano/defuddle)
```

Content extraction runs in JS (QuickJS). Markdown conversion runs in Go (html-to-markdown).

## Commands

```bash
mise run sync       # update submodule + rebuild bundle + test
mise run bundle     # rebuild JS bundle only
mise run test       # go test ./...
mise run lint       # go vet ./...
mise run ci         # bundle-check + lint + test
```

## Rules

- Run `mise run test` before committing Go changes.
- Run `mise run bundle` after touching anything in `defuddle/`, `internal/js/`, or `webpack.config.js`.
- Commit the rebuilt `internal/js/defuddle-bundle.js` — CI verifies it matches a fresh build.
- A `Parser` is not concurrency-safe. Create one per goroutine.
- Keep polyfills in `internal/js/polyfills.js` minimal — only what QuickJS lacks.
- Use conventional commits with emoji (`✨ feat:`, `🐛 fix:`, `♻️ refactor:`, `📝 docs:`, `⬆️ chore:`).

## Docs

- Keep `README.md` up to date when adding/changing API, CLI flags, types, or project structure.
- Update `CHANGELOG.md` for every user-facing change following [Keep a Changelog](https://keepachangelog.com/) format.
- `docs/` contains long-form documentation:
  - `docs/design.md` — architecture, data flow, trade-offs, and syncing strategy.
- Update relevant docs when making structural changes.
