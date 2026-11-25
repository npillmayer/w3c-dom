# Repository Guidelines

## Project Structure & Module Organization
- Root package `dom` lives in `dom.go`, `actions.go`, and `doc.go`.
- Tests are in `dom_test.go` (standard Go `*_test.go`).
- `w3cdom/` defines W3C DOM interfaces.
- `style/` contains the CSS styling engine and CSSOM helpers (see `style/README.md`).
- `style/textstyle` owns checked source font/text snapshots and HTML lang/dir,
  bdi and emphasis semantics. Keep its values free of typography/font services;
  adapters explicitly convert DOM DU. See `style/textstyle/README.md` for its
  supported profile and detached-value/source-context contract.
  `ValidateDeclaration` and `ParseKeyword` share that grammar with authored CSS
  audits; keep relative declaration validation separate from source-context
  computation and leave narrower content profiles to consumers.
- `style/css.GetBorderWithFontContext` resolves em/rem border widths from
  source-font snapshots. An inherited width uses its ancestor's computed width,
  including effective zero for none/hidden; never substitute the child's font.
  Keep `GetBorder` absolute-only and preserve existing color/cascade semantics.
- `styledtree/` holds styled tree node types.
- `domdbg/` provides DOM debugging helpers (GraphViz/DOT and SVG output); sample assets include `dom-example.svg` and `domdbg/test.dot`.

DOM-SHIP-01: break-before uses the shared ParseBoxKeyword/GetBoxKeyword path,
with auto default and non-inherited metadata. Preserve all Level 3 keywords
including page-side distinctions; narrower page profiles belong to consumers.

## Build, Test, and Development Commands
- `go test ./...` — build and run all tests in the repository.
- `go test ./style/...` — run styling-specific tests only (if added).
- `go vet ./...` — static analysis; useful before PRs.
The module declares Go 1.25.6 and local replacements for sibling tree and schuko.

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
