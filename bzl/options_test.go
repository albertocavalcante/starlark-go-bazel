package bzl

import (
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/taint"
	"github.com/albertocavalcante/starlark-go-bazel/version"
	"go.starlark.net/starlark"
)

// Zero-value Options preserves prior behavior — existing callers
// pass Options{} and shouldn't see Version, Mode, or other new
// fields change semantics.
func TestOptions_ZeroValueDefaults(t *testing.T) {
	var o Options
	if got := o.effectiveMode(); got != ModeStrict {
		t.Errorf("effectiveMode() with zero value = %s, want ModeStrict", got)
	}
	// Zero-value Version is the unallocated iota (numerically 0);
	// Resolved() translates the zero to VLatest. Pin the resolution
	// contract.
	if got := o.Version.Resolved(); got != version.VLatest {
		t.Errorf("Version.Resolved() zero value = %v, want VLatest", got)
	}
}

// LenientLoad=true with no Mode should auto-promote to ModeLenient
// so legacy callers keep working.
func TestOptions_LenientLoadAutoPromotes(t *testing.T) {
	o := Options{LenientLoad: true}
	if got := o.effectiveMode(); got != ModeLenient {
		t.Errorf("effectiveMode() = %s, want ModeLenient (LenientLoad auto-promote)", got)
	}
}

// Explicit Mode wins over LenientLoad — caller has clearly intended
// the new API and we honor it.
func TestOptions_ExplicitModeWins(t *testing.T) {
	o := Options{LenientLoad: true, Mode: ModeAnalysis}
	if got := o.effectiveMode(); got != ModeAnalysis {
		t.Errorf("effectiveMode() = %s, want ModeAnalysis", got)
	}
}

// Custom predeclared globals injected via Options must be visible to
// the .bzl evaluation. Closes the spike's workaround that bypassed
// bzl.Interpreter to access eval.Options directly.
func TestOptions_PredeclaredBzlPropagates(t *testing.T) {
	called := false
	custom := starlark.NewBuiltin("custom_audit", func(t *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		called = true
		return starlark.None, nil
	})

	interp := New(Options{
		PredeclaredBzl: starlark.StringDict{
			"custom_audit": custom,
		},
	})
	_, err := interp.Eval("test.bzl", []byte(`custom_audit("hello")`))
	if err != nil {
		t.Fatalf("eval with custom predeclared: %v", err)
	}
	if !called {
		t.Errorf("custom_audit builtin was not invoked")
	}
}

// Version field is accepted (M6 fills in the per-version delta table;
// this test only confirms the field carries through).
func TestOptions_VersionFieldAccepted(t *testing.T) {
	for _, v := range []version.Version{version.VLatest, version.V7, version.V8, version.V9} {
		interp := New(Options{Version: v})
		if interp == nil {
			t.Errorf("New() returned nil for Version=%s", v)
		}
	}
}

// CaptureSinks pointer is accepted in ModeAnalysis. M5 wires actual
// capture; M1 only verifies the field shape.
func TestOptions_CaptureSinksAccepted(t *testing.T) {
	sinks := &taint.Sinks{}
	interp := New(Options{Mode: ModeAnalysis, CaptureSinks: sinks})
	if interp == nil {
		t.Fatal("New() returned nil")
	}
}

// FeatureFlags map is accepted. M6 wires per-flag behavior; M1 only
// verifies the field shape.
func TestOptions_FeatureFlagsAccepted(t *testing.T) {
	interp := New(Options{
		FeatureFlags: map[string]bool{
			"experimental_repository_ctx_execute_wasm": true,
		},
	})
	if interp == nil {
		t.Fatal("New() returned nil")
	}
}

// Mode.String() returns stable names for diagnostics.
func TestMode_String(t *testing.T) {
	cases := []struct {
		m    Mode
		want string
	}{
		{ModeStrict, "strict"},
		{ModeLenient, "lenient"},
		{ModeAnalysis, "analysis"},
	}
	for _, c := range cases {
		if got := c.m.String(); got != c.want {
			t.Errorf("Mode(%d).String() = %q, want %q", int(c.m), got, c.want)
		}
	}
}
