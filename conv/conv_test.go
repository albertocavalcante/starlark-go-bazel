package conv

import (
	"reflect"
	"testing"

	"go.starlark.net/starlark"
)

func TestFromGo_PrimitiveTypes(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want starlark.Value
	}{
		{"string", "abc", starlark.String("abc")},
		{"bool true", true, starlark.True},
		{"bool false", false, starlark.False},
		{"integer-shaped float", float64(42), starlark.MakeInt(42)},
		{"actual float", float64(3.14), starlark.Float(3.14)},
		{"nil → None", nil, starlark.None},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FromGo(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("FromGo(%v) = %v (%T), want %v (%T)",
					c.in, got, got, c.want, c.want)
			}
		})
	}
}

func TestFromGo_ListRecursion(t *testing.T) {
	got := FromGo([]any{"a", "b", float64(7)})
	list, ok := got.(*starlark.List)
	if !ok {
		t.Fatalf("got %T, want *starlark.List", got)
	}
	if list.Len() != 3 {
		t.Errorf("Len = %d, want 3", list.Len())
	}
}

func TestFromGo_NestedMap(t *testing.T) {
	got := FromGo(map[string]any{
		"a": "x",
		"b": map[string]any{"nested": true},
	})
	dict, ok := got.(*starlark.Dict)
	if !ok {
		t.Fatalf("got %T, want *starlark.Dict", got)
	}
	if dict.Len() != 2 {
		t.Errorf("Len = %d, want 2", dict.Len())
	}
}

// Integer-shaped floats outside int32 range must not silently
// truncate. MakeInt(int(t)) would lose the upper 32 bits on 32-bit
// GOARCH for values like a unix-millis timestamp.
func TestFromGo_LargeInteger(t *testing.T) {
	const unixMillis = float64(1_700_000_000_000) // 13-digit; > MaxInt32
	got := FromGo(unixMillis)
	want := starlark.MakeInt64(1_700_000_000_000)
	if got.String() != want.String() {
		t.Errorf("FromGo(%v) = %s, want %s", unixMillis, got, want)
	}
}

// Unknown types fall back to starlark.None — see FromGo docstring.
func TestFromGo_UnknownTypeFallsBackToNone(t *testing.T) {
	type weirdType struct{ x int }
	got := FromGo(weirdType{x: 42})
	if got != starlark.None {
		t.Errorf("got %v (%T), want starlark.None", got, got)
	}
}
