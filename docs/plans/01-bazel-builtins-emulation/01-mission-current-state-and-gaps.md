# 01 — Mission, current state, and gaps

## Mission

`starlark-go-bazel` is **the Go library for safely evaluating Bazel
Starlark for static analysis purposes**. The header on the existing
README already commits to this framing:

> A Go implementation of Bazel's Starlark dialect. Execute `.bzl` and
> `BUILD` files with full Bazel builtins, compile to WASM for
> browser-based tools, and evaluate rules without requiring actual
> build execution.

This plan extends the existing mission to **also** cover:

- `MODULE.bazel` and `.bzl` files that define repository rules,
  module extensions, and tag classes (the bzlmod era).
- `repository_ctx` and `module_ctx` synthesis so consumers can drive
  these impls without an actual repository fetch.
- A version-aware builtin surface so consumers can pin "act as Bazel
  7.4 would" or "act as Bazel 9.1 would."
- A lenient mode (`Permissive`, taint tracking) for safely
  evaluating arbitrary `.bzl` files even when external loads can't
  be resolved.

### Concrete adopters in scope

Three concrete consumers determine the API surface:

- **canopy** ingest + airgap external-surface (this plan's parent
  use case) — drives M1–M8.
- **scip-bazel** indexer — needs symbol resolution + cross-module
  load chain semantics. Doesn't drive timing but the API has to
  serve it.
- **canopy compat-analyzer** (plan 05) — evaluates a user's
  `MODULE.bazel` against drift; needs `module_extension` driver +
  Version awareness.

Speculative adopters (no commitment, but the API should not preclude
them): generic Bazel linters, supply-chain scanners (SBOM emit),
WASM-compiled in-browser playgrounds (current README mentions this
explicitly).

### What this library is NOT

Explicit non-goals so consumers don't ask:

- **Not a build executor.** No real action execution, no actual file
  downloads, no compilation. `ctx.actions.run` is a no-op stub.
- **Not a faithful Bazel reimplementation.** We emulate the *Starlark
  surface* Bazel exposes — the C++ runtime semantics (configuration
  transitions, dependency resolution, action graph) are out of scope.
- **Not a hostile-input sandbox.** Today's `Mode=Lenient` evaluates
  with bounded steps + memory but stays in-process. A truly hostile
  module could write a step bomb or memory blowup; production wraps
  with a worker-pool + cgroup. Document, don't pretend.
- **Not Bazel-6-targeting.** Bazel 6 reached end-of-support in
  Dec 2025 per the LTS matrix. Minimum supported Version is V7.
- **Not WORKSPACE-era authoritative.** WORKSPACE is legacy in
  Bazel 7+ and the engineering investment goes to bzlmod. We accept
  WORKSPACE-shape `.bzl` files in `ModeLenient` best-effort but
  don't promise correctness; bzlmod (`MODULE.bazel`, `use_extension`,
  `repository_rule` inside extension impls) is the supported path.

## Current state — what's already in place

Walking the existing tree at `/Volumes/T9/dev/ws/starlark-go-bazel/`:

### `builtins/`

| Symbol | Status | Notes |
|---|---|---|
| `rule()` | Implemented | Returns `*types.RuleClass`; consumed by assay/interp's Hydrate |
| `provider()` | Implemented | Returns provider class |
| `aspect()` | Implemented | Returns aspect class |
| `select()` | Implemented | Returns SelectorValue |
| `struct()` | Implemented | Standard Starlark struct |
| `depset()` | Implemented | Real depset semantics |
| `Label()` | Implemented (via `types.LabelBuiltin`) | Parses label strings; carries `.workspace_root`, `.package_name`, `.name` |
| `attr.*` | Implemented | string/int/bool/label/label_list/string_list/etc. via `attr/module.go` |
| `repository_rule()` | **MISSING** — gap the spike fills | Pinned by `TestHydrate_RepositoryRule_StillUntouchedUntilUpstreamSupport` |
| `module_extension()` | **MISSING** — gap the spike fills | No upstream pinning yet |
| `tag_class()` | **MISSING** — companion to module_extension | |

### `attr/`

| Item | Status | Notes |
|---|---|---|
| `descriptor.go` | Partial | Holds `AttrDescriptor` but per-attr `Type` / `Default` / `Doc` / `Mandatory` are stubbed; the existing `types/rule_class.go:712` comment says "we create a basic descriptor" |
| `module.go` | Implemented | Exposes `attr.*` as a Starlark module |

**Known gap**: attr descriptor stubbing is documented in
`/Volumes/T9/dev/ws/assay/interp/LIMITATIONS.md` under "Per-attr Type
/ Default / Doc / Mandatory." Consumers (assay/interp) work around by
only extracting attr names. Production-fix needed but **not in scope
for this plan** — separate effort.

### `ctx/`

| File | Status | Notes |
|---|---|---|
| `ctx.go` | Implemented | Analysis-time `ctx` — `ctx.attr`, `ctx.label`, `ctx.actions`, `ctx.files`, `ctx.executable`, `ctx.runfiles`, etc. |
| `actions.go` | Implemented | `ctx.actions.run`, `ctx.actions.write`, etc. (stubbed for analysis) |
| `attr_proxy.go` | Implemented | The `ctx.attr.<name>` access path |
| `file.go` | Implemented | `ctx.file` / `ctx.files` |
| **`repository_ctx.go`** | **MISSING** — spike adds | Synthetic ctx for repo_rule impls |
| **`module_ctx.go`** | **MISSING** — spike adds | Synthetic ctx for module_extension impls |

### `eval/`

| File | Status | Notes |
|---|---|---|
| `evaluator.go` | Implemented | Accepts `PredeclaredBzl` / `PredeclaredBuild` override (line 40-41), merges with default builtins |
| `bzl_file.go` | Implemented | `.bzl` evaluation path |
| `build_file.go` | Implemented | BUILD evaluation path |

**Known gap**: `bzl.Options` does NOT expose the `PredeclaredBzl` /
`PredeclaredBuild` overrides. This is the workaround the spike uses
(bypassing `bzl.Interpreter` to drive `eval.Evaluator` directly).
**Fixing this is M1 of the milestone plan** — minimal change,
unblocks every consumer.

### `bzl/`

| File | Status | Notes |
|---|---|---|
| `bzl.go` | Implemented | `Interpreter` with `Eval` / `EvalFile` |
| `options.go` | Implemented | `WorkspaceRoot`, `FileSystem`, `ExternalRepos`, `PrintHandler`, `LenientLoad` |
| `lenient_load_test.go` | Implemented | Already has a "lenient" concept for missing external loads — this is the seed of our `Mode=Lenient` |

### `loader/`

| File | Status | Notes |
|---|---|---|
| FileSystem loader | Implemented | `LenientLoad` option already drops unresolvable external loads quietly |
| `BzlFileLoader` | Implemented | Handles cross-module `load("@dep//pkg:foo.bzl")` |

### `native/`

| Symbol | Status |
|---|---|
| `native.glob` | Implemented (returns empty list in analysis context) |
| `native.existing_rule` | Implemented |
| `native.package_name` | Implemented |
| `native.package_relative_label` | Implemented |
| **`native.module_name` / `native.module_version`** | **MISSING** — bzlmod-era convenience for repo-rule impls |

### `types/`

| Type | Status | Notes |
|---|---|---|
| `Label` | Implemented | Full Label semantics |
| `RuleClass` | Implemented but attr descriptors stubbed | The attr stubbing limitation lives here |
| **`RepositoryRuleClass`** | **MISSING** — spike adds | |
| **`ModuleExtensionClass`** | **MISSING** — spike adds | |

### `providers/`

| Item | Status |
|---|---|
| `DefaultInfo` | Implemented |
| `OutputGroupInfo` | Implemented |
| `Runfiles` | Implemented |

### `analysis/`

Present in tree but I haven't audited deeply — likely the
analysis-time evaluation glue. Not relevant to this plan's scope
(repo_rule + module_extension are NOT analysis-time; they're at
fetch / module-extension time).

## Gaps the spike fills

Cross-referencing what's missing above:

1. **`repository_rule()` builtin** — captures `(implementation, attrs,
   environ, local, configure, remotable, ...)`. Spike has a minimal
   version (only impl + attrs); production wants all kwargs.
2. **`module_extension()` builtin** — captures `(implementation,
   tag_classes, ...)`. Spike has minimal; production wants
   `os_dependent`, `arch_dependent`, `reproducible` (Bazel 7+).
3. **`tag_class()` builtin** — companion. Spike treats as Permissive
   stub; production wants real capture so consumers can inspect the
   schema.
4. **`repository_ctx`** — full surface: `.attr`, `.os`,
   `.download`, `.download_and_extract`, `.execute`, `.read`,
   `.file`, `.template`, `.symlink`, `.delete`, `.extract`,
   `.report_progress`, `.which`, `.path`, `.workspace_root`,
   `.name`, `.environ` (Bazel 7+), `.repo_metadata` (Bazel 8+),
   `.original_name`.
5. **`module_ctx`** — `.modules` (list of bazel_module), `.path`,
   `.download` / `.download_and_extract`, `.execute`, `.read`,
   `.os`, `.is_dev_dependency`, `.extension_metadata`,
   `.root_module_has_non_dev_dependency`.
6. **bazel_module** — `.name`, `.version`, `.is_root`, `.tags`,
   `.is_dev_dependency`.
7. **`Permissive` value type** — universal stub that satisfies
   `Callable + HasAttrs + Mapping + HasBinary + Comparable`.
8. **`permissiveLoader`** — load-time helper that fills unresolvable
   symbols with Permissive (next-level lenient load, beyond the
   existing `LenientLoad` which just drops symbols).
9. **Taint tracking** — `PermissiveMarker` constant, per-fork
   `tainted` flag on `repository_ctx`, marker-detecting
   `flattenURLs` helper.
10. **Capture sinks** — the URL-extraction pipeline needs a way to
    collect side effects (URLs, instantiations) from synthetic ctx
    invocations.
11. **`Version` enum** — runtime semantic-version targeting.
12. **`Mode` flag** — `ModeStrict` / `ModeLenient` / `ModeAnalysis`.
13. **`bazel_features` stub** — pre-populated feature-flag struct
    per `Version` so real rulesets that gate on
    `bazel_features.module_extension_has_os_arch_dependent` get
    correct true/false answers instead of Permissive.

## What's wrong, under-baked, or needs review

### In existing starlark-go-bazel

1. **Attr descriptor stubbing.** `types/rule_class.go:712` admits
   "we create a basic descriptor." Per-attr `Type` / `Default` /
   `Doc` / `Mandatory` are dropped. Documented in
   `assay/interp/LIMITATIONS.md`. Fixing is a separate plan but
   should be referenced from M8 onward.
2. **bzl.Options doesn't expose PredeclaredBzl.** Forces consumers
   into eval.Evaluator. M1 fixes.
3. **`LenientLoad` semantics are coarse.** Drops external loads
   entirely; doesn't fill in the symbol bindings. Result: any code
   that *uses* a loaded external symbol still fails. The spike's
   `permissiveLoader` is strictly more capable — should subsume
   `LenientLoad` (rename `LenientLoad` to `Mode=Lenient`).
4. **No `Version` awareness.** Library currently emulates "some
   version of Bazel" without naming it. Consumers can't pin.
5. **No `Mode` flag.** Strict vs. lenient evaluation is unfortunately
   conflated with `LenientLoad`.
6. **No public taint primitives.** Consumers can't ask "did this
   evaluation touch any opaque op?"
7. **`native.module_name` / `native.module_version` missing.** Some
   repo_rule impls call these to identify the calling module.
8. **No `BazelFeatures` stub.** Real rulesets do `if
   bazel_features.X.Y` to gate behavior; today this fails or stubs
   weirdly.
9. **WASM build path.** Currently compiles to WASM; need to confirm
   the spike additions stay WASM-compatible (no syscalls, no fs
   beyond loader). Spec'd in 05-spike-promotion-and-testing.md
   under "WASM compatibility check."

### Inherited from spike (carry over)

1. **`str(perm)` precision loss** — partially closed via marker, but
   the marker is a substring; a literal URL containing
   `"<permissive>"` would be false-positive tainted. Acceptable; flag.
2. **Both-branch exploration of `if perm == X`** — currently single
   branch (else); production wishlist concolic.
3. **Permissive as dict key** — `Hash()` errors. M-9 could add
   hashable Permissive returning consistent sentinel hash; for now,
   document.
4. **`ctx.execute` result struct** — spike returns
   `{stdout, stderr, return_code}` but real `exec_result` has more
   surface; document gap.
5. **`Label.workspace_name`** — `labelBuiltin` in spike returns
   `args[0]` (a string); real `Label()` returns a Label object with
   methods. starlark-go-bazel's existing `types.LabelBuiltin` is the
   right answer; promotion needs to remove the spike's stub.
6. **Performance not yet measured at scale.** Spike runs in 140ms
   for 23 tests; production needs to bench on the assay corpus
   (~2000 file rules_python module) to confirm <2s per-module cache
   miss target.

### Risks specific to this plan

1. **API churn during promotion.** Once we publish `Mode`, `Version`,
   `PredeclaredBzl` on `bzl.Options`, third parties may adopt before
   the API stabilizes. Mitigation: tag a v0.x throughout this plan,
   only promise stability at v1.0.
2. **bazel_features content vendoring.** Either we vendor the
   `bazel-features-bzl` data file (carries license, version-tied) or
   we curate a per-version flag table by hand. Decision deferred to
   M6.
3. **starlark-go-bazel maintainer load.** Alberto is the sole
   maintainer today. This plan ~doubles the surface area. Need to
   establish contribution / RFC process if community contributors
   appear post-publication.
