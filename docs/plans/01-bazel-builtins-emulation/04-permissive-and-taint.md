# 04 — Permissive and taint: semantic contract

The `Permissive` value type and the taint propagation rules are the
load-bearing pieces of `ModeAnalysis`. They're how the library says
"I evaluated this code but some inputs were unknown, here's what I
learned anyway." Production must preserve their semantics exactly.

## What `Permissive` is

A single Starlark value type that:

1. **Satisfies every interface that could be invoked on an unknown
   value**, so eval continues instead of erroring at name resolution
   or operator dispatch.
2. **Carries a textual marker** (`taint.Marker = "<permissive>"`)
   in its `String()` output so any path that converts it to a string
   (`str()`, `.format()`) preserves the taint signal in the resulting
   value.
3. **Returns `Shared` (a single package-level instance) from every
   operation** that would otherwise allocate, so deeply nested access
   chains don't allocate per access.

### Interface implementations (from spike, target for upstream)

| Interface | Method | Behavior |
|---|---|---|
| `starlark.Value` | `String()` | Returns `taint.Marker` (the sentinel) |
| `starlark.Value` | `Type()` | Returns `"permissive"` |
| `starlark.Value` | `Freeze()` | No-op |
| `starlark.Value` | `Truth()` | True (so `if value:` flows through to the true branch) |
| `starlark.Value` | `Hash()` | Errors — Permissive is unhashable today; M9+ may add hashable Permissive |
| `starlark.Callable` | `Name()` | `"permissive"` |
| `starlark.Callable` | `CallInternal(...)` | Returns `Shared` |
| `starlark.HasAttrs` | `Attr(name)` | Returns `Shared` |
| `starlark.HasAttrs` | `AttrNames()` | Returns nil |
| `starlark.Mapping` | `Get(k)` | Returns `(Shared, true, nil)` |
| `starlark.HasBinary` | `Binary(op, y, side)` | See below |
| `starlark.Comparable` | `CompareSameType(op, y, depth)` | EQ=false, NEQ=true, ordered=error |

### Binary operator — the subtle one

`Permissive.Binary` is **not** "return Shared for every op." It
distinguishes the string-concat case:

```go
func (p *Permissive) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
    if op == syntax.PLUS {
        if ys, ok := starlark.AsString(y); ok {
            // Preserve the known portion of the URL; embed the marker
            // so downstream taint detection still flags it.
            if side == starlark.Left {
                return starlark.String(taint.Marker + ys), nil
            }
            return starlark.String(ys + taint.Marker), nil
        }
    }
    return Shared, nil
}
```

Rationale: collapsing `"https://example.com/" + perm` to `Shared`
would yield URL `<unresolved>` — true but useless. Producing
`"https://example.com/<permissive>"` is *equally* tainted and
preserves the recognizable prefix for the airgap report.

### Comparable — closes the second residual

```go
func (p *Permissive) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
    switch op {
    case syntax.EQL: return false, nil  // conservative: don't claim equal
    case syntax.NEQ: return true,  nil
    }
    return false, fmt.Errorf("permissive: ordered comparison %s not supported", op)
}
```

Same-type EQ is conservative-false: we don't know if two Permissives
represent equal values, so default to not-equal. Same-type NEQ is
the inverse. Ordered (`<`, `<=`, `>`, `>=`) errors — there's no
sensible answer; the per-fork ForkError surfaces this to the caller.

Cross-type EQ/NEQ doesn't go through this method — go.starlark.net's
default `Equal` returns false/true on type mismatch without erroring.
So `Permissive == "linux"` resolves to false, the else branch runs.

## What `Permissive` is NOT (production wishlist)

### Not symbolic execution

`if perm == "linux": A() else: B()` runs only the else branch
(because EQ returns false). A future symbolic-execution mode could
fork eval to explore both branches; out of scope for this plan.

### Not hashable

`sdks[perm]` errors at `Permissive.Hash()`. A future hashable
Permissive that returns a sentinel hash and consistently equals
nothing would let dict lookups return "key not found" instead of
aborting the fork. Out of scope for this plan.

### Not a TaintedString

The marker-embedded `starlark.String` returned by Binary is a regular
String once it's out of Binary. If the consumer then does
`url.upper()` or splits it, the marker survives (string methods are
pure). But the *type* loses the taint affordance — `Has()` is the
only way to know.

A future `TaintedString` type could carry taint at the value level
(satisfy starlark.String semantics + an additional Tainted marker
field). Would catch:

- `str(perm)` → TaintedString instead of regular String
- `"%s" % perm` → TaintedString
- All future code paths that convert Permissive to text

Trade-off: pervasive type-checking changes throughout eval, plus
interactions with go.starlark.net's String type identity assumptions.
**Not in this plan; M10+ if a real consumer demands it.**

## Taint propagation rules

Three sources of taint, each independently sufficient to mark a URL
as `Tainted=true`:

### Source A — Direct Permissive in URL arg

```python
result = ctx.execute(["uname"])
ctx.download(url = result.stdout)  # result.stdout is Permissive
```

`flattenURLs` type-asserts; emits `URL="<unresolved>", Tainted=true`.

### Source B — Permissive in string concat (marker propagation)

```python
url = "https://example.com/" + result.stdout
# Binary returns String("https://example.com/<permissive>")
ctx.download(url = url)
```

`flattenURLs` substring-detects `taint.Marker`; emits
`URL="https://example.com/<permissive>", Tainted=true`. URL prefix
preserved.

This catches:
- Direct binary: `"x" + perm` or `perm + "x"`
- `str(perm)` then concat: `"x" + str(perm)`
- Format: `"{}".format(perm)` (Starlark's format calls `str()` on
  args, picks up Permissive.String() → marker)

### Source C — Per-fork tainted flag

```python
ctx.execute(["uname"])
ctx.download(url = "https://example.com/clean.tar.gz")  # NOT tainted at expression
                                                         # but per-fork flag is set
```

`repository_ctx.tainted` flips to true on first call to `.execute`,
`.read`, or `.which`. Subsequent `ctx.download` calls in the same
fork record `Tainted=true` even if the URL expression contains no
Permissive.

Rationale: the impl ran code we couldn't faithfully simulate; we
don't know if a different `.execute` return would have changed
control flow above the download. Conservative.

### Ordering guarantee

Downloads BEFORE any opaque op in a fork stay clean. This is
deliberately order-sensitive: a download earlier in the impl is
unaffected by a later `.execute` call (because the download already
happened with concrete inputs).

```python
ctx.download(url = "https://example.com/clean.tar.gz")  # Tainted=false
ctx.execute(["uname"])                                   # flips flag
ctx.download(url = "https://example.com/dirty.tar.gz")  # Tainted=true
```

Test pin: `TestSpike_Taint_DownloadBeforeExecuteIsClean`.

## The marker convention

`taint.Marker = "<permissive>"` is the textual sentinel embedded into
any string derived from a Permissive value (via String() or Binary).

### Why this string

- **Recognizable** in user-facing output ("URL prefix preserved,
  unresolved suffix shown as <permissive>").
- **Unlikely false-positive** in real URLs (no real URL contains
  literal `<permissive>` except in pathological test data).
- **Substring-detectable** in O(N) for tainted-URL classification.

### Stability commitment

Once published, `taint.Marker` is a public Go constant. Third-party
consumers will:
- Check `taint.Has(string)` before showing URLs to users.
- Render `<permissive>` segments as italicized "unresolved" in UIs.

**The exact string `<permissive>` must not change in a backward-
incompatible way.** If we ever need to evolve it (e.g., to a
versioned `<permissive@v2>`), do so additively — keep the v1
substring detectable for one major release.

### When the marker is a false positive

Theoretical: a literal URL like
`"https://example.com/<permissive>/foo.tar.gz"` would be flagged
tainted. In practice this doesn't happen in real corpus. CI can
include a smoke test that confirms `taint.Has()` returns false for
the 10k+ URLs in the assay corpus.

## When Permissive should NOT be returned

A symbol that has a known value should return that value, not
Permissive. The spike intentionally returns Permissive for:

- Unresolvable `load("@external//...", "symbol")` — caller wants
  the eval to continue but can't supply the real value.
- Stubbed top-level globals like `native`, `json` when the consumer
  doesn't wire them.

Things that should NOT be Permissive:
- `ctx.os.name` / `ctx.os.arch` — fork-controlled, must be a real
  String for impls to branch on.
- `module_ctx.modules` — caller-supplied real list of bazel_module.
- `Label("foo")` — must be a real Label value with `.workspace_root`
  etc.

If a consumer wants `Label()` to be permissive (e.g., scip-bazel
indexing partial code), that's a `Mode` decision — perhaps a
`ModeAnalysisLoose` variant later. Not in this plan.

## Production wishlist beyond the spike

These are explicit limitations to flag in the public docs:

1. **No symbolic both-branch exploration.** `if perm == X: A() else:
   B()` runs only B. Real-world impact: URLs in the A branch are
   missed. Mitigation: runtime interception (canopy's downloader-
   proxy) catches them at fetch time.
2. **No hashable Permissive.** `sdks[perm]` aborts the fork. Real-
   world impact: some `cc_register_toolchains`-style rules use
   permissive-derived keys. Mitigation: ForkError surfaces it.
3. **No TaintedString.** `str(perm)` produces regular String;
   subsequent type-checks see "string" not "tainted-string."
   Mitigation: marker substring covers most cases.
4. **No per-attribute taint.** If `ctx.attr.X` is Permissive but
   `ctx.attr.Y` is concrete, both are observed via the same
   `repository_attr` proxy; consumers can't easily distinguish.
   Mitigation: don't pass Permissive in attrs — supply real values.
5. **Performance under sustained load.** Spike runs 23 tests in
   ~140ms. Real production load (canopy ingesting a 50-module
   closure) is not yet benched. M5/M6 add benchmarks.

## Test pins (carry over from spike)

Each rule below corresponds to a pinning test that must continue to
pass through promotion:

| Rule | Spike test | Upstream test (proposed) |
|---|---|---|
| Direct Permissive → tainted "<unresolved>" | `TestSpike_Taint_DirectFromExecute` | `eval/invoke_test.go::TestInvokeRule_Taint_DirectPermissive` |
| Concat preserves prefix, marker-bearing | `TestSpike_Taint_BinaryPreservesPrefix` | `taint/marker_test.go::TestPermissiveBinaryPreservesPrefix` |
| str(perm) + concat → marker detected | `TestSpike_Taint_StrThenConcat` | `taint/marker_test.go::TestMarkerSurvivesStr` |
| format(perm) → marker detected | `TestSpike_Taint_FormatWithPerm` | `taint/marker_test.go::TestMarkerSurvivesFormat` |
| Cross-type EQ → else branch, no error | `TestSpike_Taint_CompareWithPermFallsThrough` | `stub/permissive_test.go::TestPermissiveCompareCrossType` |
| Same-type EQ → false | `TestSpike_Taint_PermVsPermComparesUnequal` | `stub/permissive_test.go::TestPermissiveCompareSameType` |
| Per-fork: pre-execute clean | `TestSpike_Taint_DownloadBeforeExecuteIsClean` | `ctx/repository_ctx_test.go::TestRepoCtxTaintOrder_BeforeIsClean` |
| Per-fork: post-execute tainted | `TestSpike_Taint_DownloadAfterExecuteIsTainted` | `ctx/repository_ctx_test.go::TestRepoCtxTaintOrder_AfterIsTainted` |
| Mixed list → all tainted | `TestSpike_Taint_MixedList` | `taint/sink_test.go::TestMixedListAllTainted` |
| ctx.read taints | `TestSpike_Taint_ReadTaints` | `ctx/repository_ctx_test.go::TestRepoCtxReadTaints` |

## Divergence from real Bazel behavior (worth flagging)

**Real Bazel rejects `_repo_rule(...)` calls outside `module_extension`
impls** with `EvalException("repo rules can only be called from within
module extension impl functions")` — verified at
`bazel/src/main/java/com/google/devtools/build/lib/bazel/repository/starlark/StarlarkRepositoryModule.java:166-174`.

The spike's `RepoRuleClass.CallInternal` silently returns `None` when
called outside an extension (no `instSinkKey` in `thread.Local`).
This was deliberate for the spike (allows synthetic tests that drive
a repo_rule directly) but diverges from Bazel.

The production decision: in `ModeStrict`, match Bazel — error on
top-level repo_rule instantiation. In `ModeLenient` / `ModeAnalysis`,
accept the call and capture the instantiation (current spike
behavior). Document on the API.

## When to revisit this contract

- A real consumer files an issue saying "Permissive eats my real
  value." Investigate; the contract may be too broad in some
  interface.
- A new Bazel version adds an operator we don't propagate (matrix
  ops, etc.). Add to Binary.
- Symbolic execution becomes feasible for a real use case. Separate
  plan.
