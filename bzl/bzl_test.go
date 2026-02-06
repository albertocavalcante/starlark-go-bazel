package bzl

import (
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/types"
)

func TestBasicEval(t *testing.T) {
	interp := New(Options{})
	result, err := interp.Eval("test.bzl", []byte(`
my_provider = provider(fields = ["x", "y"])
`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Globals["my_provider"]; !ok {
		t.Error("expected my_provider in globals")
	}
}

func TestProviderWithDoc(t *testing.T) {
	interp := New(Options{})
	result, err := interp.Eval("test.bzl", []byte(`
MyInfo = provider(
    doc = "My custom provider",
    fields = ["value", "deps"],
)
`))
	if err != nil {
		t.Fatal(err)
	}

	val, ok := result.Globals["MyInfo"]
	if !ok {
		t.Fatal("expected MyInfo in globals")
	}

	prov, ok := val.(*types.Provider)
	if !ok {
		t.Fatalf("expected *types.Provider, got %T", val)
	}

	if prov.Doc() != "My custom provider" {
		t.Errorf("expected doc 'My custom provider', got %q", prov.Doc())
	}

	fields := prov.Fields()
	if len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}
}

func TestRuleDefinition(t *testing.T) {
	interp := New(Options{})
	result, err := interp.Eval("test.bzl", []byte(`
def _impl(ctx):
    pass

my_rule = rule(
    implementation = _impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "deps": attr.label_list(),
    },
)
`))
	if err != nil {
		t.Fatal(err)
	}

	val, ok := result.Globals["my_rule"]
	if !ok {
		t.Fatal("expected my_rule in globals")
	}

	rc, ok := val.(*types.RuleClass)
	if !ok {
		t.Fatalf("expected *types.RuleClass, got %T", val)
	}

	attrs := rc.Attrs()
	if _, ok := attrs["srcs"]; !ok {
		t.Error("expected 'srcs' attribute")
	}
	if _, ok := attrs["deps"]; !ok {
		t.Error("expected 'deps' attribute")
	}
}

func TestLabelParsing(t *testing.T) {
	interp := New(Options{})
	result, err := interp.Eval("test.bzl", []byte(`
label1 = Label("//pkg:target")
label2 = Label("@repo//pkg:target")
`))
	if err != nil {
		t.Fatal(err)
	}

	label1, ok := result.Globals["label1"].(*types.Label)
	if !ok {
		t.Fatal("expected label1 to be a Label")
	}
	if label1.Pkg() != "pkg" || label1.Name() != "target" {
		t.Errorf("label1 parsed incorrectly: %s", label1)
	}

	label2, ok := result.Globals["label2"].(*types.Label)
	if !ok {
		t.Fatal("expected label2 to be a Label")
	}
	if label2.Repo() != "repo" {
		t.Errorf("expected repo 'repo', got %q", label2.Repo())
	}
}

func TestPrintHandler(t *testing.T) {
	var printed string
	interp := New(Options{
		PrintHandler: func(msg string) {
			printed = msg
		},
	})

	_, err := interp.Eval("test.bzl", []byte(`
print("hello world")
`))
	if err != nil {
		t.Fatal(err)
	}

	if printed != "hello world" {
		t.Errorf("expected 'hello world', got %q", printed)
	}
}

func TestStructCreation(t *testing.T) {
	interp := New(Options{})
	result, err := interp.Eval("test.bzl", []byte(`
s = struct(x = 1, y = "hello")
`))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := result.Globals["s"]; !ok {
		t.Error("expected 's' in globals")
	}
}

func TestDepsetCreation(t *testing.T) {
	interp := New(Options{})
	result, err := interp.Eval("test.bzl", []byte(`
d = depset(["a", "b", "c"])
`))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := result.Globals["d"]; !ok {
		t.Error("expected 'd' in globals")
	}

	depset, ok := result.Globals["d"].(*types.Depset)
	if !ok {
		t.Fatalf("expected *types.Depset, got %T", result.Globals["d"])
	}

	list := depset.ToList()
	if len(list) != 3 {
		t.Errorf("expected 3 elements, got %d", len(list))
	}
}
