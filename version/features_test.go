package version_test

import (
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/albertocavalcante/starlark-go-bazel/version"
)

// HasFeature returns the curated table answer per Version.
func TestHasFeature_PerVersion(t *testing.T) {
	cases := []struct {
		v       version.Version
		feature version.Feature
		want    bool
	}{
		// Bazel 7+ features.
		{version.V7, version.FeatureBzlmodDefault, true},
		{version.V7, version.FeatureUseRepoRule, true},
		{version.V7, version.FeatureModExtOsArchDependent, true},
		{version.V7, version.FeatureNativeRepoName, true},
		{version.V7, version.FeatureExtensionMetadataFunc, true},

		// Bazel 7 should NOT have Bazel-8 features.
		{version.V7, version.FeatureCtxGetenv, false},
		{version.V7, version.FeatureCtxWatch, false},
		{version.V7, version.FeatureSymbolicMacros, false},

		// Bazel 8 has Bazel-8 features.
		{version.V8, version.FeatureCtxGetenv, true},
		{version.V8, version.FeatureCtxWatch, true},
		{version.V8, version.FeatureSymbolicMacros, true},

		// Bazel 8 should NOT have Bazel-9 features.
		{version.V8, version.FeatureRepoMetadata, false},
		{version.V8, version.FeatureModExtMetadataFacts, false},

		// Bazel 9 has everything.
		{version.V9, version.FeatureRepoMetadata, true},
		{version.V9, version.FeatureModExtMetadataFacts, true},
		{version.V9, version.FeatureBzlmodDefault, true},

		// VLatest resolves to V9.
		{version.VLatest, version.FeatureRepoMetadata, true},
		{version.VLatest, version.FeatureModExtMetadataFacts, true},
	}
	for _, c := range cases {
		got := c.v.HasFeature(c.feature)
		if got != c.want {
			t.Errorf("HasFeature(%s, %s) = %v, want %v", c.v, c.feature, got, c.want)
		}
	}
}

// AllFeatures returns every defined Feature constant.
func TestAllFeatures(t *testing.T) {
	all := version.AllFeatures()
	if len(all) < 10 {
		t.Errorf("AllFeatures returned %d, want at least 10", len(all))
	}
	seen := map[version.Feature]bool{}
	for _, f := range all {
		if seen[f] {
			t.Errorf("duplicate feature in AllFeatures: %s", f)
		}
		seen[f] = true
	}
}

// AsDeltas projects HasFeature into the imperative struct shape.
func TestAsDeltas_ConsistentWithHasFeature(t *testing.T) {
	cases := []version.Version{version.V7, version.V8, version.V9, version.VLatest}
	for _, v := range cases {
		d := v.AsDeltas()
		if d.BzlmodDefault != v.HasFeature(version.FeatureBzlmodDefault) {
			t.Errorf("%s: BzlmodDefault mismatch", v)
		}
		if d.CtxHasGetenv != v.HasFeature(version.FeatureCtxGetenv) {
			t.Errorf("%s: CtxHasGetenv mismatch", v)
		}
		if d.CtxHasRepoMetadata != v.HasFeature(version.FeatureRepoMetadata) {
			t.Errorf("%s: CtxHasRepoMetadata mismatch", v)
		}
	}
}

// Bazel-9-specific delta: AllowWORKSPACE is false (default-off via
// --enable_workspace=false in 9.0.0-pre.20250121.1).
func TestAsDeltas_WORKSPACEOffAtV9(t *testing.T) {
	if version.V9.AsDeltas().AllowWORKSPACE {
		t.Error("V9 should default WORKSPACE off")
	}
	if !version.V7.AsDeltas().AllowWORKSPACE {
		t.Error("V7 should allow WORKSPACE")
	}
}

// ExperimentalFlag.DefaultAt: all experimental flags default off.
func TestExperimentalFlag_DefaultsOff(t *testing.T) {
	flags := []version.ExperimentalFlag{
		version.FlagRepoCtxExecuteWasm,
		version.FlagRepoRemoteExec,
		version.FlagIsolatedExtensionUse,
		version.FlagNoImplicitWatchLabel,
		version.FlagEnableDeprecatedLabelAPIs,
	}
	for _, f := range flags {
		for _, v := range []version.Version{version.V7, version.V8, version.V9} {
			if f.DefaultAt(v) {
				t.Errorf("Flag %s defaults true at %s, want false", f, v)
			}
		}
	}
}

// BazelFeaturesValue returns a struct with the right shape — top
// fields are subgroup structs, leaf fields are booleans matching
// HasFeature.
func TestBazelFeaturesValue_StructShape(t *testing.T) {
	v := version.BazelFeaturesValue(version.V9)
	s, ok := v.(*starlarkstruct.Struct)
	if !ok {
		t.Fatalf("BazelFeaturesValue returned %T, want *starlarkstruct.Struct", v)
	}
	// external_deps subgroup should exist.
	ed, err := s.Attr("external_deps")
	if err != nil || ed == nil {
		t.Fatalf("external_deps subgroup missing: %v", err)
	}
	edStruct, ok := ed.(*starlarkstruct.Struct)
	if !ok {
		t.Fatalf("external_deps not a struct: %T", ed)
	}
	// module_extension_has_os_arch_dependent should be True at V9.
	leaf, err := edStruct.Attr("module_extension_has_os_arch_dependent")
	if err != nil {
		t.Fatal(err)
	}
	if leaf != starlark.True {
		t.Errorf("v9.external_deps.module_extension_has_os_arch_dependent = %v, want True", leaf)
	}
}

// BazelFeaturesValue at V7 reports V8/V9 features as False (correct
// negative answers).
func TestBazelFeaturesValue_V7HidesNewerFeatures(t *testing.T) {
	v := version.BazelFeaturesValue(version.V7).(*starlarkstruct.Struct)
	ed, _ := v.Attr("external_deps")
	edStruct := ed.(*starlarkstruct.Struct)

	leaf, _ := edStruct.Attr("repository_ctx_getenv")
	if leaf != starlark.False {
		t.Errorf("v7.external_deps.repository_ctx_getenv = %v, want False", leaf)
	}
	leaf, _ = edStruct.Attr("module_extension_metadata_facts")
	if leaf != starlark.False {
		t.Errorf("v7.external_deps.module_extension_metadata_facts = %v, want False", leaf)
	}
}

// End-to-end: a .bzl file that loads @bazel_features//:features.bzl
// and gates behavior on a feature flag evaluates correctly under the
// synthetic loader. (Top-level if isn't allowed in plain Starlark; the
// gating runs inside a function the caller invokes.)
func TestBazelFeaturesLoader_GatesEvalBranch(t *testing.T) {
	src := []byte(`
load("@bazel_features//:features.bzl", "bazel_features")
def gate():
    if bazel_features.external_deps.module_extension_has_os_arch_dependent:
        return "have-os-arch"
    return "no-os-arch"

answer = gate()
`)
	loader := version.BazelFeaturesLoader(version.V8, nil)
	thread := &starlark.Thread{Name: "test", Load: loader}
	globals, err := starlark.ExecFile(thread, "test.bzl", src, nil)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	s, _ := starlark.AsString(globals["answer"])
	if s != "have-os-arch" {
		t.Errorf("answer = %q, want have-os-arch (V8 should report os/arch)", s)
	}
}

// V7 evaluation of a V8-only feature takes the else branch.
func TestBazelFeaturesLoader_V7TakesNegativeBranchForV8Feature(t *testing.T) {
	src := []byte(`
load("@bazel_features//:features.bzl", "bazel_features")
def gate():
    if bazel_features.external_deps.repository_ctx_getenv:
        return "has-getenv"
    return "no-getenv"

answer = gate()
`)
	loader := version.BazelFeaturesLoader(version.V7, nil)
	thread := &starlark.Thread{Name: "test", Load: loader}
	globals, err := starlark.ExecFile(thread, "test.bzl", src, nil)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	s, _ := starlark.AsString(globals["answer"])
	if s != "no-getenv" {
		t.Errorf("answer = %q, want no-getenv at V7", s)
	}
}

// Loader delegates to `next` for non-bazel_features modules.
func TestBazelFeaturesLoader_DelegatesToNext(t *testing.T) {
	calledWith := ""
	next := func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
		calledWith = module
		return starlark.StringDict{"x": starlark.MakeInt(42)}, nil
	}
	loader := version.BazelFeaturesLoader(version.V9, next)
	out, err := loader(&starlark.Thread{Name: "t"}, "//some/other:thing.bzl")
	if err != nil {
		t.Fatal(err)
	}
	if calledWith != "//some/other:thing.bzl" {
		t.Errorf("next called with %q", calledWith)
	}
	if out["x"] != starlark.MakeInt(42) {
		t.Errorf("delegated value not returned: %v", out)
	}
}

// Loader without next returns (nil, nil) for unhandled modules.
func TestBazelFeaturesLoader_NoNextReturnsNil(t *testing.T) {
	loader := version.BazelFeaturesLoader(version.V9, nil)
	out, err := loader(&starlark.Thread{Name: "t"}, "//other:foo.bzl")
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Errorf("expected nil for unhandled module, got %v", out)
	}
}

// Smoke test that featuresMatchUpstreamForV9 would be added in a
// follow-up. The bazel-features-bzl source isn't checked out at
// ~/dev/refs today (BCR has source.json pointing at GitHub
// release tarballs only). When that's available, this is the
// canonical CI gate.
func TestFeaturesMatchUpstreamForV9_Skipped(t *testing.T) {
	t.Skip("bazel-features-bzl source not checked out; see plan 06-risks Q7. " +
		"To enable: clone github.com/bazel-contrib/bazel_features into ~/dev/refs, " +
		"then this test diffs the AllFeatures table against the upstream features.bzl.")
}

// Ordering contract — pinned 2026-05-31 (plan 02 §2.A6).
// V7 < V8 < V9 must hold numerically so `if v >= V8` etc. behave as
// a Bazel engineer would expect. VLatest is an alias for V9 (today),
// so `v >= V8` includes VLatest without needing a disjunction.
func TestVersion_OrderingComparable(t *testing.T) {
	if version.V7 >= version.V8 || version.V8 >= version.V9 {
		t.Errorf("expected V7 < V8 < V9; got V7=%d V8=%d V9=%d",
			int(version.V7), int(version.V8), int(version.V9))
	}
}

func TestVersion_VLatestEqualsV9(t *testing.T) {
	if version.VLatest != version.V9 {
		t.Errorf("VLatest = %d (V%d), want V9 (alias contract)",
			int(version.VLatest), int(version.VLatest))
	}
}

func TestVersion_AtLeastV8Inclusive(t *testing.T) {
	cases := []struct {
		v         version.Version
		atLeastV8 bool
	}{
		{version.V7, false},
		{version.V8, true},
		{version.V9, true},
		{version.VLatest, true},
	}
	for _, c := range cases {
		got := c.v >= version.V8
		if got != c.atLeastV8 {
			t.Errorf("%v >= V8 = %v, want %v", c.v, got, c.atLeastV8)
		}
	}
}
