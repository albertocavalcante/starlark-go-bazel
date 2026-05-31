# 05 — Spike promotion + testing strategy

The mechanical work and the verification work, together so they can
share the file-by-file map.

## File-by-file migration map

The spike lives at `/Volumes/T9/dev/ws/assay/interp/spike/`. Each
file maps to one or more upstream destinations.

### `spike/builtins.go` (235 lines)

Splits across multiple upstream files:

| Spike content | Upstream destination | Notes |
|---|---|---|
| `RepoRuleClass` type | `types/repository_rule_class.go` | Rename to `RepositoryRuleClass`; mirror existing `types.RuleClass` |
| `RuleInstantiation` type, `instSinkKey` const | `taint/sink.go` | Move to capture-sink package |
| `repositoryRuleBuiltin()` | `builtins/repository_rule.go` | |
| `RepoRuleClass.CallInternal` (instantiation capture) | `builtins/repository_rule.go` method | Reads sink from thread.Local |
| `ModuleExtensionClass` type | `types/module_extension_class.go` | |
| `moduleExtensionBuiltin()` | `builtins/module_extension.go` | |
| `attrTypeOf()` | `attr/descriptor.go` helper or deleted | Deleted IF the attr descriptor fix lands (separate plan); kept as fallback otherwise |
| `makeAttrModule()` | `builtins/attr_module.go` (extends existing) | Currently in `attr/module.go`; merge or supplement |
| `predeclared` (package var) | `builtins/builtins.go` `Predeclared(Version)` | Returns version-pinned StringDict |
| `permissiveBuiltin`, `labelBuiltin`, `selectBuiltin`, `structBuiltin`, `failBuiltin`, `noopBuiltin` | **DELETE** | Use the real upstream impls from `builtins/` (select, struct, etc.) and `types.LabelBuiltin` |

### `spike/repoctx.go` (251 lines)

Splits:

| Spike content | Upstream destination |
|---|---|
| `repositoryCtx` struct + methods (Attr, AttrNames, String, etc.) | `ctx/repository_ctx.go` as `RepositoryCtx` |
| `recordDownload()` | `ctx/repository_ctx.go` method |
| `executeMethod()`, `readMethod()`, `whichMethod()` (set tainted) | `ctx/repository_ctx.go` methods |
| `noopReturnNone()`, `passThroughFirst()` (helpers) | `ctx/internal/helpers.go` (or inline) |
| `repositoryOs` struct | `ctx/repository_ctx.go` as `RepositoryOs` |
| `repositoryAttr` struct | `ctx/repository_ctx.go` as `RepositoryAttr` |
| `flattenURLs()` | `taint/sink.go` (helper) |
| `platformLabel()` | `taint/fork.go` (or inline on Platform.Label() method) |
| `starlarkstructFromDict()` | `ctx/internal/struct.go` (helper) |

### `spike/repoeval.go` (232 lines)

Splits:

| Spike content | Upstream destination |
|---|---|
| `EvalOptions` struct | Merged into `bzl.Options` (new fields per architecture plan) |
| `Platform`, `DefaultPlatforms` | `taint/fork.go` |
| `Result` struct | `eval/result.go` (`EvalResult`) |
| `InvokeResult`, `ForkError` | `taint/sink.go` |
| `EvalSource()` | Replace with `bzl.Interpreter.Eval()` + `bzl.Options.Mode = ModeAnalysis` |
| `InvokeRule()`, `StringAttrs()` | `eval/invoke.go` as `InvokeRepositoryRule()`, `StringAttrs()` |
| `dedupe()`, `anyKey()` | `taint/sink.go` helpers |
| Constants `defaultMaxSteps`, `defaultTimeout` | `eval/defaults.go` |

### `spike/extension.go` (116 lines)

| Spike content | Upstream destination |
|---|---|
| `ModuleSpec`, `TagInstance` | `ctx/module_ctx.go` |
| `ExtensionResult` | `taint/sink.go` |
| `InvokeExtension()` | `eval/invoke.go` as `InvokeModuleExtension()` |
| `buildModuleCtx()`, `buildBazelModule()` | `ctx/module_ctx.go` constructors |
| `returnFalse()` | Inline |

### `spike/permissive.go` (75 lines)

| Spike content | Upstream destination |
|---|---|
| `PermissiveMarker` const | `taint/taint.go` as `Marker` |
| `Permissive` type + all methods (String, Type, Truth, Hash, Name, CallInternal, Attr, AttrNames, Get, Binary, CompareSameType) | `stub/permissive.go` |
| `sharedPermissive` package var | `stub/permissive.go` as `Shared` |
| `permissiveLoader()` | `stub/loader.go` as `LoaderFor()` |

### `spike/loadscan.go` (33 lines)

| Spike content | Upstream destination |
|---|---|
| `loadSymbolsFromFile()` | `stub/loader.go` helper |

### `spike/*_test.go`

Move-and-rename pattern; each test stays in spirit but invokes via
the upstream API. The acceptance criterion is "23 spike tests still
pass under the new shape."

- `repoeval_test.go` → split between
  `builtins/repository_rule_test.go`, `eval/invoke_test.go`,
  `taint/sink_test.go`.
- `extension_test.go` → `eval/invoke_test.go` extension cases.
- `realcorpus_test.go` → `eval/invoke_realcorpus_test.go`. Add
  build tag `realcorpus` so CI doesn't fail on machines without
  `~/dev/refs/rules_go`.
- `taint_test.go` → split between `stub/permissive_test.go` and
  `taint/sink_test.go` and `ctx/repository_ctx_test.go`.

### `spike/README.md`

Replaced by `docs/plans/01-bazel-builtins-emulation/` (this plan).
spike/README.md gets a final paragraph that says "promoted; this
package is scheduled for deletion in M8" and otherwise kept until
M7 finishes.

## API rename map (publicly visible types only)

| Spike name | Upstream name | Why |
|---|---|---|
| `RepoRuleClass` | `types.RepositoryRuleClass` | Match `types.RuleClass` |
| `ModuleExtensionClass` | `types.ModuleExtensionClass` | Match same |
| `repositoryCtx` | `ctx.RepositoryCtx` | Public, mirrors existing `ctx.Ctx` |
| `repositoryOs` | `ctx.RepositoryOs` | Public |
| `repositoryAttr` | `ctx.RepositoryAttr` | Public |
| `CapturedURL` | `taint.CapturedURL` | Lives with the rest of capture state |
| `RuleInstantiation` | `taint.RuleInstantiation` | Same |
| `ForkError`, `Platform` | `taint.ForkError`, `taint.Platform` | Same |
| `ModuleSpec`, `TagInstance` | `ctx.ModuleSpec`, `ctx.TagInstance` | They configure module_ctx |
| `Permissive`, `sharedPermissive` | `stub.Permissive`, `stub.Shared` | Permissive-as-API lives in stub/ |
| `PermissiveMarker` | `taint.Marker` | Taint sentinel |
| `EvalOptions` | Merged into `bzl.Options` | Single options surface |
| `EvalSource` | `bzl.Interpreter.Eval` (existing) | Same entry point |
| `InvokeRule` | `eval.InvokeRepositoryRule` | Verb-noun |
| `InvokeExtension` | `eval.InvokeModuleExtension` | Same |
| `StringAttrs` | `eval.StringAttrs` | Same |
| `DefaultPlatforms` | `taint.DefaultPlatforms` | Lives with Platform |
| `Result`, `InvokeResult`, `ExtensionResult` | `eval.EvalResult`, `eval.InvokeResult`, `eval.ExtensionResult` | Same |

## Testing strategy

### Level 1 — Unit tests (per package)

Each new package has its own `_test.go` covering:

- `stub/permissive_test.go`: every interface method, Compare paths
  (same-type EQ/NEQ/ordered, cross-type via Equal), Binary string
  concat, deep attr chains.
- `stub/loader_test.go`: tryReal-first, fallback to Permissive, the
  cache, module-not-in-symbols-map case.
- `taint/sink_test.go`: dedupe correctness (including the platform
  iteration ordering fix from the spike code review), CapturedURL
  field merging.
- `taint/marker_test.go`: `Has()` with marker, without marker, with
  marker embedded; false-positive smoke against the assay-corpus URL
  list.
- `taint/fork_test.go`: `Platform.Label()`, `DefaultPlatforms` shape.
- `ctx/repository_ctx_test.go`: each attr (`download`,
  `download_and_extract`, `execute` returns struct, `read` returns
  Permissive + taints, `os.name`/`os.arch` forking, `attr.X` lookup
  hits + misses).
- `ctx/module_ctx_test.go`: same for `module_ctx`, plus
  `bazel_module` field access.
- `version/version_test.go`: `Latest()`, `HasFeature()` per-Version
  table, deprecated-builtin behavior at strict.
- `version/features_test.go`: feature flags match `bazel-features`
  for the named version (golden file diff).
- `builtins/repository_rule_test.go`: capture from `repository_rule(
  implementation=..., attrs={...}, environ=[...], local=True, ...)`.
- `builtins/module_extension_test.go`: same for module_extension +
  tag_class kwargs (os_dependent, arch_dependent, reproducible).
- `eval/invoke_test.go`: synthetic single-rule + extension drivers;
  matches the 12 spike tests.

### Level 2 — Real-corpus tests

The assay corpus (six modules) gets a top-level test at
`eval/realcorpus_test.go`:

```go
func TestEval_RealCorpus_RulesGoSDK(t *testing.T) {
    // Equivalent of spike's realcorpus_test.go but through public API
}
```

Build tag `realcorpus`. CI runs in two phases:
1. Unit phase: every PR, fast.
2. Realcorpus phase: gated on `~/dev/refs/<module>` presence; runs
   nightly + on demand.

The corpus to validate against:
- `rules_cc` — exercises analysis-time `rule()` AND
  `cc_register_toolchains`-style repo rules
- `rules_go` — `go_download_sdk_rule`, platform fork, the canonical
  case
- `rules_java` — toolchain-heavy
- `rules_python` — large impl, exercises step budget
- `rules_jvm_external` — `maven_install` (downloads via coursier,
  expected to produce tainted URLs)
- `bazel-gazelle` — meta-tool, light surface

### Level 3 — Differential tests (the gold standard)

For canopy airgap's use case, run a real `bazel fetch
--experimental_repository_resolved_file=resolved.bzl` against a
synthetic workspace; load the resolved.bzl; compare its captured
URL set to what our library produces.

```go
func TestEval_Differential_RulesGoSDK(t *testing.T) {
    // Skip if BAZEL_FETCH_REFERENCE not set; that env var points at
    // a directory containing pre-recorded resolved.bzl outputs.

    bazelURLs := loadResolvedBzl(ref+"/rules_go_sdk_resolved.bzl")
    spikeURLs := runEvalAgainstRulesGoSDK()

    diff := compareURLs(bazelURLs, spikeURLs)
    if diff.Missing != 0 || diff.Extra != 0 {
        t.Errorf("differential: missing=%d extra=%d", diff.Missing, diff.Extra)
    }
}
```

Acceptance for this test: `Missing == 0` for URLs Bazel resolves
without `ctx.execute` (the rest are expected to be tainted in our
output and not appear in Bazel's resolved.bzl — that's correct
behavior, separate metric).

### Level 4 — Version-cross tests

The same `.bzl` file evaluated under multiple `Version` targets
produces the expected behavior deltas.

```go
func TestVersion_RepoMetadataGatedByVersion(t *testing.T) {
    src := `
def _impl(ctx):
    if hasattr(ctx, "repo_metadata"):
        ctx.report_progress("repo_metadata supported")
    ctx.download(url = "https://x.com/y.tar.gz", sha256 = "z")

r = repository_rule(implementation=_impl, attrs={})
`
    // V7 surface does not include repo_metadata
    // V8+ does
}
```

### Level 5 — Fuzz tests

`go test -fuzz` against:
- `stub/permissive_test.go::FuzzPermissiveBinary` — random op tokens,
  random y values.
- `taint/marker_test.go::FuzzHas` — random strings.
- `eval/invoke_test.go::FuzzInvokeWithRandomBzl` — small random
  Starlark snippets; assert no panic, only ForkError or success.

### Level 6 — Performance regressions

`go test -bench` baselines, NOT pre-committed budgets. Approach:

1. At M5 completion, record a baseline for the operations below.
2. CI publishes per-PR delta against baseline.
3. Regressions of >20% gate merge; smaller deltas surface as comments.

Operations to baseline:
- `bzl.Interpreter.Eval(rules_go/sdk.bzl)` cold and warm-cache.
- `InvokeRepositoryRule(go_download_sdk_rule)` × the default 6-platform matrix.
- `InvokeModuleExtension(go_sdk, 1 root module)` end to end.
- `taint.Has(url)` over a synthetic corpus.
- `Permissive.Attr` deep chain.

The spike's full suite runs in ~140ms today; production load on
canopy's full corpus is not yet measured. Set actual numbers at M5
based on observed baseline.

### Level 7 — WASM build smoke

Run `GOOS=js GOARCH=wasm go build ./...` after every milestone.
Library must remain WASM-compatible (no new syscall imports, no
fs/exec/os/net direct calls).

## Acceptance criteria per milestone

See [06-risks-open-questions-and-milestones.md](06-risks-open-questions-and-milestones.md#milestone-sequence)
for the canonical milestone-acceptance table. Testing this plan's
work is the same set of gates listed there; this file's job is to
spell out how each gate is verified, not to duplicate the list.

## Risks specific to testing

1. **Realcorpus tests are non-hermetic.** Depend on
   `~/dev/refs/<module>` presence and Bazel version pinned on
   developer machine. Mitigation: build tag + CI gating + a recorded
   golden snapshot mode (run once, capture, replay).
2. **Differential tests require Bazel installation.** Same mitigation
   — record reference outputs once, store in `testdata/`.
3. **Fuzz tests find genuine bugs in go.starlark.net.** Has happened
   before. Process: file upstream, work around locally.
4. **Performance budgets get stale.** Re-baseline at every minor
   release.

## What is OUT of testing scope

- **Building real workspaces.** We don't invoke Bazel itself.
- **Network access in tests.** Differential tests use pre-recorded
  resolved.bzl; no live `bazel fetch` in CI.
- **Cross-platform behavior of the host.** Library is pure Go; we
  test on Linux + macOS CI matrices.
