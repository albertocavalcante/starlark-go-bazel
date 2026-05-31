# 06 — Risks, open questions, milestones

## Risks

### Maintenance burden

- **Annual Bazel releases.** Each major Bazel release adds builtins,
  changes semantics, deprecates features. Tracking is real ongoing
  work. Mitigation: scope to surfaces real consumers exercise — don't
  pre-implement what no one uses.
- **Sole-maintainer concentration.** Alberto is the maintainer.
  If consumers (canopy, scip-bazel, future) need urgent fixes and
  the maintainer is busy, work blocks. Mitigation: keep the spike
  pattern reusable — canopy can monkey-patch into stub/ if needed.
- **Library promises stability; spike was experimental.** Publishing
  `Mode`, `Version`, `Permissive` etc. creates implied compatibility
  obligations. Mitigation: tag a v0.x series throughout this plan;
  only promise v1.0 stability when the surface settles.

### API stability

- **The spike's APIs are 1 day old.** Some namings (`StringAttrs`,
  `InvokeResult.ForkErrors`) are first-pass. Will probably need to
  iterate during M7 promotion. Mitigation: don't lock v1.0 surface
  until a real second consumer adopts (scip-bazel).
- **Permissive's marker string is a public contract.** Once
  `taint.Marker = "<permissive>"` ships, third parties will rely
  on the exact string. Mitigation: `taint.Has()` helper makes the
  marker an implementation detail consumers don't need to know.
  Document the helper as the supported entry point.
- **`bazel_features` synthetic vs. real divergence.** Our
  `version.HasFeature()` table may drift from the upstream
  `bazel-features-bzl` over time. Mitigation: CI diff check against
  a pinned version of the upstream module data file.

### Implementation gaps already known

- **attr descriptor stubbing** (existing) — `types/rule_class.go`'s
  basic descriptor blocks attr Type/Default/Doc/Mandatory extraction.
  Documented in `assay/interp/LIMITATIONS.md`. **Not in this plan.**
  But blocks the compat-analyzer's ability to do "this rule's `srcs`
  attr now requires `mandatory=True`" delta detection. Separate plan
  needed before plan 05 ships.
- **AttrDescriptor bifurcation** — `builtins.AttrDescriptor` (legacy,
  consumed by `aspect()`/`rule()` type assertions) vs
  `types.AttrDescriptor` (canonical, produced by `attr.*`). Real
  `.bzl` source like `aspect(attrs = {"src": attr.label()})` fails
  with "aspect: attrs values must be attr objects." Discovered
  empirically during assay's E1c work (`assay/docs/registry-surface-plan.md`).
  **Addressed in M0 via plan 07.**
- **`aspect()` not in `makeBzlPredeclared`** — `builtins.Aspect`
  exists but isn't registered in the eval-time predeclared dict.
  Callers must inject it via `bzl.Options.PredeclaredBzl`. M2's
  new builtins (`repository_rule`, `module_extension`, `tag_class`)
  ARE wired, but no drift-detection prevents future omissions.
  **Addressed in M0 via plan 08.**
- **Symbolic both-branch eval.** `if perm == X: A() else: B()` runs
  only B. Real impact: missed URLs in conditional branches that
  depend on opaque ops. Mitigation: runtime interception covers it
  for the airgap case; document as known.
- **Permissive as dict key.** `Hash()` errors. Real impact: rules
  that use Permissive-derived keys abort the fork. Mitigation:
  ForkError surfaces to the caller.
- **WORKSPACE evaluation faithfulness.** Bazel 6/7 evaluation
  semantics differ from bzlmod era. We support best-effort; we
  don't promise correctness for WORKSPACE-only rulesets. Be explicit
  in README.

### External dependencies and supply chain

- **`go.starlark.net` upstream changes.** Library updates to the
  Starlark frontend can change resolution / Compare / Binary
  semantics. Mitigation: vendored at fixed version; bumps tested
  against full suite.
- **`bazel-features-bzl` upstream changes.** Our synthetic depends
  on knowing what features map to what versions. Mitigation: pinned
  version + CI diff.
- **Real-corpus modules in `~/dev/refs`.** Developer-machine-only.
  Acceptable since they're for analysis verification, not CI.

### Risk we won't discover until later

- **Permissive's "always-truth-true" affordance** may bite real
  rulesets that do `if not value:` and expect the false branch when
  loaded-but-unset. Adjust based on real corpus.
- **Some rulesets may rely on `Hash(Permissive)`** in ways the
  spike didn't see. ForkError surfaces it; impact unclear until
  rules_python / rules_jvm_external get exercised in M7.
- **`ctx.execute` stdout-derived branching is rare in the
  spike but common in toolchain detection** (rules_python's
  `python_register_toolchains` for instance). Will need more
  per-fork coverage data once M7 runs full corpus.

## Open questions (decide as the plan executes)

### M0 era

- **Q0a.** Which descriptor type stays — `types.AttrDescriptor` or
  `builtins.AttrDescriptor`? **Recommend** `types.AttrDescriptor`
  (richer field set, public access, downstream consumers already
  read it). Delete `builtins.AttrDescriptor` in the same milestone.
  See plan 07.
- **Q0b.** Should consumers type-assert against `*types.AttrDescriptor`
  directly or use the `types.AttrDescriptorHolder` interface?
  **Recommend** the interface — `eval/evaluator.go::attrDescriptorValue`
  is already a `Holder`, and the interface decouples the producer
  wrapper (a `starlark.Value`) from the descriptor type it carries.
  See plan 07.
- **Q0c.** Hand-maintain the predeclared manifest or generate it
  from a `//go:build manifest`-style reflective walk? **Recommend**
  hand-maintain — the manifest carries per-version + per-status +
  Bazel-docs-URL metadata that a reflective walk can't synthesize.
  See plan 08.

### M1-M2 era

- **Q1.** Should `bzl.LenientLoad: true` be hard-deprecated or
  silently auto-promote to `ModeLenient`? **Recommend** soft-deprecate
  (auto-promote + once-per-process log).
- **Q2.** `Mode` constants on `bzl` package or in a `bzl/mode`
  subpackage? **Recommend** top-level for ergonomics.

### M3 era

- **Q3.** `RepositoryCtx.Attr(unknown)` — return empty string (spike)
  or error or Permissive? **Recommend** empty string for backward
  compat with spike, with a `StrictUnknownAttrs` Option flag for
  consumers who want errors.
- **Q4.** Where does `ModuleSpec` live — `ctx/`, `eval/`, or
  `taint/`? **Recommend** `ctx/` since it configures `module_ctx`.

### M4-M5 era

- **Q5.** Should `Permissive.Hash()` return a sentinel hash (e.g.,
  `0xdeadbeef`) to let `dict[Permissive]` not abort? Trade-off: all
  Permissives hash-equal-collide → dict semantics get weird (look up
  one Permissive, find another's value). **Recommend** keep
  unhashable; add ForkError path; revisit if real consumer demands.
- **Q6.** Should `taint.Marker` be `"<permissive>"` or
  `"<bazel-go-permissive>"` (namespaced)? **Recommend** keep
  `"<permissive>"` — short, recognizable; namespacing has cost
  without benefit until a name collision surfaces.

### M6 era

- **Q7.** Vendor `bazel-features-bzl` source or curate per-Version
  flags by hand? **Recommend** hand-curate with CI diff against the
  vendored source as a sanity check.
- **Q8.** `VLatest` should track which version? **Recommend** Bazel
  9.1 at plan start; bump to 9.x as releases land.
- **Q9.** Versioning for the library itself — `v0.x` during this
  plan, then `v1.0` post-M9? **Recommend** yes; v1.0 implies "API
  is stable; breaking changes require new major."

### M7-M8 era

- **Q10.** When (if ever) do we move from `assay/interp/spike/` →
  `assay/interp/external/` vs. directly to upstream consumption?
  **Recommend** direct upstream consumption (skip intermediate); the
  spike serves as proof-of-correctness and the upstream form is the
  shipping form.
- **Q11.** Should `canopy/internal/external/` exist as a layer
  between assay and ctx, or does canopy consume assay directly?
  **Recommend** thin canopy-side layer for ingest-specific
  concerns (caching, schema mapping); see plan 12.

### M9 era

- **Q12.** Donate to bazelbuild? Stay independent under
  albertocavalcante/*? **Recommend** stay independent through v1.0;
  revisit if Bazel team expresses interest. v1.0 with real-corpus
  validation is the time to have that conversation.
- **Q13.** SPDX license — currently MIT. Compatible with everything;
  no change.

## Milestone sequence

Each milestone gets:
- **Scope** — what code changes.
- **Acceptance** — what tests gate it.
- **Effort** — order-of-magnitude estimate, not a commitment. A
  "3 day" milestone might be 1.5–6 days depending on what surfaces
  during implementation. Treat the numbers as ranking ("M3 is bigger
  than M4") rather than as deadlines.

### M0 — AttrDescriptor unification + predeclared dict completeness

**Scope:** Plans 07 + 08. Migrate `builtins/aspect.go` + `builtins/rule.go`
from `*builtins.AttrDescriptor` to `types.AttrDescriptorHolder` +
`.Descriptor() *types.AttrDescriptor`. Delete `builtins.AttrDescriptor`.
Wire `aspect` into `eval/evaluator.go::makeBzlPredeclared`. Commit
`eval/predeclared_manifest.go` with current builtin set + per-entry
metadata (AddedIn, Status, BazelDocsURL). Add the manifest-driven
completeness test + the universe-eval smoke test + the manifest-
exercised-by-universe meta-test.

**Acceptance:**
- `TestAspect_AttrsRoundTrip` GREEN (plan 07 T1).
- `TestRule_AttrsStillWorks` GREEN with no regression (plan 07 T2).
- `TestEval_AspectResolvesInBzl` GREEN (plan 08 T1).
- `TestPredeclared_ImplementedListResolves` GREEN at the
  no-Version signature (plan 08 T2; per-Version loop awaits M1).
- `TestPredeclared_UniverseEvalsAtVLatest` GREEN end-to-end —
  every documented builtin reachable from .bzl AND produces a
  correctly-typed value (plan 08 T3).
- `TestPredeclared_ManifestExercisedByUniverse` GREEN (plan 08 T4).
- `builtins.AttrDescriptor` no longer exported (plan 07 T5;
  catches re-introduction).
- assay's vendor refresh reactivates its E1c-deferred
  `TestHydrate_Aspect_HydratesAttrs` (currently documenting the
  upstream-block deferral) and it passes. Plan 09 details the
  vendor refresh + six-step reactivation procedure.
- WASM build green.
- `go vet ./...` + `golangci-lint run` clean.

**Effort:** 2 days. (Plan 07: 1.5; plan 08: 0.5.) Plan 09's
downstream handoff is an additional ~3 hours of assay-side work
that runs AFTER M0 ships and is tracked in assay's own
plan/changelog, not counted in this milestone's effort.

### M1 — `bzl.Options` surface

**Scope:** Add `PredeclaredBzl`, `PredeclaredBuild`, `Version`,
`Mode`, `CaptureSinks` to `bzl.Options`. Wire through to existing
`eval.Evaluator.Options`. Backward-compat the `LenientLoad` bool.

When `Predeclared(Version)` signature lands, un-skip plan 08's
`TestPredeclared_PerVersionResolves` and extend
`TestPredeclared_ImplementedListResolves` to iterate all four
versions instead of just `VLatest`.

**Acceptance:**
- Existing `bzl_test.go` and `lenient_load_test.go` pass unchanged.
- New `bzl_options_test.go` validates the new fields propagate.
- Plan 08's previously-skipped `TestPredeclared_PerVersionResolves`
  is un-skipped and passes.
- WASM build green.

**Effort:** 2 days.

### M2 — `repository_rule` + `module_extension` + `tag_class` builtins

**Scope:** `builtins/repository_rule.go`, `builtins/module_extension.go`,
`builtins/tag_class.go`. `types.RepositoryRuleClass`,
`types.ModuleExtensionClass`. `Predeclared(Version)` extends to include
these per the version's surface.

**Acceptance:**
- New unit tests pass for each builtin (kwargs accepted, value
  returned has correct interface).
- `assay/interp/LIMITATIONS.md`'s
  `TestHydrate_RepositoryRule_StillUntouchedUntilUpstreamSupport`
  flips from "pinned skip" to "supported" — assay's interp picks up
  the new builtin via the existing `bzl.Interpreter` path.
- **Holder-pattern adoption (per plan 07).** Any attrs-bearing
  builtin added in M2 stores its `attrs` kwarg's values in a way
  that exposes `types.AttrDescriptorHolder`. The plan 07 T3
  (`TestTagClass_AttrsHolderInterface`) and T4
  (`TestModuleExtension_TagClassAttrs`) MUST be GREEN at M2's
  acceptance — they fail RED in M0 (correctly; the underlying
  builtins don't exist yet) and flip GREEN when M2's
  implementations land.
- **No `*builtins.AttrDescriptor` reintroduction.** Per plan 07 T5,
  `builtins.AttrDescriptor` was deleted at M0. M2 implementations
  must not re-introduce it; consumer-side type assertions go
  through `types.AttrDescriptorHolder.Descriptor()`. A grep audit
  during code review confirms.
- **Manifest update.** `eval/predeclared_manifest.go` entries for
  `repository_rule`, `module_extension`, `tag_class` flip from
  Status="implemented at registration only" (the M0 starting state
  for the wiring) to Status="implemented" (full end-to-end).
  Plan 08's manifest-exercised-by-universe test
  (`TestPredeclared_ManifestExercisedByUniverse`) catches a
  missed manifest update.

**Effort:** 3 days.

### M3 — `RepositoryCtx` + `ModuleCtx` + `bazel_module`

**Scope:** `ctx/repository_ctx.go`, `ctx/module_ctx.go`. The Attr
table, sub-types (`RepositoryOs`, `RepositoryAttr`), helpers, and
the synthetic ctx constructors.

**Acceptance:**
- New unit tests pass for each attribute.
- Spike's repository_ctx + module_ctx tests port and pass.

**Effort:** 4 days.

### M4 — `stub/` package

**Scope:** `stub/permissive.go`, `stub/loader.go`. The Permissive
value type, the marker constant export (lives in `taint/`, imported
by stub), the loader function.

**Acceptance:**
- Spike's `permissive_test.go` ports.
- Fuzz test for `Permissive.Binary` and `Permissive.CompareSameType`.

**Effort:** 2 days.

### M5 — `taint/` package + Mode=Analysis

**Scope:** `taint/taint.go`, `taint/sink.go`, `taint/fork.go`,
`taint/marker_test.go`. Wire `Mode=Analysis` to activate the capture
sinks. `eval.InvokeRepositoryRule` and `eval.InvokeModuleExtension`
public functions.

**Acceptance:**
- All 23 spike tests pass via the new API.
- `dedupe()` correctness pinned with the platform-iteration-ordering
  bug fix from the spike code review.

**Effort:** 3 days.

### M6 — `version/` package + bazel_features

**Scope:** `version/version.go`, `version/features.go`,
`version/deltas.go`. The synthetic `bazel_features` module that the
loader serves when a `.bzl` does `load("@bazel_features//:features.bzl",
...)`.

**Acceptance:**
- Version-cross tests pass (`if hasattr(ctx, "repo_metadata"):` gated
  by Version).
- `version/features_test.go::TestFeaturesMatchUpstreamForV9` diffs
  our table against the vendored `bazel-features-bzl` data and
  passes for V9.

**Effort:** 3 days.

### M7 — spike promotion

**Scope:** Move every spike file to its destination per the migration
map. Delete spike's redundant builtins (label/select/depset/struct/
provider/aspect/rule — use upstream's). Update test files to use the
new API.

**Acceptance:**
- `go test ./...` in starlark-go-bazel passes.
- All 23 spike tests pass (`go test -tags realcorpus
  ./eval/...` covers the rules_go real-corpus cases).
- `assay/interp/spike/` directory marked deprecated; assay/interp/
  proper now depends on starlark-go-bazel for the same functionality.

**Effort:** 3 days.

### M8 — canopy plumbing (see plan 12)

**Scope:** `assay/interp/external/` thin caller. `canopy/internal/
external/` ingest layer. Per-module SQLite schema for external refs.
REST + MCP + UI surfaces stubbed.

**Acceptance:** Plan 12's milestones.

**Effort:** 5 days. (Tracked in canopy plan 12.)

### M9 — first external consumer

**Scope:** scip-bazel or compat-analyzer adopts. Validates the public
API surface. Triggers v0.x → v1.0 stabilization.

**Acceptance:**
- Second consumer ships using the library.
- API surface change list during their adoption ≤ N (small).
- v1.0 tag.

**Effort:** Separate plan — depends on which consumer goes first.

## Total effort (M0-M7)

22 working days (~4.5 calendar weeks for one engineer). M0 adds 2
days; M1-M7 stays at 20 days. M8 adds 5 days of canopy work. M9 is
open.

This is the genuine cost of upstreaming. The trade-off vs. shipping
airgap immediately from `assay/interp/spike/`: ~1.5 extra weeks now,
saving ~2 weeks of rework later AND unlocking scip-bazel / compat-
analyzer reuse without further refactoring.

M0 was inserted post-plan-writing once assay's E1c work hit the
real-input failures empirically. The 2-day cost prevents M2's
acceptance criteria from silently passing unit tests while
end-to-end aspect/tag_class evaluation still rejects valid input.

## Decision points the user can choose at

The plan structure allows for early branching:

- **After M0:** Aspect Tier-3 unblocked, drift-detection in place.
  The smallest possible scope that resolves the immediate real-
  input failures. assay's v0.2.0 release can ship with the
  deferred-Tier-3 limitations LIFTED. Lowest-investment outcome
  that delivers user-visible value (assay's E1c test reactivation).
- **After M1:** Could stop and just call this "expose
  PredeclaredBzl + Version"; that's a 4-day fix (M0+M1) and the
  spike continues using the proper API.
- **After M5:** Could pause; the library now has everything canopy
  airgap needs, but no version awareness or bazel_features. Canopy
  ingest can ship using ModeAnalysis. Versioning is then a "v0.2
  release later."
- **After M7:** Full promotion done; canopy can wire in (M8).
- **After M9:** Library stabilizes.

Each pause point is a legitimate stopping place. The first three are
particularly attractive if user wants to ship value sooner.
