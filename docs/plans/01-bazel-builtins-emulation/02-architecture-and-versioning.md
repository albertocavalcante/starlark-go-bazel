# 02 — Architecture and versioning

## Proposed package layout

```
starlark-go-bazel/
  bzl/
    bzl.go              ── existing; thin facade over eval.Evaluator
    options.go          ── grows: + PredeclaredBzl, + PredeclaredBuild,
                                  + Version, + Mode, + CaptureSinks
  eval/
    evaluator.go        ── existing; already accepts PredeclaredBzl
    bzl_file.go         ── existing
    build_file.go       ── existing
    fork.go             ── NEW: (os, arch) matrix forking helper for
                                 driving repo_rule impls
  builtins/
    rule.go             ── existing
    provider.go         ── existing
    aspect.go           ── existing
    select.go           ── existing
    struct.go           ── existing
    depset.go           ── existing
    builtins.go         ── existing; extend Predeclared() to accept
                                      Version and return version-pinned
                                      universe
    repository_rule.go  ── NEW (from spike)
    module_extension.go ── NEW (from spike)
    tag_class.go        ── NEW
    bazel_features.go   ── NEW: per-version feature-flag stub
  ctx/
    ctx.go              ── existing (analysis-time)
    actions.go          ── existing
    attr_proxy.go       ── existing
    file.go             ── existing
    repository_ctx.go   ── NEW (from spike)
    module_ctx.go       ── NEW (from spike)
    bazel_module.go     ── NEW: the bazel_module value type
  types/
    rule_class.go       ── existing
    label.go            ── existing
    repository_rule_class.go  ── NEW: types.RepositoryRuleClass
    module_extension_class.go ── NEW: types.ModuleExtensionClass
  attr/
    descriptor.go       ── existing; gap noted (separate fix plan)
    module.go           ── existing
  loader/
    *.go                ── existing
                              (no changes; lenient-load logic moves to
                               the new stub package which composes with
                               loader)
  native/
    glob.go             ── existing
    existing_rule.go    ── existing
    module.go           ── existing
    package_info.go     ── existing
    module_identity.go  ── NEW: native.module_name / native.module_version
  providers/
    default_info.go     ── existing
    output_group.go     ── existing
    runfiles.go         ── existing
  stub/                 ── NEW PACKAGE
    permissive.go       ── from spike
    loader.go           ── permissiveLoader: from spike
  taint/                ── NEW PACKAGE
    taint.go            ── Marker constant, CapturedURL, Tainted helpers
    fork.go             ── Per-fork tainted flag + flattenURLs helper
  version/              ── NEW PACKAGE
    version.go          ── Version enum (V7, V8, V9, VLatest)
    features.go         ── per-Version feature-flag registry
                                 (drives bazel_features.bazel_features)
    deltas.go           ── per-Version behavior delta table
                                 (which builtins exist, deprecations)
  analysis/             ── existing
  docs/
    plans/
      01-bazel-builtins-emulation/   ── this directory
```

Three NEW packages: `stub/`, `taint/`, `version/`. The rest is
additive within existing packages.

## API surface — extended `bzl.Options`

```go
type Options struct {
    // -------- existing --------
    WorkspaceRoot string
    FileSystem    loader.FileSystem
    ExternalRepos map[string]string
    PrintHandler  func(msg string)
    LenientLoad   bool                  // DEPRECATED: use Mode

    // -------- NEW (additive, zero-value backward compatible) --------

    // Version pins the Bazel LTS major whose default builtin surface
    // we emulate. Zero value = VLatest (Bazel 9 today). See
    // version/version.go for the orthogonal FeatureFlags axis.
    Version version.Version

    // FeatureFlags toggles individual experimental/incompatible
    // features. Orthogonal to Version: a consumer targeting V9 might
    // still have --experimental_repository_ctx_execute_wasm OFF.
    // Map key = the flag's command-line name (e.g.,
    // "experimental_repository_ctx_execute_wasm"). Unset key = default
    // for the selected Version.
    FeatureFlags map[string]bool

    // Mode selects strictness. Zero value = ModeStrict (preserves
    // existing default behavior).
    Mode Mode

    // PredeclaredBzl/Build adds or overrides the predeclared globals
    // for .bzl/BUILD evaluation. Merged on top of the version-pinned
    // default universe.
    PredeclaredBzl   starlark.StringDict
    PredeclaredBuild starlark.StringDict

    // CaptureSinks (Mode=Analysis only) collects side effects from
    // repository_ctx / module_ctx calls. Zero value = no capture.
    CaptureSinks *taint.Sinks
}

type Mode int
const (
    ModeStrict   Mode = iota // Real Bazel semantics; unresolvable loads error
    ModeLenient              // Permissive stubs fill unresolved external symbols;
                             // unknown builtins error
    ModeAnalysis             // Lenient + CaptureSinks active + (os, arch) fork support
)
```

### Why this shape (rationale per field)

- **`Version` as enum, not string.** Compile-time validation; no risk
  of typos like `"7.4"` vs `"7.4.0"`. Per-version metadata sits in
  `version/` and is loaded from the enum.
- **`Mode` separate from `LenientLoad`.** The existing `LenientLoad`
  bool is too coarse — it conflates "be tolerant of missing files"
  with "be tolerant of missing symbols." `Mode` is the proper enum.
  Backward compat: when `LenientLoad: true` and `Mode == 0`, the
  library auto-promotes to `ModeLenient` and emits a deprecation log
  the first time per process.
- **`PredeclaredBzl/Build` exposed publicly.** Closes the spike's
  workaround. Critical for any consumer wanting to inject extra
  builtins (canopy's eventual airgap proxy probably wants
  `audit_capture(...)`).
- **`CaptureSinks` only meaningful in `ModeAnalysis`.** Prevents
  accidental coupling: strict eval doesn't pay for capture
  infrastructure.

## API surface — `version` package

```go
package version

type Version int

const (
    VLatest Version = iota   // alias for current active LTS
    V7                       // 7.7.1 (Maintenance until Dec 2026)
    V8                       // 8.7.0 (Maintenance until Dec 2027)
    V9                       // 9.1.0 (Active LTS until Dec 2028)
)

// Latest returns the active LTS this library targets.
func Latest() Version { return V9 }

// Feature is the canonical name of a Bazel semantic / behavior delta.
// Names match what `bazel_features` upstream exposes (where
// applicable) so a synthetic @bazel_features//:features.bzl can be
// keyed off these directly.
type Feature string

// Curated in version/features.go from Bazel release notes +
// bazel-features-bzl upstream (BCR has bazel_features through v1.41.0
// at ~/dev/refs/bazel-central-registry/modules/bazel_features/).
// Each constant carries a doc comment with its provenance.
const (
    // Module-extension features
    FeatureModExtOsArchDependent Feature = "external_deps.module_extension_has_os_arch_dependent"
    FeatureModExtFacts           Feature = "external_deps.module_extension_metadata_facts"
    // Label semantics
    FeatureDeprecatedLabelAPIs   Feature = "label.deprecated_apis_available"
    // Macros
    FeatureSymbolicMacros        Feature = "macros.symbolic"  // Bazel 8+ (confirmed)
    // Bzlmod
    FeatureBzlmodDefault         Feature = "bzlmod.default_on"  // Bazel 7+ (confirmed)
    // ...continued in features.go, M6 curates each entry
)

func (v Version) HasFeature(f Feature) bool { /* table lookup */ }

// ExperimentalFlag is a Bazel `--experimental_*` or `--incompatible_*`
// command-line flag name (without the leading dashes). Orthogonal to
// Version: any Version can have any flag on or off.
type ExperimentalFlag string

const (
    FlagRepoCtxExecuteWasm   ExperimentalFlag = "experimental_repository_ctx_execute_wasm"
    FlagRepoRemoteExec       ExperimentalFlag = "experimental_repo_remote_exec"
    FlagIsolatedExtensionUse ExperimentalFlag = "experimental_isolated_extension_usages"
    FlagNoImplicitWatchLabel ExperimentalFlag = "incompatible_no_implicit_watch_label"
    FlagEnableDeprecatedLabelAPIs ExperimentalFlag = "incompatible_enable_deprecated_label_apis"
    // ...continued in features.go
)

// DefaultValue returns the default of a flag at the given Version.
// Consumers can override via bzl.Options.FeatureFlags.
func (f ExperimentalFlag) DefaultAt(v Version) bool { /* table */ }
```

### Two-axis semantics

`Version` controls *what builtins exist* and *what's deprecated*.
`FeatureFlags` controls *experimental opt-ins* on top.

Example:
```go
opts := bzl.Options{
    Version: version.V9,                              // Bazel 9 surface
    FeatureFlags: map[string]bool{
        "experimental_repository_ctx_execute_wasm": true,  // turn on wasm
    },
}
```

A `.bzl` evaluated under these options sees `repository_ctx` with
`execute_wasm` / `load_wasm` available; without the flag it sees the
Version-9 surface minus those methods.

### `features.go` is the source of truth, curated against upstream

The `Feature` and `ExperimentalFlag` constants are populated at M6
from two cross-referenced sources:

1. `BuildLanguageOptions.java` in the Bazel checkout — enumerates
   every `--experimental_*` / `--incompatible_*` flag the Bazel
   binary accepts. Source for `ExperimentalFlag` constants.
2. The `bazel-features-bzl` companion module (at github.com/
   bazel-contrib/bazel_features; latest in BCR is v1.41.0). Source
   for `Feature` constants and their version-to-bool table.

CI sanity check: for `VLatest`, our `HasFeature(VLatest, f)` for
each `f` matches what `bazel_features.<f>` exposes in the pinned
upstream version. Drift = test failure.

## API surface — `stub` package

```go
package stub

// Permissive is the universal-stub Starlark value. Returned by the
// permissiveLoader for unresolvable external symbols and by the
// stub builtins (`native`, `json` when not using the real `lib/json`,
// etc.) in ModeLenient/ModeAnalysis.
type Permissive struct{}

var Shared = &Permissive{}

func (p *Permissive) String() string                              { return taint.Marker }
// ... implements: Value, Callable, HasAttrs, Mapping, HasBinary, Comparable

// LoaderFor returns a Load function suitable for *starlark.Thread.Load
// that fills unresolved load() targets with Permissive. tryReal is
// consulted first for each module.
func LoaderFor(symbolsByModule map[string][]string, tryReal func(module string) (starlark.StringDict, bool)) func(*starlark.Thread, string) (starlark.StringDict, error)
```

## API surface — `taint` package

```go
package taint

// Marker is the textual sentinel injected by Permissive.String() and
// Permissive.Binary into any string derived from a Permissive. URL
// extraction substring-detects this to flag tainted URLs while
// preserving the recognizable portion of the URL.
const Marker = "<permissive>"

// CapturedURL is one network-fetch call observed during a Mode=Analysis
// evaluation. Multiple URLs from a list arg become multiple records.
type CapturedURL struct {
    URL         string
    SHA256      string
    Integrity   string
    Platform    string
    StripPrefix string
    APIName     string  // "ctx.download" | "ctx.download_and_extract"
    RuleName    string
    Tainted     bool
}

// Sinks aggregates capture outputs from Mode=Analysis evals.
type Sinks struct {
    URLs           []CapturedURL
    Instantiations []RuleInstantiation
    ForkErrors     []ForkError
}

// RuleInstantiation, ForkError, Platform: see spike for current shapes.

// Has reports whether s contains a taint marker; used by flattenURLs
// and any consumer that needs to check an arbitrary string.
func Has(s string) bool { /* strings.Contains(s, Marker) */ }
```

## API surface — `ctx` package additions

```go
// RepositoryCtx is the synthetic repository_ctx passed to a
// repository_rule's impl by eval.InvokeRepositoryRule.
type RepositoryCtx struct { /* fields */ }

// NewRepositoryCtx constructs one with the given platform + attr
// values + capture sinks. Reads the Version off the thread to gate
// .repo_metadata, .environ, etc.
func NewRepositoryCtx(opts RepositoryCtxOptions) *RepositoryCtx

type RepositoryCtxOptions struct {
    Name      string
    OSName    string
    OSArch    string
    OSEnv     map[string]string
    Attrs     map[string]starlark.Value
    Version   version.Version
    Sinks     *taint.Sinks
    WorkspaceRoot string
}

// ModuleCtx is the synthetic module_ctx for module_extension impls.
type ModuleCtx struct { /* fields */ }

// NewModuleCtx constructs one from the caller-supplied modules slice
// and platform/version metadata.
func NewModuleCtx(opts ModuleCtxOptions) *ModuleCtx
```

## API surface — `eval` package additions

```go
// InvokeRepositoryRule drives a captured RepositoryRuleClass's impl
// with a synthetic ctx, forking across the (os, arch) matrix.
// Captured URLs and errors land in opts.Sinks.
func InvokeRepositoryRule(rule *types.RepositoryRuleClass, attrs map[string]starlark.Value, opts InvokeOptions) error

// InvokeModuleExtension drives a captured ModuleExtensionClass's impl
// with a synthetic module_ctx built from modules. Repository-rule
// instantiations are captured into opts.Sinks and dispatched through
// InvokeRepositoryRule.
func InvokeModuleExtension(ext *types.ModuleExtensionClass, modules []ctx.ModuleSpec, opts InvokeOptions) error

type InvokeOptions struct {
    Version   version.Version
    Platforms []ctx.Platform
    MaxSteps  int64
    Timeout   time.Duration
    Sinks     *taint.Sinks   // required for capture
}
```

## Mode semantics — concrete behavior table

| Behavior | ModeStrict | ModeLenient | ModeAnalysis |
|---|---|---|---|
| Unknown builtin → | Error | Error (NOT Permissive — would mask typos) | Error |
| Unresolvable `load("@dep//...", ...)` → | Error | Permissive stubs for requested symbols | Permissive stubs for requested symbols |
| Unresolvable in-workspace `load(":foo.bzl", ...)` → | Error | Error (typo signal) | Error |
| `ctx.execute(...)` → | Errors (no real execution) | Errors | Returns opaque struct, sets fork tainted flag |
| `ctx.download(...)` → | Errors (no real download) | Errors | Records to sink, returns True |
| Step limit | 100M (loose for real eval) | 10M (tighter, for analysis) | 10M |
| Timeout | none (or caller-supplied) | 30s default | 30s default |
| `(os, arch)` fork over `InvokeRepositoryRule` | n/a (not invoked) | n/a | Yes, opts.Platforms |
| Captures populated | n/a | n/a | Yes |

## Versioning — runtime, not import path

The decision to use a `Version` enum on `bzl.Options` (vs `/v8`,
`/v9` Go module subpackages) was explicitly considered. Reasons for
runtime over import-path:

1. **Real consumers span versions.** A linter might evaluate
   `rules_go@7.x` for one repo and `rules_go@9.x` for another in the
   same binary. Runtime selection is the natural shape.
2. **Behavior deltas are small per version.** Bazel evolves the
   Starlark surface incrementally; most versions differ in a handful
   of attribute defaults or one new builtin. Per-version subpackages
   would mostly contain identical code.
3. **Maintenance burden is lower with one source of truth.** A
   per-version code branch invites drift; a delta table is auditable.
4. **Go module subpackages create unnecessary import-path versioning
   ambiguity.** `github.com/.../v8` and `.../v9` import paths force
   consumers into compile-time decision; runtime is more flexible.

When per-version subpackages would be justified: a genuine ABI break
where two consumers in the same binary need divergent Go-type
semantics. We have not identified such a case. If we do, that's the
moment to introduce `/v9`.

## bazel_features compatibility

Real rulesets gate behavior on `bazel_features` — a separately-maintained
Bazel module that exposes Bazel version probes:

```python
load("@bazel_features//:features.bzl", "bazel_features")

if bazel_features.external_deps.module_extension_has_os_arch_dependent:
    extra_kwargs = {"os_dependent": True, "arch_dependent": True}
else:
    extra_kwargs = {}
```

We need to make this work without forcing consumers to mirror
`bazel-features-bzl` into every analysis they run. Two implementation
options:

**Option A — Synthetic bazel_features module.** Pre-populate a
`StringDict` keyed at `@bazel_features//:features.bzl` with the
exact same struct surface, computed from `version.HasFeature(...)`.
When the analyzed `.bzl` loads from `@bazel_features//`, the loader
returns our synthetic.

**Option B — Vendor `bazel-features-bzl` source.** Ship the actual
`.bzl` files from the upstream `bazel-features-bzl` repo and let
them evaluate naturally. Always reflects upstream's view of the
truth, but adds a vendoring dependency.

**Decision:** Option A. Smaller surface area, runs faster (no .bzl
parse), version-controlled via our `Feature` enum. Risk: drift from
the real `bazel-features-bzl` over time. Mitigation: include a
diff test in CI that verifies our `HasFeature(V9, X)` matches what
`bazel-features-bzl` v1.x exposes for Bazel 9.1.

## WASM compatibility check

The library compiles to WASM today (`main.wasm` in tree). New
packages must preserve this:

- `stub/`, `taint/`, `version/` are pure Go, no syscalls, no fs.
  Safe.
- `eval/fork.go` (for (os, arch) matrix) uses no syscalls; just
  iterates a slice. Safe.
- `ctx/repository_ctx.go` is in-memory only; doesn't call
  `exec.Command`. Safe (the whole point of `ctx.execute` returning
  opaque).

CI gate: WASM build must continue to succeed throughout the milestone
sequence. Add to existing build in `justfile` if not already there.

## Open architecture questions

1. **Should `ctx/RepositoryCtx` and `ctx/ModuleCtx` live in `ctx/`
   alongside analysis-time `ctx`, or in a new `repo_ctx/` package?**
   Argument for `ctx/`: discoverable, "all Bazel ctx objects in one
   place." Argument for `repo_ctx/`: separation of concerns
   (analysis vs fetch). **Decision lean: `ctx/`.** Same import path
   reduces consumer friction.
2. **Should `taint/` be a sub-package of `stub/`?** They co-evolve.
   Argument against: `taint.Marker` and the `CapturedURL` schema are
   useful even without Permissive (e.g., a future TaintedString
   wrapper). **Decision lean: keep separate.**
3. **Should `version/` ship its own pkg.go.dev landing for the
   Feature enum?** Yes — Feature names are the contract third-party
   consumers will use. Document each constant.
4. **Should we expose `Mode` constants via the `bzl` package or via
   a new `bzl/mode` subpackage?** Aesthetic question. **Decision
   lean: top-level `bzl.ModeStrict`, etc.** Easier to read at call
   site.
