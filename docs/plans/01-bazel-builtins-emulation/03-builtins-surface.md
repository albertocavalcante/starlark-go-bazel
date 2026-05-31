# 03 — Builtins surface matrix

The list of Bazel Starlark builtins / modules / value types, mapped
to implementation tier and current/proposed status.

**Sources:** rows below are cross-checked against the upstream Bazel
sources at `/Users/adsc/dev/refs/bazel/src/main/java/...`. Specifically:

- `analysis/starlark/StarlarkRuleClassFunctions.java` — top-level
  `rule`, `provider`, `aspect`, `Label`, `select`, `attr` module.
- `analysis/starlark/BazelBuildApiGlobals.java` — `configuration_field`,
  `select`, `depset`, etc.
- `bazel/repository/starlark/StarlarkRepositoryModule.java` —
  `repository_rule` kwargs.
- `bazel/bzlmod/ModuleExtension.java` — `module_extension` kwargs.
- `bazel/repository/starlark/StarlarkBaseExternalContext.java` —
  methods shared by `repository_ctx` and `module_ctx`.
- `bazel/repository/starlark/StarlarkRepositoryContext.java` —
  `repository_ctx`-specific methods.
- `bazel/bzlmod/ModuleExtensionContext.java` — `module_ctx`-specific
  methods.

When in doubt, re-grep `grep -nE "name = \".*\"" <file>` to confirm
against the version of Bazel checked out at `~/dev/refs/bazel/`.

## Tier definitions

- **T1 — Required for canopy airgap (plan 11).** Without these, the
  URL-extraction pipeline can't work.
- **T2 — Required for canopy compat-analyzer (plan 05).** Need to
  evaluate `MODULE.bazel` and detect breaking changes.
- **T3 — Useful for other consumers** (scip-bazel, future linters,
  buildifier-like tools).
- **T4 — Nice to have but out of immediate scope.**
- **OOS — Out of scope for this library.** Real build execution.

## Top-level predeclared builtins

| Symbol | Tier | Current | Target | Notes |
|---|---|---|---|---|
| `rule()` | T2 | Implemented | Implemented + attr descriptor fix (separate plan) | |
| `provider()` | T2 | Implemented | Implemented | |
| `aspect()` | T2 | Implemented | Implemented | |
| `repository_rule(implementation, attrs, local, environ*, configure, remotable*, doc)` | T1 | **MISSING** | **builtins/repository_rule.go** | M2. Kwargs verified at `StarlarkRepositoryModule.java:57`. `environ` is **deprecated** (rules should migrate to `repository_ctx.getenv`); `remotable` is **experimental** behind `--experimental_repo_remote_exec`. |
| `module_extension(implementation, tag_classes, doc, environ*, os_dependent, arch_dependent)` | T1 | **MISSING** | **builtins/module_extension.go** | M2. Verified at `StarlarkRepositoryModule.java:210`. `environ` is **deprecated** (migrate to `module_ctx.getenv`). Note: `reproducible` is NOT a kwarg here — it lives on `module_ctx.extension_metadata()`. |
| `macro(implementation, attrs, inherit_attrs, doc, finalizer)` | T4 | Missing | Permissive stub initially | **New in Bazel 8.** Symbolic macros. M9+. |
| `tag_class(attrs, doc)` | T1 | **MISSING** | **builtins/tag_class.go** | M2 |
| `select()` | T2 | Implemented | Implemented | |
| `struct()` | T1 | Implemented | Implemented | |
| `depset()` | T2 | Implemented | Implemented | |
| `Label()` | T1 | Implemented (`types.LabelBuiltin`) | Implemented | Spike's stub `labelBuiltin` gets removed in M7 |
| `fail()` | T1 | Implemented (Starlark builtin) | Implemented | |
| `print()` | T1 | Implemented (Starlark builtin) | Implemented | |
| `attr.*` | T1 | Implemented | Implemented (descriptor fix is separate) | |
| `native` (module) | T2 | Implemented | + module_name, module_version, repo_name | M6 |
| `json` (module) | T1 | Permissive stub today | Decision in M6: (a) wire `go.starlark.net/lib/json`, (b) keep Permissive | |
| `bazel_features` (when `@bazel_features` loaded) | T1 | Permissive stub | Real per-version struct via synthetic load | M6 |
| `configuration_field()` | T3 | Missing | Permissive stub initially | M9 |
| `analysis_test_transition()` | T4 | Missing | Permissive stub | M9 |
| `exec_group()` | T4 | Missing | Permissive stub | M9 |
| `toolchain_type()` | T2 | Audit in M2 | If missing, add | bzlwalk already detects callsites |

## `ctx` (analysis-time, passed to `rule()` impl)

Existing in `starlark-go-bazel/ctx/`. NOT touched by this plan —
analysis-time is separate from repository-rule-time.

| Attribute / method | Status |
|---|---|
| `ctx.label`, `ctx.attr`, `ctx.files`, `ctx.file`, `ctx.executable`, `ctx.outputs`, `ctx.actions`, `ctx.runfiles`, `ctx.expand_*` | Implemented |
| `ctx.workspace_name`, `ctx.build_file_path`, `ctx.configuration`, `ctx.fragments`, `ctx.var`, `ctx.features`, `ctx.disabled_features`, `ctx.info_file`, `ctx.version_file`, `ctx.toolchains`, `ctx.exec_groups`, `ctx.coverage_instrumented`, `ctx.tokenize`, `ctx.resolve_command`, `ctx.resolve_tools`, `ctx.package_relative_label` | Implemented |

## `repository_ctx` (passed to `repository_rule()` impl) — NEW

Surface verified against `StarlarkBaseExternalContext.java` (shared
with module_ctx) and `StarlarkRepositoryContext.java`
(repository_ctx-specific) at `~/dev/refs/bazel/`.

### Shared base methods (from `StarlarkBaseExternalContext`)

| Method | Tier | Spike has | Target | Notes |
|---|---|---|---|---|
| `download(url, output, sha256, integrity, executable, allow_fail, canonical_id, auth, headers, block, ...)` | T1 | Yes (sink) | Full kwarg | |
| `download_and_extract(url, output, sha256, type, strip_prefix/stripPrefix, integrity, allow_fail, canonical_id, auth, headers, rename_files, ...)` | T1 | Yes | Full kwarg | |
| `execute(arguments, timeout, environment, quiet, working_directory)` | T1 | Yes (opaque + taint) | Returned struct has stdout/stderr/return_code | |
| `execute_wasm(wasm_path, function, input, timeout, max_memory_bytes, ...)` | T4 | No | Permissive stub | Exotic; document as Permissive-stub-acceptable |
| `load_wasm(path)` | T4 | No | Permissive stub | Same |
| `extract(archive, output, strip_prefix, rename_files, watch_archive)` | T2 | Noop | Noop | |
| `file(path, content, executable, legacy_utf8)` | T2 | Noop | Noop | |
| `getenv(name, default)` | T2 | No | Pulls from RepositoryCtxOptions.OSEnv; sets tainted if name absent and `default` not supplied | M3 |
| `os` (struct with `name`, `arch`, `environ`) | T1 | Yes | Same; environ from RepositoryCtxOptions.OSEnv | |
| `path(arg)` | T2 | Pass-through first arg | Stub Path object | |
| `read(path, watch)` | T1 | Yes (opaque + taint) | Same | |
| `report_progress(status)` | T1 | Noop | Noop | |
| `watch(path)` | T3 | No | Noop | Bazel 7+ |
| `which(program)` | T2 | Opaque + taint | Same | |

### Repository-ctx-specific methods (from `StarlarkRepositoryContext`)

| Method/attr | Tier | Spike has | Target | Notes |
|---|---|---|---|---|
| `name` | T1 | Yes | From RepositoryCtxOptions.Name | |
| `original_name` | T2 | No | Same | Bazel 8+ |
| `workspace_root` | T2 | "/synthetic/workspace" stub | Configurable | |
| `attr` (proxy) | T1 | Yes | Pass attrs; unknown → empty string (consider tainting in M3) | |
| `symlink(target, name)` | T2 | Noop | Noop | |
| `template(path, template, substitutions, executable, watch_template)` | T2 | Noop | Noop | |
| `delete(path)` | T2 | Noop | Noop | |
| `rename(src, dst)` | T2 | No | Noop | Add to NEW ctx |
| `patch(patch_file, strip, watch_patch)` | T2 | No | Noop | Add to NEW ctx; common in repo rules |
| `watch_tree(path)` | T3 | No | Noop | Bazel 7+ |
| `repo_metadata(reproducible, attrs_for_reproducibility)` | T2 | No | Return Permissive struct | Bazel 8+; gate on Version |

### `exec_result` struct (returned by ctx.execute)

| Field | Tier | Spike has | Target |
|---|---|---|---|
| `.stdout` | T1 | Permissive | Same |
| `.stderr` | T1 | Permissive | Same |
| `.return_code` | T1 | Int(0) | Configurable per fork; consider tainting Int(0) too |

## `module_ctx` (passed to `module_extension()` impl) — NEW

Surface verified against `ModuleExtensionContext.java`. Inherits all
shared base methods from `StarlarkBaseExternalContext` (table above:
download, download_and_extract, execute, execute_wasm, load_wasm,
extract, file, getenv, os, path, read, report_progress, watch, which).

### Module-ctx-specific methods

| Method/attr | Tier | Spike has | Target | Notes |
|---|---|---|---|---|
| `modules` | T1 | Yes (list of bazel_module) | Same | |
| `is_dev_dependency(tag)` | T2 | Returns False stub | Configurable per `ctx.TagInstance.IsDevDep` | M3 |
| `is_isolated` | T3 | No | Returns False stub | |
| `root_module_has_non_dev_dependency` | T2 | True stub | Configurable | |
| `extension_metadata(root_module_direct_deps, root_module_direct_dev_deps, reproducible, facts)` | T2 | Noop returning None | Capture into sink for analysis | M5 |
| `facts` | T3 | No | Permissive stub initially | Bazel 7+ |

## `bazel_module` (element of module_ctx.modules) — NEW

| Field | Tier | Spike has | Target |
|---|---|---|---|
| `.name` | T1 | Yes | Yes |
| `.version` | T1 | Yes | Yes |
| `.is_root` | T1 | Yes | Yes |
| `.is_dev_dependency` | T2 | No | Yes |
| `.tags` (per-tag-class list of structs) | T1 | Yes | Yes |
| `.tags.<tag_name>[<index>].<attr>` | T1 | Yes | Yes |

## `native` module additions

| Symbol | Tier | Status | Notes |
|---|---|---|---|
| `native.glob` | T2 | Implemented (empty list in analysis) | |
| `native.existing_rule` | T2 | Implemented | |
| `native.package_name` | T2 | Implemented | |
| `native.package_relative_label` | T2 | Implemented | |
| `native.module_name()` | T1 | **MISSING** | Implement: returns current module name from thread context |
| `native.module_version()` | T1 | **MISSING** | Same |
| `native.repo_name()` | T2 | **MISSING** | Bazel 7+ |
| `native.repository_name()` (deprecated) | T3 | **MISSING** | Bazel 6 only; defer |
| `native.register_toolchains` | T2 | **MISSING** | Noop in analysis |
| `native.register_execution_platforms` | T3 | **MISSING** | Noop |
| `native.bind` (deprecated) | OOS | **MISSING** | Don't implement |

## Value types

| Type | Tier | Status |
|---|---|---|
| `Label` | T1 | Implemented (`types.Label`) |
| `RuleClass` | T2 | Implemented (`types.RuleClass`) — attr descriptor stub gap |
| `RepositoryRuleClass` | T1 | **MISSING** — M2 |
| `ModuleExtensionClass` | T1 | **MISSING** — M2 |
| `TagClass` | T1 | **MISSING** — M2 (lightweight) |
| `AspectClass` | T2 | Implemented |
| `ProviderClass` | T2 | Implemented |
| `Provider` (instance) | T2 | Implemented (DefaultInfo, etc.) |
| `Depset` | T2 | Implemented |
| `Selector` | T2 | Implemented |
| `Struct` | T1 | Implemented |
| `ModuleVersion` | T3 | **MISSING** — Bazel 7+ |

## Providers (predeclared)

| Provider | Status |
|---|---|
| `DefaultInfo` | Implemented |
| `OutputGroupInfo` | Implemented |
| `Runfiles` | Implemented |
| `RunEnvironmentInfo` | **MISSING** — T3 |
| `InstrumentedFilesInfo` | **MISSING** — T3 |
| `CcInfo`, `JavaInfo`, etc. (language-specific) | OOS — not predeclared, defined in their respective rulesets |

## Bazel version landscape (grounded)

From the LTS support matrix in
`~/dev/refs/bazel-llms-full.txt:27090-27108` (current as of the
checkout date):

| Bazel LTS | Stage | Latest patch | End of support |
|---|---|---|---|
| 4 | Deprecated | 4.2.4 | Jan 2024 |
| 5 | Deprecated | 5.4.1 | Jan 2025 |
| 6 | Deprecated | 6.6.0 | Dec 2025 |
| 7 | Maintenance | 7.7.1 | Dec 2026 |
| 8 | Maintenance | 8.7.0 | Dec 2027 |
| 9 | **Active** | 9.1.0 | Dec 2028 |
| 10 | Rolling | (rolling) | n/a |

**`VLatest` should track Bazel 9 stable.** Bazel 6 is deprecated and
support is opt-in only; this plan does not commit to a working V6.
Bazel 7 is the minimum *supported* target.

## Two orthogonal version axes

It's important to keep these separate when designing `version/`:

**Axis A — Bazel LTS major.** Selects the *default* surface (which
builtins exist, what's deprecated, what's removed). This is the
`Version` enum (`V7`, `V8`, `V9`).

**Axis B — Per-feature experimental/incompatible flags.** Bazel ships
many features behind individual `--experimental_*` or
`--incompatible_*` flags before they go stable. These are orthogonal
to LTS major. A consumer running Bazel 9.1 might still have
`--experimental_repository_ctx_execute_wasm` off — that's a flag
choice, not a version choice. The library should model this as
`bzl.Options.FeatureFlags map[string]bool` (additive to `Version`),
so consumers can target "Bazel 9 with execute_wasm OFF" vs "Bazel 9
with execute_wasm ON." `Version`-defaults give sensible behavior
without the flag map.

## Verified per-version deltas

**Sources:** `~/dev/refs/bazel/CHANGELOG.md` (per-release notes, all
LTS through 9.0.0 inclusive), `~/dev/refs/bazel-llms-full.txt`
(reference docs), file-level grep of `~/dev/refs/bazel/src/main/java/`
sources, `BuildLanguageOptions.java` flag definitions.

### Pinned to specific LTS

| Feature | First Bazel LTS | Source | Notes |
|---|---|---|---|
| `module_ctx.extension_metadata()` (function exists) | **6.2.0** (2023-05-09) | CHANGELOG L3950, PR #18174 | Returns metadata struct; just the function, not its kwargs |
| Bzlmod enabled by default | **7.0.0** (2023-12-11) | CHANGELOG L2476, issue #18958 | The major Bazel-7 marquee change |
| `use_repo_rule` directive in MODULE.bazel | **7.0.0** (2023-12-11) | CHANGELOG L2470 | Declare repos visible only within a module |
| `--enable_workspace` flag (lets users opt out of WORKSPACE) | **7.1.2** (2024-05-08) | CHANGELOG L2120, PR #20855 | Default still true at this point |
| Symbolic macros (`macro()` builtin) | **8.0.0** (2024-12-09) | LLM doc 7162 + 26541 + 31727; CHANGELOG references confirm | Bazel-8 marquee change |
| `--enable_workspace` flag default flipped + behavior change | **9.0.0** (2026-01-20) | CHANGELOG L1329 in 9.0.0-pre.20250121.1 | "`--enable_bzlmod` and `--enable_workspace` flags are now [updated]" |
| `repository_ctx.repo_metadata(reproducible, attrs_for_reproducibility)` | **9.0.0** | CHANGELOG L985 in 9.0.0-pre.20250516.1 | Verified from per-release breakdown |
| `module_ctx.extension_metadata(facts=)` + `module_ctx.facts` | **9.0.0** | CHANGELOG L340-341, L370-371 in 9.0.0-pre.20251022.1 | New facts mechanism for cross-eval extension state |

### Flag-gated, NOT version-gated (orthogonal axis)

| Feature | Gating flag | Status | Source |
|---|---|---|---|
| `repository_rule(remotable=)` | `--experimental_repo_remote_exec` | Still experimental as of 9.0 / 10-pre | LLM doc 32104, `StarlarkBaseExternalContext.java:1789` |
| `repository_ctx.execute_wasm` / `load_wasm` | `--experimental_repository_ctx_execute_wasm` | Still experimental | `StarlarkBaseExternalContext.java:2046, 2143`; `BuildLanguageOptions.java:1103` |
| `module_ctx.is_isolated` (and isolated extension usages) | `--experimental_isolated_extension_usages` | Still experimental | `ModuleExtensionContext.java:171` |
| `--incompatible_no_implicit_watch_label` | Same name | Incompatible flag (defaults flip in future LTS) | `BuildLanguageOptions.java:1042` |
| `native.repository_name`, `Label.workspace_name`, `Label.relative` (legacy) | `--incompatible_enable_deprecated_label_apis` | Toggle-gated; default flip planned | LLM doc 13601 |

### Deprecations (current)

| API | Status | Migration target |
|---|---|---|
| `repository_rule(environ=)` | Deprecated | `repository_ctx.getenv` |
| `module_extension(environ=)` | Deprecated | `module_ctx.getenv` |
| `bind()` | Deprecated (long-standing) | `alias()` |

## Pinned via git-blame (unshallowed Bazel history, 2026-05-20)

The fetch since 2022-01-01 plus tags resolved the four pending
unknowns. Each row pairs the introducing commit with the first LTS
release that contains it.

| Feature | First Bazel LTS | Commit | Notes |
|---|---|---|---|
| `module_extension(os_dependent=, arch_dependent=)` | **7.0.0** (2023-12-11) | `c1165a9943` (2023-08-29) "Rename module extension fields" | Lands with the Bazel-7 Bzlmod-default-on push |
| `native.repo_name()` | **7.0.0** (2023-12-11) | `73ed74ec5d` (2022-09-05) "Allow modules to specify their own repo names" | Replacement for `native.repository_name` (which becomes deprecated) |
| `repository_ctx.getenv` (and `module_ctx.getenv`) | **8.0.0** (2024-12-09) | `c230e39fb2` (2024-01-18) "[rfc] Allow repository rules to lazily declare environment variable deps" | Pairs with `environ=` deprecation; same Bazel release |
| `repository_ctx.watch` / `module_ctx.watch` | **8.0.0** (2024-12-09) | `a5376aa3e1` (2024-02-14) "Watch arbitrary file in repo rules" | `watch_tree` likely same release; verify if needed |

Reproducer:
```bash
cd ~/dev/refs/bazel
git log -S '"<method>"' --reverse --format='%h %ad %s' --date=short -- <path>.java | head -1
git tag --contains <sha> | grep -E '^[789]\.[0-9]+\.[0-9]+$' | sort -V | head -1
```

## Complete per-version delta table

Consolidating CHANGELOG-derived + git-blame-derived data:

| Feature | First LTS | Source |
|---|---|---|
| `module_ctx.extension_metadata()` function | **6.2.0** | CHANGELOG (PR #18174) |
| `module_extension(os_dependent=, arch_dependent=)` | **7.0.0** | git-blame `c1165a9943` |
| `native.repo_name()` | **7.0.0** | git-blame `73ed74ec5d` |
| Bzlmod enabled by default | **7.0.0** | CHANGELOG (issue #18958) |
| `use_repo_rule` directive | **7.0.0** | CHANGELOG |
| `--enable_workspace` flag (opt-out introduced) | **7.1.2** | CHANGELOG (PR #20855) |
| `repository_ctx.getenv` / `module_ctx.getenv` | **8.0.0** | git-blame `c230e39fb2` |
| `repository_ctx.watch` / `module_ctx.watch` | **8.0.0** | git-blame `a5376aa3e1` |
| Symbolic macros (`macro()` builtin) | **8.0.0** | CHANGELOG + LLM doc |
| `repository_rule(environ=)` deprecated | **8.0.0** | Pairs with getenv landing |
| `module_extension(environ=)` deprecated | **8.0.0** | Same |
| `--enable_workspace` default behavior change | **9.0.0** | CHANGELOG (9.0.0-pre.20250121.1) |
| `repository_ctx.repo_metadata(reproducible, attrs_for_reproducibility)` | **9.0.0** | CHANGELOG (9.0.0-pre.20250516.1) |
| `module_ctx.extension_metadata(facts=)` + `module_ctx.facts` | **9.0.0** | CHANGELOG (9.0.0-pre.20251022.1) |

**M6 deliverable** (now downsized): `version/deltas.go` codifying this
table + a CI gate diff against `bazel-features-bzl` for `VLatest`.

## `Version` enum (revised)

```go
type Version int

const (
    VLatest Version = iota   // alias; current value = V9
    V7                       // 7.7.1 (Maintenance through Dec 2026)
    V8                       // 8.7.0 (Maintenance through Dec 2027)
    V9                       // 9.1.0 (Active through Dec 2028)
)

// V6 deliberately omitted: Bazel 6 reached end-of-support in Dec 2025.
// Adding it requires the consumer to opt in via a separate Legacy
// flag — not in scope for this plan.

// Rolling (Bazel 10) is not enumerated; consumers wanting that
// track must accept rolling semantics and use bzl.Options.FeatureFlags
// to enable specific incoming features.
```

Earlier draft had `V7_0` / `V7_4` etc. minor granularity — too fine.
Real consumers target an LTS major; minor differences are absorbed
by `FeatureFlags` if they matter.

## What's intentionally NOT in this matrix

- **Genrule / cc_library / py_library** and other concrete rule
  *implementations*. Those live in rulesets (`@rules_cc`, etc.), not
  in Bazel core; we don't reimplement them.
- **Real toolchain resolution.** Analysis-time toolchain resolution
  requires evaluating constraint_setting / constraint_value / platform
  / toolchain rules across the whole build graph. Out of scope.
- **Configuration transitions.** Same reason.
- **`bzl_library` rule** (from rules_skylib) — that's user-land too.
- **`buildifier`-specific lints.** Buildifier is a separate tool.

## Tier-1 implementation order (matches milestone sequence)

The T1 entries above map directly to the milestones in
`06-risks-open-questions-and-milestones.md`:

1. M1 — bzl.Options surface (no new builtins)
2. M2 — `repository_rule()`, `module_extension()`, `tag_class()`
3. M3 — `repository_ctx`, `module_ctx`, `bazel_module`
4. M4 — `stub/Permissive` + lenient loader
5. M5 — `taint/` + capture sinks + Mode=Analysis wiring
6. M6 — `version/`, `bazel_features` synthetic, `native.module_name/version`
