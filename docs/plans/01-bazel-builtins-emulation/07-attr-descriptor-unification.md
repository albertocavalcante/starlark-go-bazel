# 07 — AttrDescriptor unification

**Scope:** Resolve the bifurcation between `types.AttrDescriptor` and
`builtins.AttrDescriptor` that prevents real `.bzl` source from calling
`aspect()`, `rule()`, and (once M2 fills out their consumer sites)
`tag_class()`, `repository_rule()`, `module_extension()` with the
output of `attr.*` constructors.

**Origin:** discovered during downstream assay's Round E1c work
(`assay/docs/registry-surface-plan.md`). Aspect Tier-3 hydration of
`my_aspect = aspect(implementation = _impl, attrs = {"src": attr.label()})`
failed empirically with `aspect: attrs values must be attr objects,
got Attribute for "src"`. Plan 02 (line 45) noted this gap as
"existing; gap noted (separate fix plan)" — this doc is that
separate fix plan, promoted into Round 01 because it blocks M2's
acceptance criteria on real input.

## Current state

Two `AttrDescriptor` types exist:

### `types.AttrDescriptor` (canonical going forward)

`types/rule_class.go:34`. Public fields, richer surface, mirrors
Bazel's Java `AttrDescriptor` shape:

```go
type AttrDescriptor struct {
    Name            string
    Type            AttrType
    Default         starlark.Value
    Mandatory       bool
    Doc             string
    AllowedFiles    []string
    AllowedRules    []string
    Configurable    bool
    NonConfigurable bool
    Executable      bool
    SingleFile      bool
    AllowEmpty      bool
    Providers       []string
}
```

Used by `types.RuleClass.Attrs()`. Consumed by downstream tooling
(assay's `interp.attrsFromRuleClass` reads `ad.Type`, `ad.Doc`,
`ad.Mandatory`, `ad.Providers`).

### `builtins.AttrDescriptor` (legacy)

`builtins/rule.go:350`. Private fields, narrower accessor-based
surface:

```go
type AttrDescriptor struct {
    attrType        string
    defaultValue    starlark.Value
    doc             string
    mandatory       bool
    allowEmpty      bool
    allowFiles      starlark.Value
    allowRules      []string
    providers       []starlark.Value
    allowSingleFile bool
    executable      bool
    cfg             starlark.Value
    aspects         []starlark.Value
    values          []string
    frozen          bool
}
```

Accessor methods: `AttrType()`, `DefaultValue()`, `IsMandatory()`.
No `Doc()` accessor today.

### The bridge that already exists

`types/attr_descriptor_holder.go` defines:

```go
type AttrDescriptorHolder interface {
    Descriptor() *AttrDescriptor
}
```

`eval/evaluator.go:478` shows the wrapper that implements it:

```go
type attrDescriptorValue struct {
    desc *types.AttrDescriptor
}

func (a *attrDescriptorValue) Descriptor() *types.AttrDescriptor { return a.desc }
```

`attr.label()`, `attr.string()`, etc. return `*attrDescriptorValue`
— a `starlark.Value` that holds a `*types.AttrDescriptor` and
exposes it via the holder interface.

### Where the bridge is NOT used

Two consumer sites still type-assert against `*builtins.AttrDescriptor`:

- `builtins/aspect.go:180` —
  ```go
  desc, ok := item[1].(*AttrDescriptor)
  ```
- `builtins/rule.go:222` — identical pattern.

Neither site checks for `types.AttrDescriptorHolder`. The producer
returns one shape; the consumer asserts another; the assertion
fails on every real call.

`tag_class()`, `repository_rule()`, `module_extension()` (the new
M2-era builtins) currently store attrs loosely as
`map[string]starlark.Value` and defer the descriptor extraction.
That postpones the problem rather than solving it — anything that
tries to read a tag_class's attrs back as descriptors hits the same
wall.

## Recommended approach: consumers adopt the holder interface

The holder interface IS the intended path. We just haven't wired the
consumer sites yet.

Concretely: every consumer that previously type-asserted against
`*builtins.AttrDescriptor` switches to:

```go
holder, ok := item[1].(types.AttrDescriptorHolder)
if !ok {
    return nil, fmt.Errorf("aspect: attrs values must be attr objects, got %s for %q", item[1].Type(), name)
}
desc := holder.Descriptor()
// desc is *types.AttrDescriptor; read .Type, .Default, .Mandatory, etc.
```

Field access switches from accessor methods (`desc.AttrType()`) to
exported fields (`string(desc.Type)`).

`builtins.AttrDescriptor` becomes dead code once both consumer
sites switch, and gets deleted in the same milestone.

### Why this over the alternatives

**Alternative A — make `aspect()`/`rule()` accept either type.**
Define an interface, implement on both, accept the interface.
Rejected: leaves both types alive, doubles the documentation
burden, no semantic benefit.

**Alternative B — keep `builtins.AttrDescriptor`, mirror `attr.*` to
return it.** Would mean `attr.label()` produces the older, narrower
type. Rejected: regresses on the richer field set downstream uses;
the producer side already returns the better type and the system
is mid-migration.

**Recommended C — adopt the holder, delete the loser.** Minimal
churn (two consumer sites + delete one type). Builds on the
infrastructure already in place. Preserves the richer downstream
shape.

## Migration steps

### Step 1 — `builtins/aspect.go`

Change the type assertion at line 180. Update downstream field
references in the same function (lines 188–203) from
`desc.attrType` → `string(desc.Type)`, `desc.defaultValue` →
`desc.Default`. Adjust the `AspectClass` struct's `attrs` field
type from `map[string]*AttrDescriptor` to
`map[string]*types.AttrDescriptor` and its `Attrs()` accessor's
return type to match.

### Step 2 — `builtins/rule.go`

Same migration, line 222 and downstream. The `rule()` builtin
threads attrs into `types.RuleClass`; verify the construction
chain ends up storing `*types.AttrDescriptor` (it should already;
the type assertion is the only mismatch).

### Step 3 — `builtins/tag_class.go`, `module_extension.go`,
`repository_rule.go`

The loose `map[string]starlark.Value` storage stays for now (it
defers descriptor extraction). When their downstream consumers
materialize (e.g., a module_extension wants to enumerate its
tag_classes' attrs), those consumers use
`types.AttrDescriptorHolder.Descriptor()` to extract.

Add explicit doc-comments to each builtin's source:
"Values in this map are expected to implement
`types.AttrDescriptorHolder`. Use `Descriptor()` to extract the
canonical `*types.AttrDescriptor`."

### Step 4 — delete `builtins.AttrDescriptor`

After steps 1 + 2, the type has no remaining references. Delete it
(plus its constructor helpers, the `Freeze()` impl, the
`starlark.Value` interface methods). Run `go vet ./...`,
`golangci-lint run`.

### Step 5 — vendor refresh in assay

`assay/vendor/.../starlark-go-bazel/` refreshes. Assay's E1c
deferred test (`TestHydrate_Aspect_HydratesAttrs` in
`interp/interp_test.go`) flips from deferred to active. The
`attrsFromAspectClass` helper assay drafted during the spike (now
reverted) gets restored.

## Edge cases

### Forgiving consumers

A consumer might want to accept BOTH the canonical descriptor and
something looser (e.g., `None` to mean "no descriptor required").
Use a chained type assertion:

```go
switch v := item[1].(type) {
case types.AttrDescriptorHolder:
    desc := v.Descriptor()
    // ...
case starlark.NoneType:
    // attr without descriptor; whether this is legal depends on
    // the consumer
default:
    return nil, fmt.Errorf(...)
}
```

### Frozen state

`types.AttrDescriptor` has public fields. A caller could
theoretically mutate them after construction. Mitigations:

- Add a `frozen bool` field + `Freeze()` method that flips it
- Document that `AttrDescriptor` values returned from `attr.*` are
  conceptually frozen; mutation is undefined behavior
- Defer enforcement until a real-world misuse surfaces

Today the risk is theoretical; no code mutates a descriptor after
construction.

### Future Bazel versions

If Bazel adds new descriptor fields (Bazel 9.x typically does), the
unification keeps adding-fields-trivial: extend
`types.AttrDescriptor` with the new field; producers populate it;
consumers ignore it until they need it. No type-version axis
needed within this concern.

## TDD plan

Tests to write FIRST (red), then implementation, then verify
(green). Each test isolates one consumer's interaction with the
holder interface.

### T1. `builtins/aspect_attrs_test.go::TestAspect_AttrsRoundTrip`

```go
func TestAspect_AttrsRoundTrip(t *testing.T) {
    src := `
def _impl(target, ctx): pass

my_aspect = aspect(
    implementation = _impl,
    attrs = {
        "src": attr.label(),
        "level": attr.int(),
    },
)
`
    globals := evalBzl(t, "defs.bzl", src)
    asp, ok := globals["my_aspect"].(*AspectClass)
    require.True(t, ok, "my_aspect not an AspectClass: got %T", globals["my_aspect"])
    attrs := asp.Attrs()
    require.Len(t, attrs, 2)
    require.Equal(t, "label", string(attrs["src"].Type))
    require.Equal(t, "int", string(attrs["level"].Type))
}
```

State today: RED (aspect: attrs values must be attr objects).
After step 1: GREEN.

### T2. `builtins/rule_attrs_round_trip_test.go::TestRule_AttrsStillWorks`

```go
func TestRule_AttrsStillWorks(t *testing.T) {
    src := `
def _impl(ctx): pass

my_rule = rule(
    implementation = _impl,
    attrs = {"srcs": attr.label_list(), "lib": attr.label()},
)
`
    globals := evalBzl(t, "defs.bzl", src)
    r, ok := globals["my_rule"].(*types.RuleClass)
    require.True(t, ok)
    attrs := r.Attrs()
    require.Len(t, attrs, 2)
    require.Equal(t, "label_list", string(attrs["srcs"].Type))
}
```

State today: should already be GREEN if `rule()`'s end-to-end path
already works. The test guards against regression during step 2's
type-assertion swap. (If it's RED today, the same migration fixes
both builtins simultaneously.)

### T3. `builtins/tag_class_attrs_test.go::TestTagClass_AttrsHolderInterface`

```go
func TestTagClass_AttrsHolderInterface(t *testing.T) {
    src := `
_install = tag_class(attrs = {"src": attr.label(), "name": attr.string()})
`
    globals := evalBzl(t, "defs.bzl", src)
    tc, ok := globals["_install"].(*types.TagClass)
    require.True(t, ok)
    rawAttrs := tc.Attrs() // map[string]starlark.Value per step 3
    require.Len(t, rawAttrs, 2)
    // Caller extracts descriptors via the holder interface:
    src1, ok := rawAttrs["src"].(types.AttrDescriptorHolder)
    require.True(t, ok)
    require.Equal(t, "label", string(src1.Descriptor().Type))
}
```

State today: RED (tag_class isn't yet wired in `makeBzlPredeclared`
— see plan 08 — and even when wired, the loose-storage decision
means the test exercises the holder pattern explicitly).
GREEN after plan 08 wires tag_class + this test passes via the
documented loose-storage path.

### T4. `builtins/module_extension_tag_class_test.go::TestModuleExtension_TagClassAttrs`

```go
func TestModuleExtension_TagClassAttrs(t *testing.T) {
    src := `
_install = tag_class(attrs = {"src": attr.label()})
my_ext = module_extension(
    implementation = lambda mctx: None,
    tag_classes = {"install": _install},
)
`
    globals := evalBzl(t, "defs.bzl", src)
    ext, ok := globals["my_ext"].(*types.ModuleExtensionClass)
    require.True(t, ok)
    // Tag classes accessible and their attrs reach back to types.AttrDescriptor:
    tcs := ext.TagClasses()
    inst := tcs["install"]
    rawAttrs := inst.Attrs()
    h := rawAttrs["src"].(types.AttrDescriptorHolder)
    require.Equal(t, "label", string(h.Descriptor().Type))
}
```

End-to-end shape test — covers the M2/M3 surface using the
unified descriptor path. RED until M2 lands; GREEN at M2's
acceptance.

### T5. `builtins/no_legacy_attr_descriptor_test.go::TestBuiltinsAttrDescriptor_Deleted`

```go
//go:build !legacy_attr_descriptor

func TestBuiltinsAttrDescriptor_Deleted(t *testing.T) {
    // Reflectively assert builtins.AttrDescriptor no longer exists
    // as an exported type. Catches a re-introduction during a
    // future refactor.
    pkg := reflect.TypeOf((*AspectClass)(nil)).Elem().PkgPath()
    _ = pkg
    // ... lookup mechanism here. If we delete the type, this test
    //     compiles; if someone re-adds it, the compile fails.
}
```

State today: doesn't apply (type still exists). After step 4
(deletion): GREEN by virtue of the type not being there. Belt-
and-suspenders against re-introduction.

## Acceptance

- Steps 1–4 land in `starlark-go-bazel`.
- All five tests above pass (T3/T4 pass after their respective
  M2-side dependencies; T1/T2 pass at M0).
- `go vet ./...` clean.
- `golangci-lint run` clean.
- assay's vendor refresh + reactivation of the deferred aspect
  Tier-3 test (`assay/interp/interp_test.go`'s
  `TestHydrate_Aspect_HydratesAttrs` — currently the test file
  documents the deferral) passes.

## Effort

| Step | Work | Days |
|---|---|---|
| 1 | aspect.go consumer fix + T1 | 0.25 |
| 2 | rule.go consumer fix + T2 | 0.25 |
| 3 | tag_class/M2 doc-comments + T3/T4 stubs | 0.25 |
| 4 | Delete `builtins.AttrDescriptor` + T5 | 0.25 |
| 5 | assay vendor refresh + reactivate deferred test | 0.5 |

Total: **1.5 days.**

## Risk register additions

- **Risk:** Type assertion sites missed during the audit fail
  silently (silently accept None instead of erroring loudly).
  **Mitigation:** the explicit type assertions in steps 1–2 DO
  error loudly; the `default:` case in the type switch always
  returns a "got %s, want attr object" error.
- **Risk:** `types.AttrDescriptor`'s public fields enable external
  mutation. **Mitigation:** add `Freeze()` if a real-world misuse
  surfaces; today theoretical.
- **Risk:** A future Bazel version reshapes `AttrDescriptor`
  semantically (not just adds fields). **Mitigation:** the type's
  shape lives in `types/`, gated by Version where needed — same
  versioning mechanism the rest of the library uses.

## Public API impact

### `AspectClass.Attrs()` return-type change

Current signature:

```go
func (a *AspectClass) Attrs() map[string]*AttrDescriptor
```

where `AttrDescriptor` resolves to `*builtins.AttrDescriptor`. M0
changes this to:

```go
func (a *AspectClass) Attrs() map[string]*types.AttrDescriptor
```

**Consumer audit.** A `grep -rn "AspectClass.*Attrs()" --include="*.go"`
across assay + canopy worktrees identifies no live consumers as of
plan-writing time. The helper assay drafted during the E1c spike
(`attrsFromAspectClass`) was reverted with the deferral and already
targets `*types.AttrDescriptor` when restored. The change is
breaking by Go's API rules but has no shipping consumer to break.

**CHANGELOG entry for the M0 release tag of `starlark-go-bazel`:**

```markdown
### Changed

- **BREAKING**: `builtins.AspectClass.Attrs()` returns
  `map[string]*types.AttrDescriptor` instead of
  `map[string]*builtins.AttrDescriptor`. Migrate by reading the
  descriptor's exported fields (`.Name`, `.Type`, `.Default`,
  `.Mandatory`, `.Doc`, `.Providers`) instead of the legacy
  accessor methods. `types.AttrDescriptor`'s field set is strictly
  richer; no information is lost.

### Removed

- `builtins.AttrDescriptor` type. The wrapper struct
  (`builtins/rule.go:350`) is replaced by `types.AttrDescriptor`
  accessed through the `types.AttrDescriptorHolder` interface
  (already exposed by `attrDescriptorValue` in
  `eval/evaluator.go`).

### Fixed

- `aspect()` and `rule()` no longer reject the values produced by
  `attr.*`. Previously the consumer-side type assertion against
  `*builtins.AttrDescriptor` failed because `attr.*` returns a
  value wrapping `*types.AttrDescriptor`; every real `.bzl` source
  path triggered the rejection.
```

### M2-builtin forward-compatibility

`tag_class()`, `module_extension()`, `repository_rule()` (already
landed locally per `git status` — files exist as untracked) store
attrs as `map[string]starlark.Value`, deferring descriptor
extraction. M0 codifies the contract by adding a normative
doc-comment on each constructor:

```go
// tag_class stores its attrs kwarg as a map of starlark.Value.
// Each value is expected to implement types.AttrDescriptorHolder —
// concretely, the *attrDescriptorValue wrappers returned by the
// attr.* module. Consumers extract the canonical descriptor via:
//
//   for name, v := range tc.Attrs() {
//       holder, ok := v.(types.AttrDescriptorHolder)
//       if !ok { ... }
//       desc := holder.Descriptor() // *types.AttrDescriptor
//   }
//
// This pattern is exercised by integration tests (T3, T4 below)
// and enforced by M2's acceptance criteria (plan 06).
```

The doc-comment is half the contract. The other half lives in
M2's acceptance criteria (plan 06's revised M2 section): any new
helper added in M2 that returns typed descriptors MUST go through
the holder interface, not a `*builtins.AttrDescriptor` shortcut.

### Cyclic-dependency safety check

`builtins/` imports `types/`. `types/` does not import `builtins/`.
The holder interface lives in `types/`, so `builtins/` can adopt it
without inverting the existing edge:

```bash
go list -deps ./types/... | grep starlark-go-bazel/builtins
# (empty)
go list -deps ./builtins/... | grep starlark-go-bazel/types
# starlark-go-bazel/types
```

No risk of introducing a cycle. The migration is a one-way edge
addition: `builtins/aspect.go` learns about `types.AttrDescriptorHolder`,
which is in the package it already imports.

## Coordination with other M0 work

This plan covers HALF of M0. Plan 08 covers the other half (wiring
`aspect()` and future M2 builtins into `makeBzlPredeclared`, with
drift-detection tests). Both should land together — the holder-
interface fix alone doesn't help if `aspect()` isn't reachable from
.bzl source in the first place.

**Implementation order within M0:**

1. Plan 08 — register `aspect` in `makeBzlPredeclared` (one line).
2. Plan 07 step 1 — migrate `builtins/aspect.go`'s consumer site
   to the holder interface; plan 07 T1 (`TestAspect_AttrsRoundTrip`)
   flips RED → GREEN.
3. Plan 07 step 2 — migrate `builtins/rule.go`'s consumer site;
   verify T2 stays GREEN (no regression).
4. Plan 07 step 4 — delete `builtins.AttrDescriptor`; T5 confirms
   the deletion.
5. Plan 08 — commit `predeclared_manifest.go`; T2, T3, T4 flip
   GREEN.
6. Plan 09 (next) — assay-side vendor refresh + deferred-test
   reactivation.

Order matters: step 1 before step 2 because T1 fails to even
compile without `aspect` being reachable from `.bzl` (the eval
errors with "undefined: aspect" before the type assertion runs).
