# Module inventory

Reviewed 2026-11-10 at commit `4b7dda6`. This inventory describes the checked-out
implementation, not a claim of W3C conformance. A follow-up on 2026-11-10 fixes
the confirmed accessor failures A1–A5 and adds regression tests. A further fix
makes empty-style extraction and missing head/body containers safe. `Children()`
now returns only element nodes, with correct length and no typed nil padding.
Confirmed margin and border-radius expansion bugs are also fixed and tested.

- [Architecture and scope](architecture.md): purpose, packages, data flow, and boundaries.
- [Quality and accessor safety](quality.md): confirmed failures, verification, and repair priorities.
- [TODO reconciliation](todo.md): status of every item in the root README.

The module is a usable experimental foundation for attaching CSS declarations to
HTML trees for a typesetting pipeline. Its strongest tested area is rule selection.
It is not yet a dependable general-purpose computed-style API: ordinary inherited
property panic paths A1–A5 are now fixed, but property semantics remain incomplete
and broader CSS conversion support remains incomplete. Confirmed dimension and
position variant bugs and percentage payload parsing are now fixed. Remaining correctness
work should precede feature expansion or concurrency optimization.

The review covered the Go source, tests, inactive refactoring file, module manifest,
and existing documentation. The reported client findings were treated as leads;
their original reports were not supplied or located in the nearby Markdown searched.
The confirmed failures below were independently reproduced against this checkout.

Inline declaration parsing, duplicate precedence, importance, and value-case
preservation were corrected on 2026-11-11, with focused regression tests.

Stylesheet-adapter duplicate selection now also preserves importance and the
winning declarations’ source order (2026-11-11).

Custom shorthand splitter registration is now retained and exercised during
styling, with regression coverage (2026-11-11).

Subtree stylesheet registration now applies to descendants, with nested-scope
ordering and inline isolation tested (2026-11-11).

Property inheritance and initial/UA default resolution now share one metadata
registry and resolver, with focused coverage (2026-11-11).

The `word-wrap` alias now shares canonical `overflow-wrap` storage and metadata.
Inline and stylesheet declarations compete as one property using declaration order
and importance; both lookup names work. Property enumeration uses `overflow-wrap`.
Focused alias regressions were added on 2026-11-11.

Dimension parsing now supports signed decimal values and checked scaling with
half-away-from-zero rounding. RelativeValue exposes fractional font/viewport
magnitudes. Overflow and fractional regression tests were added (2026-11-11).

Color conversion now supports the named palette and common sRGB formats, with
explicit ParseColor errors and alpha-preserving serialization (2026-11-11).

The downstream missing-font-metadata finding is resolved for the five requested
properties; font computation and layout remain downstream responsibilities.

Recursive, complete-content stylesheet extraction and explicit extraction/DOM
construction errors now address the downstream extraction finding (2026-11-11).

CSS absolute units are now consistent throughout constants and both parsers:
96 px/in and 72 pt/in, retaining fixed-point DU storage (2026-11-11).

Supported inline elements now receive inline display defaults; tag classification
and explicit display override tests cover the downstream finding (2026-11-11).

ParseDisplay now recognizes all table-part values returned by the HTML defaults,
with distinct typed flags and regression coverage (2026-11-11).
