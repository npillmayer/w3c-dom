# Repository Guidelines

## Project Structure & Module Organization
- Root package `dom` lives in `dom.go`, `actions.go`, and `doc.go`.
- Tests are in `dom_test.go` (standard Go `*_test.go`).
- `w3cdom/` defines W3C DOM interfaces.
- `style/` contains the CSS styling engine and CSSOM helpers (see `style/README.md`).
- `styledtree/` holds styled tree node types.
- `domdbg/` provides DOM debugging helpers (GraphViz/DOT and SVG output); sample assets include `dom-example.svg` and `domdbg/test.dot`.

## Build, Test, and Development Commands
- `go test ./...` — build and run all tests in the repository.
- `go test ./style/...` — run styling-specific tests only (if added).
- `go vet ./...` — static analysis; useful before PRs.
Note: there is no `go.mod` in this repo; if your environment defaults to modules, you may need a GOPATH setup or a local `go.work` to run tests successfully.

## Coding Style & Naming Conventions
- Use standard Go formatting (`gofmt`); tabs for indentation.
- Exported identifiers are `PascalCase`; unexported are `camelCase`.
- Prefer short, descriptive package names (`domdbg`, `styledtree`, `w3cdom`).
- Keep comments for exported types and functions in GoDoc style.

## Testing Guidelines
- Use Go’s `testing` package and name tests `TestXxx` in `*_test.go`.
- Add focused unit tests near the package they cover (e.g., `style/...`).
- No explicit coverage threshold is defined; aim to cover new behavior.

## Commit & Pull Request Guidelines
- Git history shows only an initial commit, so no established convention exists.
- Recommended commit format: short, imperative summary (optionally scoped, e.g., `style: add property parser`).
- PRs should include a concise summary, testing notes (`go test ./...`), and call out any API-breaking changes (APIs are marked “early draft” in docs).
- Include screenshots or DOT/SVG output if you change DOM visualization or styling output.

## Project Status & Stability
- The project is an early draft; public APIs may change without notice.
- Keep changes small and well-scoped; update docs when behavior changes.
