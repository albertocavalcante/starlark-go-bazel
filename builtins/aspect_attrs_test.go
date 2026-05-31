package builtins_test

import (
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/builtins"
	"github.com/albertocavalcante/starlark-go-bazel/bzl"
)

// Plan 07 T1: aspect(attrs = {...}) accepts the values produced by
// the eval-side attr.* module (which wraps *types.AttrDescriptor in
// an *attrDescriptorValue implementing types.AttrDescriptorHolder).
//
// Before plan 07's holder migration, builtins/aspect.go's consumer
// site type-asserts against *builtins.AttrDescriptor and rejects the
// eval-side wrapper with "got Attribute for ...".
func TestAspect_AttrsRoundTrip(t *testing.T) {
	src := []byte(`
def _impl(target, ctx):
    pass

my_aspect = aspect(
    implementation = _impl,
    attrs = {
        "_lib": attr.label(default = Label("//:lib")),
        "level": attr.int(),
    },
)
`)
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("defs.bzl", src)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	v, ok := res.Globals["my_aspect"]
	if !ok {
		t.Fatal("my_aspect not in globals")
	}
	asp, ok := v.(*builtins.AspectClass)
	if !ok {
		t.Fatalf("got %T, want *builtins.AspectClass", v)
	}
	attrs := asp.Attrs()
	if got, want := len(attrs), 2; got != want {
		t.Fatalf("Attrs len = %d, want %d", got, want)
	}
	if got, want := string(attrs["_lib"].Type), "label"; got != want {
		t.Errorf("_lib.Type = %q, want %q", got, want)
	}
	if got, want := string(attrs["level"].Type), "int"; got != want {
		t.Errorf("level.Type = %q, want %q", got, want)
	}
}
