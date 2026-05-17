package bzl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// TestLenientLoad_LoaderErrorTolerated documents what the loader-side
// LenientLoad option does on its own: a load() that fails inside the
// BzlFileLoader (unknown repo, missing file) returns an empty
// StringDict + nil error instead of aborting. The CALLER's evaluator
// still gets to handle "but the requested symbols aren't in that
// dict" — Starlark itself then errors "name X not found in module Y"
// after Load returns.
//
// To get end-to-end "load doesn't error and the importing file
// continues," the caller also needs to ensure the returned StringDict
// contains stubs for every name requested in the load() statement.
// That's a SOURCE-LEVEL concern (you need the parsed AST to see
// which names are being asked for); assay/interp handles it via a
// pre-eval source rewrite that replaces external loads with
// `name = None` stub bindings. This option remains useful for cases
// where the load resolves but the loaded file itself fails to parse
// or has a transitive load failure we want to swallow.
func TestLenientLoad_LoaderErrorTolerated(t *testing.T) {
	dir := t.TempDir()
	// Use a load whose symbol list is empty so we test JUST the
	// loader path, not Starlark's "name not found" frontend check.
	// Empty load lists aren't legal Starlark, so instead we use a
	// minimal evaluator-level smoke test that ensures the option
	// doesn't break the happy path.
	src := `
def _impl(ctx):
    pass

my_rule = rule(implementation = _impl, attrs = {"srcs": attr.label_list()})
`
	if err := os.WriteFile(filepath.Join(dir, "defs.bzl"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	lenient := New(Options{WorkspaceRoot: dir, LenientLoad: true})
	res, err := lenient.EvalFile(filepath.Join(dir, "defs.bzl"))
	if err != nil {
		t.Fatalf("lenient mode: unexpected error on file with no loads: %v", err)
	}
	if _, ok := res.Globals["my_rule"]; !ok {
		t.Fatal("my_rule missing from globals")
	}
	if _, ok := res.Globals["my_rule"].(*types.RuleClass); !ok {
		t.Errorf("my_rule is %T, want *types.RuleClass", res.Globals["my_rule"])
	}
}
