# 01 — Bazel builtins emulation: the upstream library plan

**Scope:** evolve `starlark-go-bazel` from "Bazel builtins for analysis-time
`rule()`/`provider()`/`aspect()`" into a complete, version-aware Bazel
Starlark engine usable by linters, indexers, registries, security
scanners, and any other static-analysis tool that needs to *interpret*
Bazel `.bzl` and `MODULE.bazel` files without executing a build.

**Origin:** validated by the spike at
`/Volumes/T9/dev/ws/assay/interp/spike/` (see
`canopy/docs/plans/11-airgap-external-surface/` for the consumer-side
motivation and the
`~/dev/md/2026-05-19-canopy-airgap-eval-spike.md` journal for the
session-level findings). The spike proved repository_rule +
module_extension + repository_ctx + module_ctx + Permissive +
taint-tracking are viable on real production rulesets (`rules_go`).
This plan moves that work to its right home.

## Why this plan exists

Three forces converge:

1. **assay/interp/LIMITATIONS.md** documents `repository_rule()` and
   `module_extension()` as known gaps in starlark-go-bazel. Pinning
   tests are already in place to flip when upstream lands the
   builtins.
2. **canopy plan 11** (airgap external surface) needs these builtins
   to ship its Layer 3 static-eval URL extraction.
3. **scip-bazel**, **canopy's compat-analyzer (plan 05)**, and any
   other Bazel-aware tooling Alberto's portfolio or the wider
   community produces will want the same engine. Building it once
   upstream compounds.

The alternative — keeping the spike inside assay/interp/ — locks the
work away from those other consumers and forces eventual rewrites.

## Files in this plan

- [01-mission-current-state-and-gaps.md](01-mission-current-state-and-gaps.md)
  — what the library is, who it's for, what's already implemented,
  what the spike adds, and the list of "what's still wrong / missing
  / under-baked."
- [02-architecture-and-versioning.md](02-architecture-and-versioning.md)
  — proposed package layout, the `Version` enum, the `Mode` flag
  (strict / lenient / analysis), the extended `bzl.Options` surface,
  and the bazel_features compatibility story.
- [03-builtins-surface.md](03-builtins-surface.md) — exhaustive matrix
  of Bazel builtins / modules / types vs implementation status, tiered
  by "must have for canopy airgap," "needed for compat-analyzer,"
  "nice to have," and "out of scope."
- [04-permissive-and-taint.md](04-permissive-and-taint.md) — the
  semantic contract of the `Permissive` universal-stub value, the
  marker convention, the taint propagation rules, and what production
  needs to grow before this lib is suitable for hostile-input
  evaluation (it is not, today).
- [05-spike-promotion-and-testing.md](05-spike-promotion-and-testing.md)
  — file-by-file migration map from `assay/interp/spike/` to upstream
  packages, the API rename plan, the testing strategy (golden fixtures
  per-version, real-corpus tests, differential tests against
  `bazel fetch --experimental_repository_resolved_file`), and
  acceptance criteria per milestone.
- [06-risks-open-questions-and-milestones.md](06-risks-open-questions-and-milestones.md)
  — maintenance burden, API stability, WORKSPACE-era support
  decision, donation-to-bazelbuild question, the milestone sequence
  M0–M9 with concrete acceptance criteria.
- [07-attr-descriptor-unification.md](07-attr-descriptor-unification.md)
  — half of M0. The `builtins.AttrDescriptor` vs
  `types.AttrDescriptor` bifurcation that prevents real `.bzl`
  source from calling `aspect()` (and `tag_class()` once M2 wires
  consumers). Migrates the two consumer sites
  (`builtins/aspect.go`, `builtins/rule.go`) to the
  `types.AttrDescriptorHolder` interface that already exists,
  deletes the legacy type. TDD plan with 5 tests.
- [08-predeclared-dict-completeness.md](08-predeclared-dict-completeness.md)
  — other half of M0. Wires `aspect` into
  `eval/evaluator.go::makeBzlPredeclared` (one line) plus the
  drift-detection test family: manifest-driven completeness test,
  universe-eval smoke test, manifest-exercised-by-universe meta-
  test. Prevents M2's `repository_rule`/`module_extension`/
  `tag_class` wiring from being silently forgotten. Sub-manifests
  for `native.*` and `attr.*`; Bazel 8+ Status=missing entries
  for `subrule`, `exec_group`, `json`, `proto`.
- [09-assay-migration-coordination.md](09-assay-migration-coordination.md)
  — downstream-side handoff when M0 ships. Vendor refresh
  procedure, six-step reactivation of assay's deferred aspect
  Tier-3 (interp.go loop, PredeclaredBzl removal,
  TestHydrate_Aspect_HydratesAttrs restoration, CHANGELOG move,
  corpus verification, release tag). Cross-project release
  coordination table.

## Decision summary (so the rest of the plan has shape to push against)

1. **Versioning is runtime, not import path.** A `Version` enum on
   `bzl.Options` (`V7`, `V8`, `V9`, `VLatest`) pins the builtin
   surface and behavior deltas. Per-version subpackages (`/v8`, `/v9`)
   are not introduced until a real ABI break demands it.
2. **Mode flag selects strictness.** `ModeStrict` (real Bazel
   semantics, errors on unknowns), `ModeLenient` (permissive stubs
   for unresolvable loads — what the spike calls "Permissive"),
   `ModeAnalysis` (lenient + capture sinks active — the URL-extraction
   path).
3. **`stub/` and `taint/` are new packages.** Permissive and the
   marker constant live there, opt-in via `Mode`. Keeps the strict
   path free of the analysis-only machinery.
4. **`repository_rule` + `module_extension` go into `builtins/` and
   `ctx/`** next to existing `rule()` and analysis-time `ctx`. Not a
   side project; they fill the gap upstream's own README and
   downstream's LIMITATIONS.md already point at.
5. **bzl.Options grows additively** — new fields (`Version`, `Mode`,
   `PredeclaredBzl`, `PredeclaredBuild`, `CaptureSinks`) with
   zero-value backward compatibility. No breaking changes to
   existing consumers.

## Sequencing (high level)

```
M1  bzl.Options surface  ──── 2 days       ── unblocks consumer custom-builtins
M2  builtins/repository_rule + module_extension + tag_class ── 3 days
M3  ctx/repository_ctx + module_ctx          ── 4 days
M4  stub/ package (Permissive + loader)      ── 2 days
M5  taint/ package + Mode=Analysis wiring    ── 3 days
M6  version/ registry + Version enum + bazel_features stub ── 3 days
M7  spike promotion: assay/interp/spike/ → upstream + thin caller ── 3 days
M8  canopy/internal/external wires through assay/interp/external ── 5 days
M9  first external consumer adoption (scip-bazel or compat-analyzer) ── separate plan
```

Total to "canopy airgap ships against the upstreamed library": ~3
weeks of focused work. M1–M6 are starlark-go-bazel changes; M7 spans
two repos; M8 is canopy-only.

## What "done" means

For starlark-go-bazel:
- 100% of spike test cases (23) re-pass via the upstreamed API.
- `assay/interp/LIMITATIONS.md`'s "repository_rule + module_extension"
  entry flips from "pinned by negative test" to "supported."
- `bazel mod show_repo` against canopy with mirrored modules still
  works (no regression in canopy's smoke path).
- Real-corpus tests pass for the assay corpus (rules_cc, rules_go,
  rules_java, rules_python, rules_jvm_external, bazel-gazelle).
- Public API surface documented + Sonar/CI green.

For canopy:
- `canopy ingest` produces an `external_refs` table.
- Per-module UI surface shows the "External" tab populated.
- Closure-wide airgap report renders.

The spike is not the endpoint; it's the proof. This plan is the path
from proof to production.

## Cross-references

- Consumer plan: [canopy/docs/plans/12-starlark-go-bazel-integration/](../../../../canopy/docs/plans/12-starlark-go-bazel-integration/README.md)
- Spike code: `/Volumes/T9/dev/ws/assay/interp/spike/`
- Spike README: `/Volumes/T9/dev/ws/assay/interp/spike/README.md`
- Plan 11 (airgap, plan 12's parent): `/Volumes/T9/dev/ws/canopy/docs/plans/11-airgap-external-surface/`
- Existing limitations doc: `/Volumes/T9/dev/ws/assay/interp/LIMITATIONS.md`
- Spike journal: `~/dev/md/2026-05-19-canopy-airgap-eval-spike.md`
