package builtins_test

import (
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/bzl"
	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// tag_class captures attr schema + doc.
func TestTagClass_Minimal(t *testing.T) {
	src := `
my_tag = tag_class(
    attrs = {
        "version": attr.string(),
        "sha256": attr.string(),
    },
    doc = "Test tag",
)
`
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	tc, ok := res.Globals["my_tag"].(*types.TagClass)
	if !ok {
		t.Fatalf("got %T, want *types.TagClass", res.Globals["my_tag"])
	}
	if tc.Doc() != "Test tag" {
		t.Errorf("Doc = %q", tc.Doc())
	}
	attrs := tc.Attrs()
	if _, ok := attrs["version"]; !ok {
		t.Error("attrs[version] missing")
	}
	if _, ok := attrs["sha256"]; !ok {
		t.Error("attrs[sha256] missing")
	}
}

// tag_class accepts empty attrs.
func TestTagClass_EmptyAttrs(t *testing.T) {
	src := `my_tag = tag_class()`
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	tc, ok := res.Globals["my_tag"].(*types.TagClass)
	if !ok {
		t.Fatal("my_tag not a tag_class")
	}
	if len(tc.Attrs()) != 0 {
		t.Errorf("attrs = %v, want empty", tc.Attrs())
	}
}
