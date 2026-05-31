package builtins_test

import (
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/bzl"
	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// module_extension captures a minimal definition.
func TestModuleExtension_Minimal(t *testing.T) {
	src := `
def _impl(module_ctx):
    pass

my_ext = module_extension(implementation = _impl)
`
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	m, ok := res.Globals["my_ext"].(*types.ModuleExtensionClass)
	if !ok {
		t.Fatalf("got %T, want *types.ModuleExtensionClass", res.Globals["my_ext"])
	}
	if m.Name() != "my_ext" {
		t.Errorf("Name = %q", m.Name())
	}
	if m.OsDependent() || m.ArchDependent() {
		t.Error("os/arch_dependent should default to false")
	}
}

// module_extension captures all kwargs including tag_classes.
func TestModuleExtension_AllKwargs(t *testing.T) {
	src := `
def _impl(module_ctx):
    pass

download_tag = tag_class(attrs = {"version": attr.string()}, doc = "Download tag")

my_ext = module_extension(
    implementation = _impl,
    tag_classes = {"download": download_tag},
    doc = "Test extension.",
    environ = ["MY_TOKEN"],
    os_dependent = True,
    arch_dependent = True,
)
`
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	m := res.Globals["my_ext"].(*types.ModuleExtensionClass)
	if !m.OsDependent() {
		t.Error("OsDependent = false, want true")
	}
	if !m.ArchDependent() {
		t.Error("ArchDependent = false, want true")
	}
	if env := m.Environ(); len(env) != 1 || env[0] != "MY_TOKEN" {
		t.Errorf("Environ = %v", env)
	}
	if m.Doc() != "Test extension." {
		t.Errorf("Doc = %q", m.Doc())
	}
	tcs := m.TagClasses()
	tc, ok := tcs["download"]
	if !ok {
		t.Fatal("tag_classes[download] missing")
	}
	if tc.Name() != "download" {
		t.Errorf("tag_class name = %q, want download", tc.Name())
	}
}

// module_extension rejects positional args.
func TestModuleExtension_RejectsPositional(t *testing.T) {
	src := `
def _impl(module_ctx):
    pass

my_ext = module_extension(_impl)
`
	interp := bzl.New(bzl.Options{})
	_, err := interp.Eval("test.bzl", []byte(src))
	if err == nil {
		t.Error("expected error for positional argument")
	}
}
