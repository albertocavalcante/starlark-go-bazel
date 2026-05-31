package builtins_test

import (
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/bzl"
	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// repository_rule captures a minimal definition and assigns the
// identifier name post-eval.
func TestRepositoryRule_Minimal(t *testing.T) {
	src := `
def _impl(ctx):
    pass

my_repo = repository_rule(implementation = _impl)
`
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	v, ok := res.Globals["my_repo"]
	if !ok {
		t.Fatal("my_repo not in globals")
	}
	rc, ok := v.(*types.RepositoryRuleClass)
	if !ok {
		t.Fatalf("got %T, want *types.RepositoryRuleClass", v)
	}
	if rc.Name() != "my_repo" {
		t.Errorf("Name = %q, want my_repo", rc.Name())
	}
	if rc.Implementation() == nil {
		t.Error("Implementation is nil")
	}
}

// repository_rule captures every documented kwarg.
func TestRepositoryRule_AllKwargs(t *testing.T) {
	src := `
def _impl(ctx):
    pass

my_repo = repository_rule(
    implementation = _impl,
    attrs = {"version": attr.string(), "urls": attr.string_list()},
    local = True,
    environ = ["MY_TOKEN", "OTHER"],
    configure = True,
    remotable = False,
    doc = "Test rule.",
)
`
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	rc := res.Globals["my_repo"].(*types.RepositoryRuleClass)
	if !rc.Local() {
		t.Error("Local = false, want true")
	}
	if !rc.Configure() {
		t.Error("Configure = false, want true")
	}
	if rc.Remotable() {
		t.Error("Remotable = true, want false")
	}
	if env := rc.Environ(); len(env) != 2 || env[0] != "MY_TOKEN" || env[1] != "OTHER" {
		t.Errorf("Environ = %v, want [MY_TOKEN OTHER]", env)
	}
	if rc.Doc() != "Test rule." {
		t.Errorf("Doc = %q", rc.Doc())
	}
	attrs := rc.Attrs()
	if _, ok := attrs["version"]; !ok {
		t.Error("attrs[version] missing")
	}
	if _, ok := attrs["urls"]; !ok {
		t.Error("attrs[urls] missing")
	}
}

// repository_rule rejects positional args (matches Bazel).
func TestRepositoryRule_RejectsPositional(t *testing.T) {
	src := `
def _impl(ctx):
    pass

my_repo = repository_rule(_impl)
`
	interp := bzl.New(bzl.Options{})
	_, err := interp.Eval("test.bzl", []byte(src))
	if err == nil {
		t.Error("expected error for positional argument")
	}
}

// repository_rule rejects non-callable implementation.
func TestRepositoryRule_RejectsNonCallableImpl(t *testing.T) {
	src := `my_repo = repository_rule(implementation = "not a callable")`
	interp := bzl.New(bzl.Options{})
	_, err := interp.Eval("test.bzl", []byte(src))
	if err == nil {
		t.Error("expected error for non-callable implementation")
	}
}
