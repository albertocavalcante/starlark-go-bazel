# 09 — assay migration coordination

**Scope:** The downstream-side handoff when M0 lands in
`starlark-go-bazel`. Specifies the vendor refresh procedure,
deferred-test reactivation order, CHANGELOG flip, and corpus
verification that assay should run before re-tagging.

**Why this plan exists in `starlark-go-bazel`'s tree:** M0 was
inserted into this library's plan because assay's E1c work
discovered the upstream gap. The handoff back to assay needs to
be precise so the deferral entries in assay's CHANGELOG
(currently pointing at "needs upstream type unification") flip
cleanly to "fixed in starlark-go-bazel v0.x." Documenting it
here keeps the cross-project contract in one place.

## Prerequisites — what must be true before this plan runs

1. `starlark-go-bazel`'s M0 has shipped (plans 07 + 08 landed).
2. `starlark-go-bazel` carries a tag identifying the M0
   completion (e.g., `v0.0.0-YYYYMMDDhhmmss-<sha>` if pre-v1,
   `v0.x.0` if tag-based). The exact form is whatever the
   downstream `go.mod` reachable-version mechanism prefers.
3. All five plan 07 + four plan 08 acceptance tests are GREEN.

If any of those is false, this plan is blocked.

## The assay-side state today (M0 prerequisite snapshot)

assay carries the following deferral markers as of Round E1c
commit `2e94e0f` (`feat(interp): document aspect Tier-3 deferral (E1c)`):

### `interp/interp.go`

The aspect-walking section in `Hydrate` is a documented no-op:

```go
// Aspects are NOT walked here today. starlark-go-bazel's
// aspect() builtin rejects the AttrDescriptor type produced by
// the attr.* module: it expects builtins.AttrDescriptor but
// receives types.AttrDescriptor (the path that everything else
// in the codebase uses). The .bzl fails to eval, so Hydrate has
// nothing to project. Track this as a known limitation in
// CHANGELOG and revisit when upstream unifies the two types.
```

### `interp/interp_test.go`

`TestHydrate_Aspect_HydratesAttrs` was removed during E1c. Only
`TestHydrate_Aspect_LeavesTier1ResultsAlone` remains, with a
comment:

```go
// Aspect Tier-3 (interpreter fallback) is NOT supported today —
// starlark-go-bazel's aspect() builtin rejects the AttrDescriptor
// shape produced by the attr.* module (it expects builtins.AttrDescriptor
// but receives types.AttrDescriptor that the rest of the codebase
// flows through). The .bzl fails to eval, so Hydrate has nothing to
// project. Until upstream unifies the two types, aspects with
// Tier-3-only attrs carry an empty AttrsExtractionMethod ...
```

### `CHANGELOG.md`

Under the `[Unreleased]` → `### Known limitations`:

> Aspect attrs that need Tier-3 are similarly not hydrated. Worked
> through the implementation locally and discovered an upstream
> incompatibility: `starlark-go-bazel`'s `aspect()` builtin expects
> `builtins.AttrDescriptor` but the `attr.*` module produces
> `types.AttrDescriptor`, so the aspect-hosting `.bzl` fails to
> evaluate before Hydrate can run. Revisit when upstream unifies
> the two types.

All three markers reference the SAME upstream constraint.

## Vendor refresh procedure

assay vendors `starlark-go-bazel` via a relative-path replace
directive (see assay's `go.mod`):

```
replace github.com/albertocavalcante/starlark-go-bazel => ../starlark-go-bazel
```

Steps:

1. **Bump starlark-go-bazel locally.** From assay's repo root:
   ```bash
   go get -u github.com/albertocavalcante/starlark-go-bazel@<m0-commit-sha>
   ```
   The replace directive resolves to `../starlark-go-bazel`, so the
   commit reference matters only for `go.mod`'s recorded version
   string (used downstream when the replace eventually disappears).

2. **Refresh the vendor tree.**
   ```bash
   go mod tidy
   go mod vendor
   ```
   Expected diff: `vendor/.../starlark-go-bazel/` updates to the
   M0 state. Specifically:
   - `vendor/.../starlark-go-bazel/builtins/aspect.go` no longer
     uses `*AttrDescriptor` in the type assertion (line ~180
     pre-M0); uses `types.AttrDescriptorHolder` post-M0.
   - `vendor/.../starlark-go-bazel/builtins/rule.go` same.
   - `vendor/.../starlark-go-bazel/builtins/rule.go` no longer
     contains the `type AttrDescriptor struct { ... }` definition
     at line ~350 — the type is deleted.
   - `vendor/.../starlark-go-bazel/eval/evaluator.go` includes
     `"aspect": starlark.NewBuiltin("aspect", builtins.Aspect)`
     in `makeBzlPredeclared`.
   - New file `vendor/.../starlark-go-bazel/eval/predeclared_manifest.go`
     appears.

3. **Verify the vendor tree.**
   ```bash
   go build ./...
   ```
   Expected: clean compile. If any consumer of `*builtins.AttrDescriptor`
   leaked into assay's own code (it shouldn't — the type was
   internal to starlark-go-bazel), this catches it.

## Reactivation steps

### Step A — restore `interp/interp.go`'s aspect-walking loop

Replace the documented no-op with the actual implementation
(reverse of the E1c revert):

```go
// Aspects flow through the same evalFile/global-lookup machinery.
// The aspect's local binding name (`my_aspect = aspect(...)`) IS
// the global it's exported under, so lookupAspectClass(file, name)
// is exactly parallel to the rule path.
for i := range rep.Aspects {
    a := &rep.Aspects[i]
    if a.AttrsExtractionMethod != "" {
        continue
    }
    if a.Provenance.File == "" {
        continue
    }
    ac := cache.lookupAspectClass(a.Provenance.File, a.Name)
    if ac == nil {
        continue
    }
    a.Attrs = attrsFromAspectClass(ac)
    a.AttrsExtractionMethod = report.AttrsInterpreted
}
```

Plus the import of `builtins`:

```go
"github.com/albertocavalcante/starlark-go-bazel/builtins"
```

And the helpers (`lookupAspectClass`, `attrsFromAspectClass`),
both of which assay drafted during E1c and then reverted. The
git history at commit `2e94e0f^` carries the working drafts.

### Step B — un-defer the `PredeclaredBzl` aspect injection

The E1c spike added `PredeclaredBzl: starlark.StringDict{"aspect":
starlark.NewBuiltin("aspect", builtins.Aspect)}` to assay's
evalCache config. Post-M0 this is unnecessary — `aspect` is in
`makeBzlPredeclared` upstream. Delete the override; keep
`bzl.Options{WorkspaceRoot, LenientLoad}` as it was pre-E1c.

### Step C — restore `TestHydrate_Aspect_HydratesAttrs`

Re-add the test that E1c deleted:

```go
func TestHydrate_Aspect_HydratesAttrs(t *testing.T) {
    src := `
def _impl(target, ctx):
    pass

def make_attrs():
    return {
        "src": attr.label(),
        "level": attr.int(),
    }

my_aspect = aspect(
    implementation = _impl,
    attrs = make_attrs(),
)
`
    dir := t.TempDir()
    if err := os.WriteFile(filepath.Join(dir, "defs.bzl"), []byte(src), 0o644); err != nil {
        t.Fatal(err)
    }
    rep := &report.ModuleReport{
        Aspects: []report.AspectSpec{{
            Name:       "my_aspect",
            Provenance: report.Provenance{File: "defs.bzl"},
        }},
    }
    interp.Hydrate(context.Background(), dir, rep)
    a := rep.Aspects[0]
    if a.AttrsExtractionMethod != report.AttrsInterpreted {
        t.Errorf("AttrsExtractionMethod = %q, want %q", a.AttrsExtractionMethod, report.AttrsInterpreted)
    }
    if len(a.Attrs) != 2 {
        t.Fatalf("Attrs = %d, want 2 (src, level); got %+v", len(a.Attrs), a.Attrs)
    }
}
```

Run it. Expected: GREEN (because step A wired the loop and the
vendor refresh installed the upstream fix).

### Step D — update `CHANGELOG.md`

Move the "Aspect attrs that need Tier-3" entry from "Known
limitations" to "Fixed":

```markdown
### Fixed

- Aspect Tier-3 (interpreter fallback for `attrs = make_helper()`-style
  aspects) now works. Was blocked on
  `starlark-go-bazel`'s `aspect()` builtin rejecting the
  `AttrDescriptor` type that `attr.*` produces; upstream M0 (plans
  07 + 08) unified the type via the `AttrDescriptorHolder` interface.
  assay's vendor refresh picks up the fix; aspects with attrs that
  defeat Tier 0/1/2 (`attrs = make_attrs()`, `attrs = A | B if cond
  else C`) now hydrate via Tier 3.
```

### Step E — corpus verification

Re-run the existing corpus test:

```bash
REFS_DIR=$HOME/dev/refs just test-corpus
```

Expected: green. Specifically:
- rules_go's `go_pkg_info_aspect` continues to surface 2 attrs
  (was Tier-1, still is). Verify the count didn't regress.
- Any synthetic corpus aspect with `attrs = make_attrs()` style
  (if one exists; none in the v0.1 corpus) would flip from
  `attrs=[]` to populated.

Add a NEW synthetic corpus aspect in `testdata/` exercising the
Tier-3 path — small `.bzl` with `make_attrs()`, an aspect that
references it, a test asserting Tier-3 hydration produces the
expected attrs. This is the regression guard against a future
vendor refresh accidentally re-breaking the path.

### Step F — release tag

assay's v0.2.0 wraps up (Round E + F + G of the registry-surface
plan). The aspect-Tier-3 fix is one item in the "Fixed" list.
Tag and ship coordinated with canopy's upgrade.

## Cross-project release coordination

Two repos, two release events, one bug fix:

| Event | Repo | Action |
|---|---|---|
| 1 | starlark-go-bazel | Land M0 (plans 07 + 08). |
| 2 | starlark-go-bazel | Tag M0 commit. |
| 3 | assay | Vendor refresh + steps A-E. |
| 4 | assay | Run full corpus; verify no regression. |
| 5 | assay | Tag v0.2.0 if also gated on this; otherwise commit and continue Round E2+. |
| 6 | canopy | Upgrade assay reachable version when ready. |

The starlark-go-bazel tag exists for `go.mod` provenance but
isn't load-bearing since assay vendors. The tag is for clean
history + signals readiness for non-assay consumers (scip-bazel,
compat-analyzer).

## Verification checklist

Before considering the migration complete:

- [ ] assay's `go build ./...` clean.
- [ ] `just check` clean (fmt, vet, lint, modernize, test-race).
- [ ] `TestHydrate_Aspect_HydratesAttrs` GREEN.
- [ ] Existing `TestHydrate_Aspect_LeavesTier1ResultsAlone` still
      GREEN (no regression on the no-op-when-resolved contract).
- [ ] Corpus test GREEN; specifically rules_go's
      `go_pkg_info_aspect` attr count stable.
- [ ] CHANGELOG entry moved from "Known limitations" to "Fixed".
- [ ] No leftover `// TODO(starlark-go-bazel M0)` comments in
      assay code.
- [ ] If a synthetic Tier-3 corpus fixture was added (step E),
      it's wired into `testdata/` and exercised by `TestCorpus`.

## Risk register

- **Risk:** Vendor refresh introduces other unrelated changes
  from starlark-go-bazel (M2 builtins already in flight, etc.) that
  break assay. **Mitigation:** the M0 commit is tagged; assay
  bumps to exactly that SHA, not `@latest`. If subsequent
  starlark-go-bazel work needs to ship first, assay holds at the
  M0 tag until ready.
- **Risk:** A consumer-side fix breaks for an edge case the M0
  tests didn't cover. **Mitigation:** assay's corpus check
  catches it. If a regression surfaces, file an issue against
  starlark-go-bazel referencing the failing fixture; revert
  assay's reactivation if blocking.
- **Risk:** The "synthetic corpus aspect with Tier-3 attrs"
  fixture (step E) takes longer to construct than expected.
  **Mitigation:** ship step E as a TODO in step F's tag and add
  the fixture in a follow-up commit. Steps A-D are the
  load-bearing work; step E is the regression guard.

## Effort

| Step | Work | Hours |
|---|---|---|
| Vendor refresh + verify | go mod tidy + go mod vendor + go build | 0.5 |
| A | Restore aspect-walking loop in interp.go | 0.5 |
| B | Remove PredeclaredBzl override | 0.1 |
| C | Restore TestHydrate_Aspect_HydratesAttrs | 0.25 |
| D | CHANGELOG move | 0.25 |
| E | Synthetic corpus fixture + corpus run | 1.0 |
| F | Verify checklist + commit + push | 0.5 |

Total: **~3 hours**, i.e., a single sitting once starlark-go-bazel's
M0 tag lands.

## When this plan runs

After all M0 acceptance criteria in plan 06's M0 section pass.
Specifically:

- Plan 07's T1 + T2 + T5 GREEN.
- Plan 08's T1 + T2 + T3 + T4 GREEN.
- M0 commit tagged in starlark-go-bazel.

NOT during assay's Round E2 / F / G work for v0.2.0 — those
proceed in parallel; the migration happens as a separate
commit chain once upstream is ready.

assay's v0.2.0 release can include this fix if M0 lands first,
or ship without it if M0 isn't ready in time. The CHANGELOG
entry's location (Fixed vs. Known limitations) depends on
which ships first.
