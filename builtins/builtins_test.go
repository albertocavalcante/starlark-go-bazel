package builtins

import (
	"testing"

	"go.starlark.net/starlark"
)

// TestPredeclared verifies that all predeclared builtins are properly exported.
func TestPredeclared(t *testing.T) {
	predeclared := Predeclared()

	expectedBuiltins := []string{
		"rule",
		"provider",
		"aspect",
		"select",
		"struct",
		"depset",
		"Label",
		"attr",
	}

	for _, name := range expectedBuiltins {
		if _, ok := predeclared[name]; !ok {
			t.Errorf("Predeclared() missing builtin %q", name)
		}
	}
}

// TestStruct verifies struct() behavior.
func TestStruct(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	// Test basic struct creation
	code := `s = struct(x = 1, y = "hello")`
	globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	s := globals["s"].(*Struct)
	if x, ok := s.Get("x"); !ok || x != starlark.MakeInt(1) {
		t.Errorf("struct.x = %v, want 1", x)
	}
	if y, ok := s.Get("y"); !ok || y != starlark.String("hello") {
		t.Errorf("struct.y = %v, want \"hello\"", y)
	}
}

// TestSelect verifies select() behavior.
func TestSelect(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	// Test basic select creation
	code := `s = select({"//conditions:default": ["a", "b"]})`
	globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	s := globals["s"].(*SelectorList)
	if len(s.Elements()) != 1 {
		t.Errorf("select has %d elements, want 1", len(s.Elements()))
	}

	// Test select with no_match_error
	code2 := `s = select({"//a:b": 1}, no_match_error = "custom error")`
	globals2, err := starlark.ExecFile(thread, "test.bzl", code2, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	s2 := globals2["s"].(*SelectorList)
	selector := s2.Elements()[0].(*SelectorValue)
	if selector.NoMatchError() != "custom error" {
		t.Errorf("no_match_error = %q, want \"custom error\"", selector.NoMatchError())
	}
}

// TestSelectEmpty verifies that empty select is rejected.
func TestSelectEmpty(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	code := `s = select({})`
	_, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err == nil {
		t.Error("expected error for empty select, got none")
	}
}

// TestDepset verifies depset() behavior.
func TestDepset(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	// Test basic depset creation
	code := `d = depset([1, 2, 3])`
	globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	d := globals["d"].(*Depset)
	if d.Order() != "default" {
		t.Errorf("depset order = %q, want \"default\"", d.Order())
	}

	list := d.ToList()
	if len(list) != 3 {
		t.Errorf("depset has %d elements, want 3", len(list))
	}
}

// TestDepsetOrder verifies depset order parameter.
func TestDepsetOrder(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	for _, order := range ValidDepsetOrders {
		code := `d = depset([1], order = "` + order + `")`
		globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
		if err != nil {
			t.Fatalf("ExecFile failed for order %q: %v", order, err)
		}

		d := globals["d"].(*Depset)
		if d.Order() != order {
			t.Errorf("depset order = %q, want %q", d.Order(), order)
		}
	}
}

// TestDepsetInvalidOrder verifies that invalid order is rejected.
func TestDepsetInvalidOrder(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	code := `d = depset([1], order = "invalid")`
	_, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err == nil {
		t.Error("expected error for invalid order, got none")
	}
}

// TestProvider verifies provider() behavior.
func TestProvider(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	// Test basic provider creation
	code := `MyInfo = provider()`
	globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	if globals["MyInfo"] == nil {
		t.Error("provider not created")
	}
}

// TestProviderWithFields verifies provider with fields restriction.
func TestProviderWithFields(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	code := `
MyInfo = provider(fields = ["x", "y"])
info = MyInfo(x = 1, y = 2)
`
	globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	if globals["info"] == nil {
		t.Error("provider instance not created")
	}
}

// TestProviderWithInit verifies provider with init callback.
func TestProviderWithInit(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	code := `
def _init(value):
    return {"doubled": value * 2}

MyInfo, _new_myinfo = provider(init = _init)
info = MyInfo(5)
raw = _new_myinfo(doubled = 10)
`
	globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	if globals["MyInfo"] == nil {
		t.Error("provider not created")
	}
	if globals["_new_myinfo"] == nil {
		t.Error("raw constructor not created")
	}
}

// TestAttrModule verifies the attr module.
func TestAttrModule(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	code := `
s = attr.string()
i = attr.int()
b = attr.bool()
l = attr.label()
ll = attr.label_list()
sl = attr.string_list()
`
	globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	for _, name := range []string{"s", "i", "b", "l", "ll", "sl"} {
		if globals[name] == nil {
			t.Errorf("attr.%s not created", name)
		}
	}
}

// TestRule verifies rule() behavior.
func TestRule(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	code := `
def _impl(ctx):
    pass

my_rule = rule(
    implementation = _impl,
    attrs = {
        "srcs": attr.label_list(),
        "deps": attr.label_list(),
    },
)
`
	globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	r := globals["my_rule"].(*RuleClass)
	if r == nil {
		t.Fatal("rule not created")
	}

	attrs := r.Attrs()
	if _, ok := attrs["srcs"]; !ok {
		t.Error("rule missing 'srcs' attribute")
	}
	if _, ok := attrs["deps"]; !ok {
		t.Error("rule missing 'deps' attribute")
	}
}

// TestAspect verifies aspect() behavior.
func TestAspect(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := Predeclared()

	code := `
def _impl(target, ctx):
    pass

my_aspect = aspect(
    implementation = _impl,
    attr_aspects = ["deps"],
)
`
	globals, err := starlark.ExecFile(thread, "test.bzl", code, predeclared)
	if err != nil {
		t.Fatalf("ExecFile failed: %v", err)
	}

	a := globals["my_aspect"].(*AspectClass)
	if a == nil {
		t.Fatal("aspect not created")
	}

	if len(a.AttrAspects()) != 1 || a.AttrAspects()[0] != "deps" {
		t.Errorf("aspect attr_aspects = %v, want [deps]", a.AttrAspects())
	}
}
