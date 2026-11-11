
# w3c-dom

Experimental HTML DOM and CSS styling utilities for a typesetting pipeline.

See the [module inventory](doc/README.md) for architecture, current quality,
confirmed accessor failures, and repair priorities (reviewed 2026-11-10).

## Historical TODO list

The entries below are retained for reference. Their line numbers and priorities
are not maintained; see the [TODO reconciliation](doc/todo.md) for current status.

  - `dom.go:302` — include pseudo-elements in computed styles.
  - `dimen/dimen.go:66` — see methods in image.Point.
  - `style/defaults.go:66` — load user agent defaults.
  - `style/defaults.go:182` — set background-color to transparent (CSS default).
  - `style/defaults.go:221` — position TODO.
  - `style/cssom/cssom.go:495` — consider not syncing after node creation; pipeline overlap.
  - `style/cssom/cssom.go:664` — debug-print helper for node path.
  - `style/convert.go:5` — use standard palette.
  - `style/css/option.go.refact:174` — TODO in refactored options.
  - `style/css/option.go.refact:486` — TODO bidi variants.
  - `style/css/display.go:126` — TODO (display handling).
