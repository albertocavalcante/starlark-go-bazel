package ctx_test

import (
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/ctx"
	"github.com/albertocavalcante/starlark-go-bazel/version"
	"go.starlark.net/starlark"
)

// module_ctx.modules surfaces ModuleSpec entries as bazel_module
// structs with name, version, is_root, is_dev_dependency, tags.
func TestModuleCtx_ModulesSurface(t *testing.T) {
	mctx := ctx.NewModuleCtx(ctx.ModuleCtxOptions{
		Modules: []ctx.ModuleSpec{
			{
				Name: "root", Version: "0.0.0", IsRoot: true,
				Tags: map[string][]ctx.TagInstance{
					"download": {
						{Attrs: map[string]starlark.Value{"version": starlark.String("1.21.0")}},
					},
				},
			},
		},
	})
	src := `
def impl(mctx):
    mod = mctx.modules[0]
    tag = mod.tags.download[0]
    return (mod.name, mod.version, mod.is_root, tag.version)
`
	thread := &starlark.Thread{Name: "test"}
	globals, err := starlark.ExecFile(thread, "test.bzl", src, nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := starlark.Call(thread, globals["impl"].(starlark.Callable), starlark.Tuple{mctx}, nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	tup := out.(starlark.Tuple)
	if s, _ := starlark.AsString(tup.Index(0)); s != "root" {
		t.Errorf("name = %q", s)
	}
	if s, _ := starlark.AsString(tup.Index(1)); s != "0.0.0" {
		t.Errorf("version = %q", s)
	}
	if tup.Index(2) != starlark.True {
		t.Errorf("is_root = %v", tup.Index(2))
	}
	if s, _ := starlark.AsString(tup.Index(3)); s != "1.21.0" {
		t.Errorf("tag.version = %q", s)
	}
}

// Multiple modules + multiple tag instances each surface separately.
func TestModuleCtx_MultiModuleMultiTag(t *testing.T) {
	mctx := ctx.NewModuleCtx(ctx.ModuleCtxOptions{
		Modules: []ctx.ModuleSpec{
			{Name: "root", IsRoot: true, Tags: map[string][]ctx.TagInstance{
				"use": {{Attrs: map[string]starlark.Value{"v": starlark.String("a")}}},
			}},
			{Name: "dep", IsRoot: false, Tags: map[string][]ctx.TagInstance{
				"use": {
					{Attrs: map[string]starlark.Value{"v": starlark.String("b")}},
					{Attrs: map[string]starlark.Value{"v": starlark.String("c")}},
				},
			}},
		},
	})
	src := `
def impl(mctx):
    out = []
    for mod in mctx.modules:
        for tag in mod.tags.use:
            out.append(mod.name + ":" + tag.v)
    return out
`
	thread := &starlark.Thread{Name: "test"}
	globals, err := starlark.ExecFile(thread, "test.bzl", src, nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := starlark.Call(thread, globals["impl"].(starlark.Callable), starlark.Tuple{mctx}, nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	list := out.(*starlark.List)
	if list.Len() != 3 {
		t.Fatalf("result len = %d, want 3", list.Len())
	}
}

// Version-gated module_ctx attrs.
func TestModuleCtx_VersionGating(t *testing.T) {
	cases := []struct {
		ver  version.Version
		attr string
		want bool
	}{
		{version.V7, "getenv", false},
		{version.V7, "facts", false},
		{version.V8, "getenv", true},
		{version.V8, "facts", false},
		{version.V9, "facts", true},
		{version.VLatest, "getenv", true},
		{version.VLatest, "facts", true},
	}
	for _, c := range cases {
		mctx := ctx.NewModuleCtx(ctx.ModuleCtxOptions{Version: c.ver})
		v, _ := mctx.Attr(c.attr)
		got := v != nil
		if got != c.want {
			t.Errorf("Version=%s Attr(%q) present=%v, want %v", c.ver, c.attr, got, c.want)
		}
	}
}
