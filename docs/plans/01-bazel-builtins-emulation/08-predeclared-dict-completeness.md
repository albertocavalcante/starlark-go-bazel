# 08 — Predeclared dict completeness

**Scope:** Wire every builtin that the library implements into
`eval/evaluator.go::makeBzlPredeclared`, and add a guard test that
catches future "added the builtin, forgot to register it" drift.

**Origin:** discovered during downstream assay's E1c work alongside
plan 07. The library's `builtins.Aspect` exists but isn't
registered in the eval-time predeclared dict; real `.bzl` files
that call `my_aspect = aspect(...)` fail to evaluate with
"undefined: aspect" unless the caller injects `aspect` themselves
via `bzl.Options.PredeclaredBzl`.

The one-line fix is obvious. The structural fix — a drift-
detection test family that prevents the same bug from recurring
on M2's `repository_rule()`, `module_extension()`, `tag_class()` —
is what this plan covers.

## Current state

`eval/evaluator.go::makeBzlPredeclared` (around line 270) returns a
hand-maintained `starlark.StringDict`:

```go
func makeBzlPredeclared() starlark.StringDict {
    return starlark.StringDict{
        "Label":            starlark.NewBuiltin("Label", types.LabelBuiltin),
        "provider":         starlark.NewBuiltin("provider", providerBuiltin),
        "struct":           starlark.NewBuiltin("struct", starlarkstruct.Make),
        "depset":           starlark.NewBuiltin("depset", types.DepsetBuiltin),
        "rule":             starlark.NewBuiltin("rule", types.RuleBuiltin),
        "repository_rule":  starlark.NewBuiltin("repository_rule", builtins.RepositoryRule),
        "module_extension": starlark.NewBuiltin("module_extension", builtins.ModuleExtension),
        "tag_class":        starlark.NewBuiltin("tag_class", builtins.TagClass),
        "attr":             newAttrModule(),
        "native":           newNativeStub(),
        "True":             starlark.True,
        "False":            starlark.False,
        "None":             starlark.None,
    }
}
```

`aspect` is absent. The `builtins.Aspect` function exists in
`builtins/aspect.go`; the wiring is just missing.

Adding a builtin to `builtins/` does not automatically register it.
The plan documents are the only place that asserts "this builtin
should be available." Without a guard test, M2's new builtins are
likely to ship with a wiring step that's easy to overlook.

## What completeness means

Two layers:

1. **Functional completeness** — every builtin that plan 03's
   surface matrix marks as "Implemented" or "Required (M2)" is
   present in `makeBzlPredeclared` for the relevant Version.

2. **Drift detection** — a test family that catches future
   misalignment between the surface matrix and the predeclared
   dict.

### Functional completeness — what to wire NOW

Add `aspect` immediately:

```go
"aspect": starlark.NewBuiltin("aspect", builtins.Aspect),
```

(M2 lands `repository_rule`, `module_extension`, `tag_class`
similarly. The single-line wiring is the trivial part; the test
that catches "forgot to wire" is the load-bearing part.)

When M1 introduces `Predeclared(Version Version)`, the version-
gates apply (e.g., `tag_class` is conditional on V7+). The
unconditional set above is the V7+ baseline; per-version
differences attach as the version matrix grows.

### Drift detection — two complementary tests

**A. Manifest-driven completeness test (`eval/predeclared_documented_test.go`)**

Tests assert every entry in a committed manifest
(`eval/predeclared_manifest.go`) marked `Implemented` is present
in `makeBzlPredeclared`'s output for the version it claims to be
implemented at.

```go
// eval/predeclared_manifest.go
package eval

import "github.com/albertocavalcante/starlark-go-bazel/version"

type ManifestEntry struct {
    Name        string
    Kind        string         // "builtin" | "module" | "constant"
    AddedIn     version.Version
    RemovedIn   version.Version // VLatest+1 if never removed
    Status      string         // "implemented" | "stubbed" | "missing"
    BazelDocsURL string         // Source-of-truth pointer
}

var Manifest = []ManifestEntry{
    {Name: "rule",             Kind: "builtin",  AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#rule"},
    {Name: "aspect",           Kind: "builtin",  AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#aspect"},
    {Name: "repository_rule",  Kind: "builtin",  AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#repository_rule"},
    {Name: "module_extension", Kind: "builtin",  AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#module_extension"},
    {Name: "tag_class",        Kind: "builtin",  AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#tag_class"},
    {Name: "provider",         Kind: "builtin",  AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#provider"},
    {Name: "depset",           Kind: "builtin",  AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#depset"},
    {Name: "Label",            Kind: "builtin",  AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#Label"},
    {Name: "struct",           Kind: "builtin",  AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/builtins/struct"},
    {Name: "attr",             Kind: "module",   AddedIn: version.V7, Status: "implemented",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/attr"},
    {Name: "native",           Kind: "module",   AddedIn: version.V7, Status: "stubbed",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/native"},
    {Name: "True",             Kind: "constant", AddedIn: version.V7, Status: "implemented"},
    {Name: "False",            Kind: "constant", AddedIn: version.V7, Status: "implemented"},
    {Name: "None",             Kind: "constant", AddedIn: version.V7, Status: "implemented"},
}
```

```go
// eval/predeclared_documented_test.go
func TestPredeclared_ImplementedListResolves(t *testing.T) {
    for _, v := range []version.Version{version.V7, version.V8, version.V9, version.VLatest} {
        t.Run(v.String(), func(t *testing.T) {
            dict := makeBzlPredeclared(v) // Post-M1 signature.
            for _, entry := range Manifest {
                if entry.Status != "implemented" && entry.Status != "stubbed" {
                    continue
                }
                if entry.AddedIn > v {
                    continue
                }
                if entry.RemovedIn > 0 && entry.RemovedIn <= v {
                    continue
                }
                if _, ok := dict[entry.Name]; !ok {
                    t.Errorf("version %s: missing predeclared %q (status=%q, added=%s, docs=%s)",
                        v, entry.Name, entry.Status, entry.AddedIn, entry.BazelDocsURL)
                }
            }
        })
    }
}
```

**B. Universe-eval smoke test (`eval/predeclared_evals_test.go`)**

Tests evaluate a synthetic `.bzl` that exercises every documented
builtin and asserts each returns without "undefined: X". Catches
not just "missing key in dict" but also "wired but type-checks
fail" (e.g., `aspect()` is registered but its consumer-side type
assertion still rejects valid input — which is plan 07's domain
but the integration shows up here).

```go
// eval/predeclared_evals_test.go
const universeSrc = `
_provider = provider(fields = ["x"])
_rule = rule(implementation = lambda ctx: None, attrs = {"src": attr.label()})
_aspect = aspect(implementation = lambda target, ctx: None, attrs = {"src": attr.label()})
_repo_rule = repository_rule(implementation = lambda ctx: None, attrs = {"url": attr.string()})
_mod_ext = module_extension(implementation = lambda mctx: None)
_tag = tag_class(attrs = {"name": attr.string()})
_l = Label("//foo:bar")
_d = depset(["a", "b"])
_s = struct(x = 1)
`

func TestPredeclared_UniverseEvalsAtVLatest(t *testing.T) {
    globals, err := evalBzl(t, "universe.bzl", universeSrc, version.VLatest)
    require.NoError(t, err)
    // Each top-level symbol exists with its expected concrete type:
    require.IsType(t, (*types.RuleClass)(nil),             globals["_rule"])
    require.IsType(t, (*builtins.AspectClass)(nil),        globals["_aspect"])
    require.IsType(t, (*types.RepositoryRuleClass)(nil),   globals["_repo_rule"])
    require.IsType(t, (*types.ModuleExtensionClass)(nil),  globals["_mod_ext"])
    require.IsType(t, (*types.TagClass)(nil),              globals["_tag"])
    // ...
}
```

The universe source MUST include every entry in the manifest. A
manifest entry without a corresponding universe-source line is
caught by a third meta-test:

```go
func TestPredeclared_ManifestExercisedByUniverse(t *testing.T) {
    src := universeSrc
    for _, entry := range Manifest {
        if entry.Status != "implemented" {
            continue
        }
        if !strings.Contains(src, entry.Name) {
            t.Errorf("universe.bzl doesn't exercise %q; manifest claims %q is implemented",
                entry.Name, entry.Name)
        }
    }
}
```

### Per-version coverage

Each Version (`V7`, `V8`, `V9`, `VLatest`) gets its own dict via
`Predeclared(Version)` (M1). The smoke test iterates versions:

```go
for _, v := range []version.Version{version.V7, version.V8, version.V9, version.VLatest} {
    t.Run(v.String(), func(t *testing.T) {
        src := buildUniverseSrcFor(v) // gated by AddedIn/RemovedIn
        globals, err := evalBzl(t, "universe.bzl", src, v)
        // ...
    })
}
```

Today (pre-M1), `makeBzlPredeclared` takes no arguments. The
manifest test runs against `VLatest` only; per-version coverage
arrives with M1's signature change.

## Migration alignment with M1

M1 introduces `Predeclared(Version Version)`. The completeness fix
should land in two stages:

1. **M0 stage** — fix the immediate `aspect` omission. Universe-
   eval test runs at the current (no-version) signature. The
   manifest is committed but the version-iteration loop runs over
   the trivial set { current }.

2. **M1 stage** — `Predeclared(Version)` lands. Manifest-driven
   test gains the per-version loop; universe source builder
   constructs version-appropriate `.bzl` based on AddedIn /
   RemovedIn metadata.

The two stages don't conflict — M0 ships the trivial form; M1
extends it.

## TDD plan

### T1. `eval/predeclared_evals_test.go::TestEval_AspectResolvesInBzl`

Smallest possible test for the immediate gap:

```go
func TestEval_AspectResolvesInBzl(t *testing.T) {
    src := `_a = aspect(implementation = lambda t, c: None)`
    _, err := evalBzl(t, "defs.bzl", src)
    require.NoError(t, err)
}
```

State today: RED (`undefined: aspect`). GREEN after the one-line
edit in `makeBzlPredeclared`.

### T2. `eval/predeclared_documented_test.go::TestPredeclared_ImplementedListResolves`

State today: GREEN once the manifest is committed and aspect is
wired (T1's edit). Catches future "added to builtins/ but not
makeBzlPredeclared" regressions.

### T3. `eval/predeclared_evals_test.go::TestPredeclared_UniverseEvalsAtVLatest`

Depends on plan 07's holder-interface fix to actually populate
attrs. Until plan 07 lands, the test asserts the symbols resolve
but the attrs are empty (since aspect()/rule() reject the attr.*
values today). After plan 07 + the wiring fix, the test passes
end-to-end.

### T4. `eval/predeclared_evals_test.go::TestPredeclared_ManifestExercisedByUniverse`

State today: GREEN as soon as universe.bzl is written. Catches
future "added entry to manifest but didn't exercise it in
universe.bzl" drift.

### T5. `eval/predeclared_versions_test.go::TestPredeclared_PerVersionResolves`

Post-M1 only. Iterates `V7..VLatest`, asserts each version's
expected universe evaluates cleanly. Catches "added builtin in V9
but registered for V7" version-leakage bugs.

## Acceptance

- `aspect` registered in `makeBzlPredeclared` (T1 GREEN).
- Manifest committed at `eval/predeclared_manifest.go` with at
  least the 14 entries listed above.
- All four T1–T4 tests passing.
- T5 stub committed with a `t.Skip("requires M1's Predeclared(Version) signature")`
  guard; un-skip during M1.
- assay's vendor refresh: assay's E1c-deferred
  `TestHydrate_Aspect_HydratesAttrs` (currently documenting the
  deferral) flips to active and passes.

## Effort

| Work | Days |
|---|---|
| One-line wire of `aspect` | 0.05 |
| Manifest construction (14 entries) | 0.15 |
| T1 + T2 + T3 + T4 | 0.25 |
| Doc + skipped T5 | 0.05 |

Total: **0.5 days.**

## Risk register additions

- **Risk:** Manifest gets out of sync with reality (entry claims
  Implemented but the builtin isn't ready). **Mitigation:** the
  universe-eval smoke test (T3) attempts to USE the builtin. If
  the builtin is registered but rejects valid input, T3 fails. The
  manifest can't lie unilaterally.

- **Risk:** Universe source becomes large + fragile as new
  builtins land. **Mitigation:** keep it grouped by category
  (builtins, modules, constants) with comments. Total target size
  ~50 lines through M9.

- **Risk:** Per-version source construction (T5) is awkward.
  **Mitigation:** start with hand-curated per-version source
  snippets keyed by Version; refactor to programmatic
  construction only if the snippets diverge significantly.

## Sub-manifests beyond top-level

The top-level `predeclared_manifest.go` covers globals reachable as
bare identifiers in `.bzl` source (`rule`, `aspect`, `attr`, etc.).
Several Bazel surfaces are reached by attribute access on those
globals — `attr.string()`, `native.glob()`, `native.existing_rule()`.
Drift detection at the dotted-name level needs its own manifest
per module.

### `native.*` sub-manifest

`eval/native_manifest.go` (sibling file) lists every member of the
`native` module. The stub implementation lives in
`eval/native_stub.go` (already present per
`newNativeStub()`); the manifest commits the contract about which
members are stubbed vs. implemented faithfully.

```go
// eval/native_manifest.go
var NativeManifest = []ManifestEntry{
    {Name: "native.glob",            Kind: "method",   AddedIn: version.V7, Status: "stubbed",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/native#glob"},
    {Name: "native.existing_rule",   Kind: "method",   AddedIn: version.V7, Status: "stubbed",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/native#existing_rule"},
    {Name: "native.existing_rules",  Kind: "method",   AddedIn: version.V7, Status: "stubbed",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/native#existing_rules"},
    {Name: "native.package_name",    Kind: "method",   AddedIn: version.V7, Status: "stubbed",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/native#package_name"},
    {Name: "native.repository_name", Kind: "method",   AddedIn: version.V7, Status: "stubbed",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/native#repository_name"},
    {Name: "native.package_relative_label", Kind: "method", AddedIn: version.V7, Status: "stubbed",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/native#package_relative_label"},
    {Name: "native.module_name",     Kind: "method",   AddedIn: version.V7, Status: "missing",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/native#module_name"},
    {Name: "native.module_version",  Kind: "method",   AddedIn: version.V7, Status: "missing",
     BazelDocsURL: "https://bazel.build/rules/lib/toplevel/native#module_version"},
}
```

Test:

```go
func TestNative_DocumentedMembersResolve(t *testing.T) {
    // For each Status=stubbed|implemented entry, evaluate:
    //   _x = native.<member>(...)
    // and assert no "no attribute" error.
}
```

`native.module_name` / `native.module_version` are listed as
Status=missing today; M6 lands them, the manifest flips
Status=implemented, the test starts exercising them.

### `attr.*` sub-manifest

Same shape for the `attr` module — `attr.string`, `attr.label`,
`attr.label_list`, `attr.int`, `attr.bool`, `attr.string_list`,
`attr.string_dict`, `attr.label_keyed_string_dict`, `attr.output`,
`attr.output_list`. Most are implemented; the manifest tracks
which.

Test:

```go
func TestAttr_DocumentedConstructorsResolve(t *testing.T) {
    // For each entry, evaluate:
    //   _d = attr.<constructor>(...)  # with sensible args
    // assert the result implements types.AttrDescriptorHolder.
}
```

This test couples directly to plan 07's holder pattern: it
asserts every `attr.*` constructor returns a holder. A regression
in plan 07's migration (e.g., one constructor still returns the
old type) shows up here, not deep in a downstream consumer.

### `ctx.*` and `repository_ctx.*` (deferred)

The `ctx` surface is huge and per-version. A manifest for `ctx.*`
belongs in plan 03's surface matrix and is exercised by M3's
acceptance tests, not by plan 08's drift-detection. Plan 08 stops
at top-level + native.* + attr.* — three places where the eval-
dict registration is the load-bearing step.

## Bazel 8+ builtins (Status=missing/stubbed)

The manifest captures gaps explicitly so plan 03's surface table
and plan 08's manifest can be cross-checked at any time.
Status=missing means "Bazel has it, we don't"; Status=stubbed
means "we accept the call but return a placeholder."

Entries to add to `predeclared_manifest.go`:

```go
{Name: "subrule",           Kind: "builtin", AddedIn: version.V8, Status: "missing",
 BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#subrule"},
{Name: "exec_group",        Kind: "builtin", AddedIn: version.V7, Status: "missing",
 BazelDocsURL: "https://bazel.build/rules/lib/globals/bzl#exec_group"},
{Name: "exec_compatible_with", Kind: "constant", AddedIn: version.V7, Status: "missing",
 // Not strictly a global; this is a kwarg on rules. Listed for visibility.
 BazelDocsURL: "https://bazel.build/reference/be/common-definitions#common.exec_compatible_with"},
{Name: "json",              Kind: "module",  AddedIn: version.V7, Status: "missing",
 BazelDocsURL: "https://bazel.build/rules/lib/toplevel/json"},
{Name: "proto",             Kind: "module",  AddedIn: version.V7, Status: "missing",
 BazelDocsURL: "https://bazel.build/rules/lib/toplevel/proto"},
{Name: "fail",              Kind: "builtin", AddedIn: version.V7, Status: "stubbed",
 // Implemented as starlark's fail (vanilla); behavior matches.
 BazelDocsURL: "https://bazel.build/rules/lib/globals/all#fail"},
{Name: "print",             Kind: "builtin", AddedIn: version.V7, Status: "implemented",
 BazelDocsURL: "https://bazel.build/rules/lib/globals/all#print"},
```

`TestPredeclared_ImplementedListResolves` skips Status=missing
entries (they're documented gaps, not failures). A second test
`TestPredeclared_MissingListIsFailingAtVLatest` confirms that
Status=missing entries DO fail to resolve at VLatest:

```go
func TestPredeclared_MissingListIsFailingAtVLatest(t *testing.T) {
    for _, entry := range Manifest {
        if entry.Status != "missing" {
            continue
        }
        if entry.AddedIn > version.VLatest {
            continue
        }
        src := fmt.Sprintf("_x = %s", entry.Name)
        _, err := evalBzl(t, "missing.bzl", src, version.VLatest)
        if err == nil {
            t.Errorf("Manifest says %q is missing, but it resolved (manifest stale?)",
                entry.Name)
        }
    }
}
```

When `subrule()` gets implemented in a future milestone, the
implementer flips the manifest entry to Status=implemented AND
adds it to the universe source. The test family catches forgetting
either step.

## Coordination with plan 03

Plan 03 (builtins-surface) is the narrative description of which
builtins exist and at what tier. Plan 08's manifest is the
executable form of the same information.

Going forward:

- Plan 03 stays the prose source-of-truth ("subrule is M8+ scope
  because it's bazel 8 only and canopy doesn't need it yet").
- Plan 08's manifest stays the code source-of-truth ("at runtime,
  here's what's wired").
- A CI-time cross-check confirms every Status=implemented entry in
  the manifest is mentioned in plan 03's tables, and vice-versa.
  Implementation deferred — for now, code-review enforces it.

## Coordination with plan 07

Plan 07 (AttrDescriptor unification) and this plan together form
M0. They're independent in scope (one is consumer-side type
assertions; one is registration in the eval dict), but their
TESTS depend on each other:

- Plan 07's T1 (`TestAspect_AttrsRoundTrip`) requires `aspect` to
  be reachable from .bzl, which is plan 08's responsibility.
- Plan 08's T3 (`TestPredeclared_UniverseEvalsAtVLatest`) requires
  `aspect(attrs = ...)` to type-check correctly, which is plan
  07's responsibility.

Land both within M0. See plan 07's "Implementation order within
M0" for the step-by-step sequence.
