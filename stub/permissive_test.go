package stub_test

import (
	"strings"
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/stub"
	"github.com/albertocavalcante/starlark-go-bazel/taint"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// Permissive satisfies the basic Starlark interface.
func TestPermissive_Value(t *testing.T) {
	p := stub.Shared
	if p.String() != taint.Marker {
		t.Errorf("String() = %q, want %q", p.String(), taint.Marker)
	}
	if p.Type() != "permissive" {
		t.Errorf("Type() = %q", p.Type())
	}
	if !bool(p.Truth()) {
		t.Error("Truth() = false, want true")
	}
	if _, err := p.Hash(); err == nil {
		t.Error("Hash() should error (unhashable)")
	}
}

// Permissive is a starlark.Callable; calling it returns Shared.
func TestPermissive_Callable(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	result, err := starlark.Call(thread, stub.Shared, nil, nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result != stub.Shared {
		t.Errorf("Call result = %v, want Shared", result)
	}
}

// Attr returns Shared for any name — chained access cascades.
func TestPermissive_AttrCascade(t *testing.T) {
	v, err := stub.Shared.Attr("any")
	if err != nil {
		t.Fatal(err)
	}
	if v != stub.Shared {
		t.Errorf("Attr returned %v, want Shared", v)
	}
	// Chained: x.y.z all return Shared without allocating.
	v2, _ := v.(starlark.HasAttrs).Attr("further")
	if v2 != stub.Shared {
		t.Errorf("Chained Attr returned %v, want Shared", v2)
	}
}

// Get implements Mapping — x[k] returns Shared.
func TestPermissive_Mapping(t *testing.T) {
	v, found, err := stub.Shared.Get(starlark.String("key"))
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("expected found=true")
	}
	if v != stub.Shared {
		t.Errorf("Get returned %v, want Shared", v)
	}
}

// Binary string concat preserves prefix via the marker. Tests both
// sides (Permissive on left vs right of +) and direction matters.
func TestPermissive_BinaryStringConcat(t *testing.T) {
	cases := []struct {
		name     string
		side     starlark.Side
		operand  starlark.Value
		expected string
	}{
		{"perm + str (Left)", starlark.Left, starlark.String("/foo"), taint.Marker + "/foo"},
		{"str + perm (Right)", starlark.Right, starlark.String("https://"), "https://" + taint.Marker},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := stub.Shared.Binary(syntax.PLUS, c.operand, c.side)
			if err != nil {
				t.Fatal(err)
			}
			s, ok := starlark.AsString(result)
			if !ok {
				t.Fatalf("result not a string: %v", result)
			}
			if s != c.expected {
				t.Errorf("Binary = %q, want %q", s, c.expected)
			}
		})
	}
}

// Non-string Binary operand falls back to Shared.
func TestPermissive_BinaryNonString(t *testing.T) {
	result, err := stub.Shared.Binary(syntax.PLUS, starlark.MakeInt(5), starlark.Left)
	if err != nil {
		t.Fatal(err)
	}
	if result != stub.Shared {
		t.Errorf("non-string Binary = %v, want Shared", result)
	}
}

// Non-PLUS op falls back to Shared.
func TestPermissive_BinaryNonPlus(t *testing.T) {
	result, err := stub.Shared.Binary(syntax.STAR, starlark.MakeInt(5), starlark.Left)
	if err != nil {
		t.Fatal(err)
	}
	if result != stub.Shared {
		t.Errorf("non-PLUS Binary = %v, want Shared", result)
	}
}

// Same-type EQ is conservative-false; NEQ true.
func TestPermissive_CompareSameType(t *testing.T) {
	cases := []struct {
		op    syntax.Token
		want  bool
		isErr bool
	}{
		{syntax.EQL, false, false},
		{syntax.NEQ, true, false},
		{syntax.LT, false, true},
		{syntax.LE, false, true},
		{syntax.GT, false, true},
		{syntax.GE, false, true},
	}
	for _, c := range cases {
		got, err := stub.Shared.CompareSameType(c.op, stub.Shared, 10)
		if c.isErr {
			if err == nil {
				t.Errorf("op %v: expected error", c.op)
			}
			continue
		}
		if err != nil {
			t.Errorf("op %v: unexpected error: %v", c.op, err)
		}
		if got != c.want {
			t.Errorf("op %v: got %v, want %v", c.op, got, c.want)
		}
	}
}

// Marker survives through .format() — embedded in any string that
// references Permissive via str() or format substitution.
func TestPermissive_MarkerSurvivesFormat(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := starlark.StringDict{"perm": stub.Shared}
	globals, err := starlark.ExecFile(thread, "test.bzl", `
result = "https://example.com/v{}/foo".format(perm)
`, predeclared)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	result := globals["result"]
	s, _ := starlark.AsString(result)
	if !taint.Has(s) {
		t.Errorf("format result %q does not contain marker", s)
	}
	if !strings.Contains(s, "https://example.com/v") {
		t.Errorf("format result %q lost the literal prefix", s)
	}
}

// String concat (the direct +) preserves prefix via Permissive.Binary.
func TestPermissive_MarkerSurvivesConcat(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := starlark.StringDict{"perm": stub.Shared}
	globals, err := starlark.ExecFile(thread, "test.bzl", `
result = "https://example.com/" + perm
`, predeclared)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	s, _ := starlark.AsString(globals["result"])
	if s != "https://example.com/"+taint.Marker {
		t.Errorf("concat result = %q", s)
	}
}

// Cross-type EQ (Permissive == "linux") goes through go.starlark.net's
// default Equal which returns false on type mismatch — without error.
// The else branch is reachable.
func TestPermissive_CrossTypeEqualFallsThrough(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	predeclared := starlark.StringDict{"perm": stub.Shared}
	globals, err := starlark.ExecFile(thread, "test.bzl", `
def check(p):
    if p == "linux":
        return "linux-branch"
    return "else-branch"
result = check(perm)
`, predeclared)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	s, _ := starlark.AsString(globals["result"])
	if s != "else-branch" {
		t.Errorf("result = %q, want else-branch (cross-type EQ should resolve to false)", s)
	}
}

// LoaderFor stubs requested symbols when tryReal doesn't supply them.
func TestLoaderFor_StubsMissing(t *testing.T) {
	symbols := map[string][]string{
		"@external//:foo.bzl": {"helper", "CONST"},
	}
	loader := stub.LoaderFor(symbols, nil)
	out, err := loader(&starlark.Thread{Name: "t"}, "@external//:foo.bzl")
	if err != nil {
		t.Fatal(err)
	}
	if out["helper"] != stub.Shared {
		t.Errorf("helper not stubbed to Shared")
	}
	if out["CONST"] != stub.Shared {
		t.Errorf("CONST not stubbed to Shared")
	}
}

// LoaderFor honors tryReal results when supplied.
func TestLoaderFor_TryRealOverridesStub(t *testing.T) {
	symbols := map[string][]string{
		"@external//:foo.bzl": {"helper"},
	}
	customValue := starlark.MakeInt(42)
	tryReal := func(module string) (starlark.StringDict, bool) {
		return starlark.StringDict{"helper": customValue}, true
	}
	loader := stub.LoaderFor(symbols, tryReal)
	out, err := loader(&starlark.Thread{Name: "t"}, "@external//:foo.bzl")
	if err != nil {
		t.Fatal(err)
	}
	if out["helper"] != customValue {
		t.Errorf("helper = %v, want customValue", out["helper"])
	}
}

// LoaderFor fills in only-missing symbols when tryReal partially
// resolves.
func TestLoaderFor_TryRealPartial(t *testing.T) {
	symbols := map[string][]string{
		"@external//:foo.bzl": {"helper", "CONST"},
	}
	tryReal := func(module string) (starlark.StringDict, bool) {
		return starlark.StringDict{"helper": starlark.MakeInt(1)}, true
	}
	loader := stub.LoaderFor(symbols, tryReal)
	out, _ := loader(&starlark.Thread{Name: "t"}, "@external//:foo.bzl")
	if h := out["helper"]; h == stub.Shared {
		t.Error("helper should not be stubbed (tryReal supplied it)")
	}
	if out["CONST"] != stub.Shared {
		t.Error("CONST should be stubbed (tryReal didn't supply)")
	}
}

// ScanLoads extracts load() targets and From-side symbol names.
func TestScanLoads(t *testing.T) {
	src := []byte(`
load("@external//:foo.bzl", "helper", local_name = "remote_name")
load("//pkg:internal.bzl", "X")
`)
	f, err := syntax.LegacyFileOptions().Parse("test.bzl", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := stub.ScanLoads(f)
	if len(got["@external//:foo.bzl"]) != 2 ||
		got["@external//:foo.bzl"][0] != "helper" ||
		got["@external//:foo.bzl"][1] != "remote_name" {
		t.Errorf("external loads = %v", got["@external//:foo.bzl"])
	}
	if got["//pkg:internal.bzl"][0] != "X" {
		t.Errorf("internal load = %v", got["//pkg:internal.bzl"])
	}
}

// taint.FlattenURLs sees Permissive directly and tags as unresolved.
func TestPermissive_DetectedByFlattenURLs(t *testing.T) {
	urls, tainted := taint.FlattenURLs(stub.Shared)
	if !tainted {
		t.Error("FlattenURLs should taint on Permissive")
	}
	if len(urls) != 1 || urls[0] != "<unresolved>" {
		t.Errorf("urls = %v", urls)
	}
}

// Fuzz: Binary with random op tokens + random operand types should
// never panic. The function returns either a String (for PLUS+String
// fast path) or Shared.
func FuzzPermissive_Binary(f *testing.F) {
	f.Add(int(syntax.PLUS), "foo", true)  // standard string concat
	f.Add(int(syntax.STAR), "x", false)   // non-PLUS op
	f.Add(int(syntax.MINUS), "", true)    // empty string
	f.Add(int(syntax.PLUS), "", false)    // empty string PLUS
	f.Fuzz(func(t *testing.T, op int, s string, leftSide bool) {
		var side starlark.Side
		if leftSide {
			side = starlark.Left
		} else {
			side = starlark.Right
		}
		_, _ = stub.Shared.Binary(syntax.Token(op), starlark.String(s), side)
		// If we got here without panic, fuzz iteration succeeds.
	})
}

// Fuzz: CompareSameType with random op tokens. EQ/NEQ should never
// error; ordered ops should error but not panic.
func FuzzPermissive_CompareSameType(f *testing.F) {
	f.Add(int(syntax.EQL))
	f.Add(int(syntax.NEQ))
	f.Add(int(syntax.LT))
	f.Add(int(syntax.GE))
	f.Fuzz(func(t *testing.T, op int) {
		_, _ = stub.Shared.CompareSameType(syntax.Token(op), stub.Shared, 10)
	})
}
