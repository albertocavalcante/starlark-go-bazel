# 02 — Pre-M1 cleanup and publish

**Status:** drafted 2026-05-31. Open. Targets ~3 h of focused work.

**Scope.** The WIP on disk at `/Volumes/T9/dev/ws/starlark-go-bazel/`
spans M1 through M6 of the
[`01-bazel-builtins-emulation`](01-bazel-builtins-emulation/README.md)
dossier — `bzl.Options` surface, `builtins/`, `ctx/`, `stub/`,
`taint/`, `eval/invoke.go`, `version/`, and `bzl.EvalFromAST`. 30+
new files + 3 modified across the repo, all uncommitted. Plus a 10-doc
plan dossier under `docs/plans/01-…/`.

This plan covers what has to happen before that work is safe to push:

1. A grumpy-review reflection on **dead config vs forward-compat
   scaffold** (the part of the on-disk surface that publishes fields
   without behavior — is each field forward-compat, or accidentally
   shipping a lie?).
2. The **A1–A8 fix list** with TDD specs.
3. The **commit-split plan** (turn the one-big-WIP into a
   bisectable history per milestone before pushing).
4. The **downstream handoff** (assay drops its last `replace`;
   canopy verifies the v0.2 surface).

**Origin.** The reflection started from a code review of the WIP
that concluded "this is dead config" too quickly. Re-reading
`01-…/02-architecture-and-versioning.md` made clear the surface is
deliberately published at M1 so consumers pin once and M5/M6 fill
in behavior. The right critique is **godoc honesty**, not field
removal. This plan codifies that distinction.

## 0. Executive summary

| Phase | Items | Effort |
|---|---|---|
| A — code cleanup | A1 delete dead helper, A2 add TDD tests, A3 honest godoc, A4 known-limitation README, A5 alias fix, A6 Version enum, A7 fresh-slice DefaultPlatforms, A8 typed accessor | ~2 h |
| B — commit-split | One commit per milestone (M1 / M2 / M3 / M4 / M5 / M6 / M7 + docs) before push | ~40 min |
| C — push + assay pin | Push to origin, drop assay's last `replace`, pin pseudo-version, verify | ~20 min |
| D — canopy verification | Refresh canopy's vendor against the new assay; confirm the v0.2 fields render | ~30 min |

**Total:** ~3.5 h.

## 1. Reflection: dead config vs forward-compat scaffold

The initial grumpy-review verdict was "four fields shipped in
`bzl.Options` with no behavior — gaslighting the user." Re-reading
the dossier corrects that. From
[`01-…/02-architecture-and-versioning.md`](01-bazel-builtins-emulation/02-architecture-and-versioning.md):

> bzl.Options grows additively — new fields (Version, Mode,
> PredeclaredBzl, PredeclaredBuild, CaptureSinks) with zero-value
> backward compatibility. No breaking changes to existing
> consumers.

That is a deliberate choice: ship the surface at M1 so consumers
pin once and M5/M6 fill in behavior without a struct revision.
Adding fields later would require a coordination round with assay,
canopy, and any other consumer. Adding them up front with
zero-value-safe defaults closes that gap.

The actual sin is therefore **godoc honesty**, not field
existence. Per-field verdict:

| Field | Forward-compat scaffold? | Verdict |
|---|---|---|
| `bzl.Options.Mode` | Yes (M5 wires the routing) | Godoc currently *describes* `ModeLenient stubs unresolvable external loads...` as if it works. It doesn't. Add a `SCAFFOLD:` preamble line saying behavior wires in M5. |
| `bzl.Options.FeatureFlags` | Yes (M6 wires the table) | Godoc already admits "future M6 work populates" — honestly labeled. Leave. |
| `bzl.Options.CaptureSinks` | Yes (M5 wires the dispatch) | Godoc admits "M1 surface accepts the pointer." Honest. Leave. |
| `bzl.Options.effectiveMode()` | No — unexported helper, unused | Adding it in M5 is not a breaking change (unexported). **Delete now.** |
| `eval/invoke.go::InvokeOptions.Timeout` | Half — comment says "Not enforced today" | Honest scaffold, but borderline because the field *implies* an effect. Leave for now; revisit in M5 when context-deadline is wired. |
| `version.HasFeature`, `Deltas`, `BazelFeaturesValue` | Yes (M6 populates the table) | Internal-only — no public consumers in tree yet. Add a per-package `SCAFFOLD:` doc-line. |

So the real surgical move is: delete `effectiveMode`, label every
other scaffold field with a `SCAFFOLD: routing lands in M<N>`
preamble, and leave the publishing decision intact.

## 2. A1–A8 — code cleanup

Each step is small, TDD where applicable, and orthogonal to the
others. Order is the suggested ordering for a clean diff stream.

### A1. Label `Options.effectiveMode()` as scaffold (reflection-corrected)

File: `bzl/options.go:84-…`.

**Original verdict (revised).** I initially proposed deleting
`effectiveMode` as actually-dead unexported scaffold. Reflection on
the user's "is dead code really dead?" pushback corrected this:
`effectiveMode` is the documented *pinning* of the auto-promotion
contract (`LenientLoad → ModeLenient` when Mode is zero). The unit
tests in `bzl/options_test.go` (`TestOptions_ZeroValueDefaults`,
`TestOptions_LenientLoadAutoPromotes`, `TestOptions_ExplicitModeWins`)
freeze the contract that M5 will honor when it wires Mode into the
loader switch.

Deleting `effectiveMode` now would either (a) require re-deriving
the auto-promotion logic from scratch when M5 lands or (b) lose the
test coverage that proves the contract holds.

**Action.** Keep the helper. Add a `SCAFFOLD:` preamble to its
godoc pointing at this plan §1 and the test suite that pins the
contract.

**Risk:** zero.

### A2. Add TDD tests for `taint.FlattenURLs` and `stub.ScanLoads`

These are the two subtle, currently-untested functions in the WIP.
`stub.LoaderFor` is already covered by `stub/permissive_test.go:213…`;
the gap is `ScanLoads` (the load-statement scanner that pairs with
`LoaderFor`) and `FlattenURLs` (the URL normalizer).

#### A2.1 `taint/url_test.go` (new file, external test package)

External `package taint_test` so the test can import `stub` to
exercise the `Permissive`-detection branch without an import cycle.

| Test | Input | Expected output |
|---|---|---|
| `TestFlattenURLs_Nil` | `nil` | `(nil, false)` |
| `TestFlattenURLs_SingleString` | `starlark.String("https://x/v.tar.gz")` | `(["https://x/v.tar.gz"], false)` |
| `TestFlattenURLs_SingleStringWithMarker` | `starlark.String("https://x/<permissive>/v.tar.gz")` | `([..."<permissive>"...], true)` |
| `TestFlattenURLs_List` | `starlark.NewList(["a", "b"])` | `(["a", "b"], false)` |
| `TestFlattenURLs_ListWithMarker` | `["a", "https://<permissive>/x", "c"]` | 3 entries; `tainted=true` |
| `TestFlattenURLs_PermissiveValue` | `stub.Shared` | `(["<unresolved>"], true)` |
| `TestFlattenURLs_ListContainingPermissive` | `["a", stub.Shared, "c"]` | 3 entries; `["a", "<unresolved>", "c"]`; `tainted=true` |
| `TestFlattenURLs_NonIterable` | `starlark.MakeInt(42)` | `(nil, false)` |

These pin the existing behavior. RED → run against current impl,
expect GREEN. The tests become the regression gate that catches
the future M5 refactors when capture sinks get wired.

#### A2.2 `stub/loader_test.go::TestScanLoads_*` (extend existing file)

The current `stub/permissive_test.go` covers `LoaderFor`; the new
suite below covers `ScanLoads` specifically.

| Test | Input | Expected output |
|---|---|---|
| `TestScanLoads_SingleLoadOneSymbol` | `load("@x//:y.bzl", "foo")` | `{"@x//:y.bzl": ["foo"]}` |
| `TestScanLoads_AliasedLoadStoresFromName` | `load("@x//:y.bzl", baz_local = "baz_remote")` | `{"@x//:y.bzl": ["baz_remote"]}` (pins the "we store From, not To" decision) |
| `TestScanLoads_MixedPositionalAndAliased` | `load("@x//:y.bzl", "foo", baz_local = "baz_remote")` | `{"@x//:y.bzl": ["foo", "baz_remote"]}` |
| `TestScanLoads_MultipleLoadsAccumulate` | two `load()` to same module | accumulated list |
| `TestScanLoads_NoLoads` | source with only a def | empty map (non-nil) |
| `TestScanLoads_NilFile` | `nil` | empty map (non-nil) |

**Risk:** low. Tests pin existing behavior.

**Effort:** ~45 min including running.

### A3. Honest-godoc pass on the scaffold fields

Touched: `bzl/options.go`, `bzl/mode.go`, `version/version.go`,
`version/features.go`, `version/deltas.go`, `version/bazel_features.go`.

Pattern: each scaffold field gets a one-line preamble at the top
of its doc-comment.

```
// SCAFFOLD: routing wires in M5 (see plan 02-…/01-…/05-spike-promotion-and-testing.md).
// Today the field is read but values other than the zero behave
// identically to the zero. Pin against this surface knowing
// behavior is reserved.
//
// Mode selects the strictness of Bazel Starlark evaluation. (…)
```

Specific touches:

- `bzl/options.go::Mode` field — add the preamble above.
- `bzl/mode.go` per-constant docs — soften "ModeLenient stubs
  unresolvable external loads..." to "When M5 ships, ModeLenient
  will stub...".
- `version/version.go` package doc — already admits "Per-version
  feature tables ... are populated in M6"; tighten the language.
- `version/features.go::HasFeature` doc — add "SCAFFOLD: table is
  empty until M6 populates per-Version defaults."
- `version/deltas.go::Deltas` — add `SCAFFOLD:` preamble; today
  every field is the zero value regardless of Version.
- `version/bazel_features.go::BazelFeaturesValue` — add
  `SCAFFOLD:` preamble; today the returned struct has empty
  values.

**Risk:** zero. Documentation only.

**Effort:** ~15 min.

### A4. Promote `Hash(Permissive) → error` into user-facing docs

`stub/permissive.go::Hash()` returns an error so dict-keying a
`Permissive` aborts the (per-platform) fork. Plan
[`01-…/06`](01-bazel-builtins-emulation/06-risks-open-questions-and-milestones.md)
§Q5 explicitly considered this and chose to keep it unhashable.
Acceptable trade-off — but the limitation is currently buried 5
docs deep.

**Add:**

1. To `stub/permissive.go` package godoc, a "Known limitation"
   section: dict-key abort, ForkError surfaces, see plan 06 §Q5.
2. To the repo `README.md`, a "Known limitations" section if not
   present; otherwise extend.

**Risk:** zero.

**Effort:** ~10 min.

### A5. Use the `bzl.LoadFunc` type alias (or delete it)

`bzl/options.go:11` defines `type LoadFunc = func(*starlark.Thread,
string) (starlark.StringDict, error)`. `bzl/options.go::Options.LoadResolver`
on line 82 repeats the same signature inline.

**Pick:** use the alias.

```
LoadResolver LoadFunc
```

So the named type is the public-facing surface.

**Risk:** zero (alias means identical underlying type).

**Effort:** ~5 min.

### A6. `version.Version` enum ordering

Current:

```go
const (
    VLatest Version = iota  // = 0
    V7                       // = 1
    V8                       // = 2
    V9                       // = 3
)
```

A consumer writing `if opts.Version < version.V8` hits "VLatest is
less than V7," which is confusing.

**Recommend:** reorder to make iota match the major number's
ascending order.

```go
const (
    _ Version = iota // skip 0 to keep VLatest as a sentinel
    V7
    V8
    V9
)

const VLatest = V9
```

Plus an explicit test: `TestVersion_OrderingComparable` pinning
`V7 < V8 < V9` and `VLatest == V9`.

**Risk:** moderate IF an external consumer is pinning to the
iota *values* (not the symbolic names). No public consumer exists
yet (zero hits in tree). Confirm before changing.

**Effort:** ~15 min.

### A7. `taint.DefaultPlatforms` from `var` to `func`

`taint/taint.go::DefaultPlatforms` is currently a package-level
`var` slice. A consumer that does `taint.DefaultPlatforms = append(
taint.DefaultPlatforms, custom)` mutates global state for the next
test.

Replace with:

```go
func DefaultPlatforms() []Platform {
    return []Platform{
        {OS: "linux", Arch: "amd64"},
        // …
    }
}
```

Plus a regression test
`TestDefaultPlatforms_ReturnsFreshSlice` confirming two calls return
distinct backing arrays.

**Risk:** breaking for any caller using `DefaultPlatforms` as a
slice variable. Audit before changing — no internal users today.

**Effort:** ~10 min.

### A8. Typed accessor for `RuleInstantiation.Rule`

`taint/taint.go::RuleInstantiation.Rule` is typed `starlark.Value`
to keep the `taint` package free of a `types` dependency
(comment: "held as starlark.Value to keep taint independent of
types"). Consumers reach the typed rule via manual type assertion.

Add a helper in `types/` (or as a free function in `taint/` that
*returns* a `starlark.Value` and leaves the assertion to the
consumer's package — preserves the no-import-cycle property):

```go
// In types/repository_rule_class.go or a new types/instantiation.go:

// FromInstantiation type-asserts a taint.RuleInstantiation.Rule
// to *RepositoryRuleClass. Returns nil if the assertion fails (the
// taint package stores the value as starlark.Value to avoid an
// import cycle).
func FromInstantiation(r taint.RuleInstantiation) *RepositoryRuleClass {
    rc, _ := r.Rule.(*RepositoryRuleClass)
    return rc
}
```

Plus a regression test that round-trips a captured instantiation
through the accessor.

**Risk:** low.

**Effort:** ~15 min.

## 3. B — commit-split before pushing

Split the 30+ files into milestone-shaped commits. Each commit
must be independently buildable and pass `go test ./...` at HEAD.

Suggested order:

| # | Commit | Files |
|---|---|---|
| B1 | M1: `bzl.Options` surface | `bzl/options.go`, `bzl/mode.go`, `bzl.LoadFunc` + A3 godoc + A5 alias fix |
| B2 | M2: `repository_rule` + `module_extension` + `tag_class` builtins | `builtins/{repository_rule,module_extension,tag_class}.go` + their tests |
| B3 | M3: `ctx` packages | `ctx/{module_ctx,repository_ctx}.go` + their tests |
| B4 | M4: `stub/` package | `stub/permissive.go`, `stub/loader.go` + tests + A2.2 ScanLoads suite |
| B5 | M5: `taint/` package + `eval/invoke.go` | `taint/taint.go`, `taint/url.go` + A2.1 FlattenURLs suite + A7 fresh-slice + A8 typed accessor; `eval/invoke.go`, `eval/{from_ast,invoke,bench}_test.go`; types/*  |
| B6 | M6: `version/` package | `version/{version,features,deltas,bazel_features}.go` + tests + A6 ordering |
| B7 | M7: `bzl/bzl.go` (EvalFromAST + exportNames) | `bzl/bzl.go` + `bzl/options_test.go` |
| B8 | docs | `docs/plans/01-bazel-builtins-emulation/` (10 files) + `docs/plans/02-pre-m1-cleanup-and-publish.md` (this file) |

Verification at each commit:

```sh
go test ./...     # must be green
GOOS=js GOARCH=wasm go build ./wasm/...   # wasm still builds
```

**Risk note.** B5 is the biggest commit (taint + eval/invoke + 4
types/* files) because they're tightly coupled. If it gets unwieldy,
split into B5a (types/) + B5b (taint + invoke). Discretion at the
time.

## 4. C — push + assay pin

After B8 lands on origin:

```sh
cd /Volumes/T9/dev/ws/assay
go get github.com/albertocavalcante/starlark-go-bazel@<sha-from-B8>
# Remove from go.mod:
#   replace github.com/albertocavalcante/starlark-go-bazel => ../starlark-go-bazel
go mod tidy
go mod vendor
just check
REFS_DIR=$HOME/dev/refs go test -run TestCorpus .
git add go.mod go.sum vendor/modules.txt vendor/github.com/albertocavalcante/starlark-go-bazel
git commit -m "build: pin starlark-go-bazel to pseudo-version <sha>"
git push origin main
```

When CI returns green, Phase 0E (per `assay/docs/lib-carveout-plan.md`
§13.2) is officially closed.

## 5. D — canopy verification

After C lands:

```sh
cd /Volumes/T9/dev/ws/canopy
# Refresh assay dep
go get github.com/albertocavalcante/assay@<sha-of-assay-pin-commit>
go mod tidy
go mod vendor
just check
# Run analysis pipeline against rules_python (high tag_classes yield)
# and confirm renders include the new D-E-F fields
```

Expected output: canopy renders new fields cleanly OR surfaces a
list of "renderer missed field X." Either outcome validates v0.2;
the latter just becomes a canopy-side punch list.

Owner: canopy maintenance. Outside this plan's strict scope; tracked
here for end-to-end visibility.

## 6. Risk register

| ID | Risk | Mitigation |
|---|---|---|
| R1 | A6 reorder breaks an unknown external consumer pinning to iota values | grep across `~/dev/ws/` confirms no public reads of `version.V7`/etc. as numeric values; revert if found |
| R2 | Commit-split takes longer than estimated because cross-package dependencies aren't strictly milestone-shaped (e.g., M2 builtins reference types/ structs) | Land types/* in B5 alongside taint/; if M2 builtins need them earlier, split B5 to land types/ in a pre-B2 commit |
| R3 | Canopy's renderer is itself out-of-date on the D-E-F fields | Surface as canopy work, not assay; verification reveals it cleanly |
| R4 | starlark-go-bazel push reveals additional CI breakage on origin (test that passes locally fails on GH Actions due to env diff) | Test against `go test -race ./...` before B8; budget one fix-forward cycle |
| R5 | A2 tests reveal genuine bugs in `FlattenURLs` or `ScanLoads` (not just no-coverage) | Treat as TDD: fix the bug under the new test, land in the same commit as the test |

## 7. Decision log

| # | Decision | Why |
|---|---|---|
| 1 | Keep scaffold fields; fix godoc, not field existence | Plan §02 mandates publish-once-pin-once; removing fields forces a coordination round with consumers when M5/M6 ships |
| 2 | Delete `effectiveMode` | Unexported, unused; re-adding in M5 is not a breaking change |
| 3 | Use `bzl.LoadFunc` alias over inline signature | Type names belong in public docs; aliases are the cheapest way to surface a documented name |
| 4 | Reorder `version.Version` enum to put V7 < V8 < V9 | `if v < V8` should behave as a Bazel engineer would expect; current ordering trips |
| 5 | Replace `taint.DefaultPlatforms` var with func | Mutable global slice leaks across tests; pure function is the standard idiom |
| 6 | Single commit per milestone (B1–B7) + docs commit (B8) | Matches the existing plan's milestone language; bisectable; small enough to review per-commit |
| 7 | Phase 0E (assay's last replace) lands in `assay`, not in this repo | Closes the cross-repo loop where it belongs |

## 8. Open questions

1. **Q1: A6 ordering — confirm no external consumer pins to iota
   numeric values?** Grep proved no internal consumers; external
   visibility is the question. Recommend: proceed.
2. **Q2: Should B5 split (types/ vs taint+invoke)?** Depends on
   commit size after A8 lands. Decide live.
3. **Q3: Should `taint.Marker` be the literal `"<permissive>"`
   or a less likely-to-collide sentinel like `"\x00<permissive>\x00"`?**
   Plan §06 Q6 picked the printable form for ergonomics. Acceptable;
   leave for now.

## 9. After this plan ships

Three things to surface:

1. **canopy verification result** — if the renderer is missing
   fields, file a canopy-side ticket. Don't sprawl this plan.
2. **First external consumer adoption** — scip-bazel or
   compat-analyzer per the M9 line in the 01-dossier. Out of scope
   here.
3. **starlark-go-bazel M0 (attr descriptor unification)** — open
   per [`01-…/07-attr-descriptor-unification.md`](01-bazel-builtins-emulation/07-attr-descriptor-unification.md).
   Track separately.
