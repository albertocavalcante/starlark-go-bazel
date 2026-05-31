package taint_test

import (
	"testing"

	"go.starlark.net/starlark"

	"github.com/albertocavalcante/starlark-go-bazel/stub"
	"github.com/albertocavalcante/starlark-go-bazel/taint"
)

// External test package so we can import stub (for stub.Shared) to
// exercise the Permissive-detection branch without an import cycle.

func TestFlattenURLs_Nil(t *testing.T) {
	urls, tainted := taint.FlattenURLs(nil)
	if urls != nil {
		t.Errorf("urls = %v, want nil", urls)
	}
	if tainted {
		t.Errorf("tainted = true, want false")
	}
}

func TestFlattenURLs_SingleString(t *testing.T) {
	urls, tainted := taint.FlattenURLs(starlark.String("https://x/v.tar.gz"))
	if len(urls) != 1 || urls[0] != "https://x/v.tar.gz" {
		t.Errorf("urls = %v, want [https://x/v.tar.gz]", urls)
	}
	if tainted {
		t.Errorf("tainted = true, want false")
	}
}

func TestFlattenURLs_SingleStringWithMarker(t *testing.T) {
	in := "https://x/" + taint.Marker + "/v.tar.gz"
	urls, tainted := taint.FlattenURLs(starlark.String(in))
	if len(urls) != 1 || urls[0] != in {
		t.Errorf("urls = %v, want [%q]", urls, in)
	}
	if !tainted {
		t.Errorf("tainted = false, want true (marker substring should propagate)")
	}
}

func TestFlattenURLs_List(t *testing.T) {
	list := starlark.NewList([]starlark.Value{starlark.String("a"), starlark.String("b")})
	urls, tainted := taint.FlattenURLs(list)
	if len(urls) != 2 || urls[0] != "a" || urls[1] != "b" {
		t.Errorf("urls = %v, want [a b]", urls)
	}
	if tainted {
		t.Errorf("tainted = true, want false")
	}
}

func TestFlattenURLs_ListWithMarker(t *testing.T) {
	withMarker := "https://" + taint.Marker + "/x"
	list := starlark.NewList([]starlark.Value{
		starlark.String("a"),
		starlark.String(withMarker),
		starlark.String("c"),
	})
	urls, tainted := taint.FlattenURLs(list)
	if len(urls) != 3 {
		t.Fatalf("len(urls) = %d, want 3", len(urls))
	}
	if urls[1] != withMarker {
		t.Errorf("urls[1] = %q, want %q", urls[1], withMarker)
	}
	if !tainted {
		t.Errorf("tainted = false, want true (one entry carries marker)")
	}
}

func TestFlattenURLs_PermissiveValue(t *testing.T) {
	urls, tainted := taint.FlattenURLs(stub.Shared)
	if len(urls) != 1 || urls[0] != "<unresolved>" {
		t.Errorf("urls = %v, want [<unresolved>]", urls)
	}
	if !tainted {
		t.Errorf("tainted = false, want true (Permissive value)")
	}
}

func TestFlattenURLs_ListContainingPermissive(t *testing.T) {
	list := starlark.NewList([]starlark.Value{
		starlark.String("a"),
		stub.Shared,
		starlark.String("c"),
	})
	urls, tainted := taint.FlattenURLs(list)
	if len(urls) != 3 {
		t.Fatalf("len(urls) = %d, want 3", len(urls))
	}
	if urls[0] != "a" || urls[1] != "<unresolved>" || urls[2] != "c" {
		t.Errorf("urls = %v, want [a <unresolved> c]", urls)
	}
	if !tainted {
		t.Errorf("tainted = false, want true (list contains Permissive)")
	}
}

func TestFlattenURLs_NonIterable(t *testing.T) {
	// An int isn't iterable, isn't a string, isn't Permissive. Should
	// return (nil, false) — the silent-skip policy.
	urls, tainted := taint.FlattenURLs(starlark.MakeInt(42))
	if urls != nil {
		t.Errorf("urls = %v, want nil for non-iterable non-string", urls)
	}
	if tainted {
		t.Errorf("tainted = true, want false")
	}
}

// DefaultPlatforms returns a fresh slice on each call so that a
// caller appending to one return doesn't leak into the next.
// (Currently a var slice with the trip-prone shared-backing-array
// property; func form locks the freshness contract.)
func TestDefaultPlatforms_ReturnsFreshSlice(t *testing.T) {
	first := taint.DefaultPlatforms()
	if len(first) == 0 {
		t.Fatal("DefaultPlatforms() returned empty slice")
	}
	// Mutate the first return; second call must be unaffected.
	first[0].OS = "MUTATED"
	second := taint.DefaultPlatforms()
	if second[0].OS == "MUTATED" {
		t.Errorf("second call OS = MUTATED, want the original value (slices share backing)")
	}
}
