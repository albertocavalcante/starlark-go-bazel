package stub_test

import (
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/stub"
	"go.starlark.net/syntax"
)

// parseLoadStmts is the test scaffold: build a syntax.File with the
// given source so we can hand it to ScanLoads.
func parseLoadStmts(t *testing.T, src string) *syntax.File {
	t.Helper()
	opts := &syntax.FileOptions{}
	f, err := opts.Parse("test.bzl", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return f
}

func TestScanLoads_SingleLoadOneSymbol(t *testing.T) {
	f := parseLoadStmts(t, `load("@x//:y.bzl", "foo")`)
	got := stub.ScanLoads(f)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	syms, ok := got["@x//:y.bzl"]
	if !ok {
		t.Fatalf("got = %v, missing module key @x//:y.bzl", got)
	}
	if len(syms) != 1 || syms[0] != "foo" {
		t.Errorf("got[@x//:y.bzl] = %v, want [foo]", syms)
	}
}

func TestScanLoads_AliasedLoadStoresFromName(t *testing.T) {
	// `baz_local = "baz_remote"` — the local alias is `baz_local`, the
	// loaded module's name is `baz_remote`. ScanLoads stores the
	// FROM-side (the loaded-module symbol the permissive loader needs
	// to populate). Pin this contract.
	f := parseLoadStmts(t, `load("@x//:y.bzl", baz_local = "baz_remote")`)
	got := stub.ScanLoads(f)
	syms := got["@x//:y.bzl"]
	if len(syms) != 1 || syms[0] != "baz_remote" {
		t.Errorf("got[@x//:y.bzl] = %v, want [baz_remote] (From-side, not local alias)", syms)
	}
}

func TestScanLoads_MixedPositionalAndAliased(t *testing.T) {
	f := parseLoadStmts(t, `load("@x//:y.bzl", "foo", baz_local = "baz_remote")`)
	got := stub.ScanLoads(f)
	syms := got["@x//:y.bzl"]
	if len(syms) != 2 {
		t.Fatalf("len(syms) = %d, want 2", len(syms))
	}
	if syms[0] != "foo" || syms[1] != "baz_remote" {
		t.Errorf("syms = %v, want [foo baz_remote]", syms)
	}
}

func TestScanLoads_MultipleLoadsAccumulate(t *testing.T) {
	src := `load("@x//:y.bzl", "foo")
load("@x//:y.bzl", "bar")
load("@x//:z.bzl", "qux")`
	f := parseLoadStmts(t, src)
	got := stub.ScanLoads(f)
	if got["@x//:y.bzl"] == nil {
		t.Fatalf("missing y.bzl")
	}
	if len(got["@x//:y.bzl"]) != 2 {
		t.Errorf("y.bzl syms = %v, want 2 entries (accumulated)", got["@x//:y.bzl"])
	}
	if len(got["@x//:z.bzl"]) != 1 || got["@x//:z.bzl"][0] != "qux" {
		t.Errorf("z.bzl syms = %v, want [qux]", got["@x//:z.bzl"])
	}
}

func TestScanLoads_NoLoads(t *testing.T) {
	f := parseLoadStmts(t, `x = 1`)
	got := stub.ScanLoads(f)
	if got == nil {
		t.Errorf("got = nil, want empty map (non-nil)")
	}
	if len(got) != 0 {
		t.Errorf("got = %v, want empty map", got)
	}
}

func TestScanLoads_NilFile(t *testing.T) {
	got := stub.ScanLoads(nil)
	if got == nil {
		t.Errorf("got = nil, want empty map (non-nil)")
	}
	if len(got) != 0 {
		t.Errorf("got = %v, want empty map", got)
	}
}
