package builtins_test

import (
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/bzl"
	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// Plan 07 T2: rule(attrs = {...}) threads the attr.* values into the
// resulting *types.RuleClass with their declared types preserved.
//
// Before the fix, types.RuleBuiltin stored a placeholder
// AttrDescriptor{Type: AttrTypeString} for every attr regardless of
// what attr.* returned — so attrs["srcs"].Type was "string", not
// "label_list".
func TestRule_AttrsStillWorks(t *testing.T) {
	src := []byte(`
def _impl(ctx):
    pass

my_rule = rule(
    implementation = _impl,
    attrs = {"srcs": attr.label_list(), "lib": attr.label()},
)
`)
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("defs.bzl", src)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	v, ok := res.Globals["my_rule"]
	if !ok {
		t.Fatal("my_rule not in globals")
	}
	rc, ok := v.(*types.RuleClass)
	if !ok {
		t.Fatalf("got %T, want *types.RuleClass", v)
	}
	attrs := rc.Attrs()
	if got, want := string(attrs["srcs"].Type), "label_list"; got != want {
		t.Errorf("srcs.Type = %q, want %q", got, want)
	}
	if got, want := string(attrs["lib"].Type), "label"; got != want {
		t.Errorf("lib.Type = %q, want %q", got, want)
	}
}
