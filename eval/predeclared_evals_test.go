package eval_test

import (
	"fmt"
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/bzl"
	"github.com/albertocavalcante/starlark-go-bazel/eval"
)

// Plan 08 T1: aspect() resolves as a predeclared global in .bzl.
//
// Before the fix, evaluation fails with "undefined: aspect" because
// makeBzlPredeclared didn't register builtins.Aspect.
func TestEval_AspectResolvesInBzl(t *testing.T) {
	src := []byte(`
def _impl(target, ctx):
    pass

_a = aspect(implementation = _impl)
`)
	interp := bzl.New(bzl.Options{})
	if _, err := interp.Eval("defs.bzl", src); err != nil {
		t.Fatalf("eval: %v", err)
	}
}

// Plan 08 T3: a single .bzl that exercises every Implemented or
// Stubbed entry evaluates cleanly. Catches "wired but rejects valid
// input" — the exact failure aspect() had before plan 07's holder
// migration.
func TestPredeclared_UniverseEvalsAtVLatest(t *testing.T) {
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("universe.bzl", []byte(eval.UniverseSrc))
	if err != nil {
		t.Fatalf("universe eval: %v", err)
	}
	wantBindings := []string{
		"_provider", "_rule", "_aspect", "_repo_rule", "_mod_ext",
		"_tag", "_l", "_d", "_s", "_attr_mod", "_native_mod",
		"_t", "_f", "_n",
	}
	for _, name := range wantBindings {
		if _, ok := res.Globals[name]; !ok {
			t.Errorf("universe.bzl missing binding %q after eval", name)
		}
	}
}

// Plan 08: every Status=missing entry in the manifest does NOT
// resolve at runtime. Catches the inverse drift — a missing-marked
// builtin getting silently wired without flipping the manifest to
// Status=implemented.
func TestPredeclared_MissingListIsFailingAtVLatest(t *testing.T) {
	for _, entry := range eval.Manifest {
		if entry.Status != eval.StatusMissing {
			continue
		}
		// Manifest currently has no missing constants; skip the
		// kind defensively (assigning a constant name to itself
		// is a no-op rather than a resolution failure).
		if entry.Kind == "constant" {
			continue
		}
		t.Run(entry.Name, func(t *testing.T) {
			src := []byte(fmt.Sprintf("_x = %s\n", entry.Name))
			interp := bzl.New(bzl.Options{})
			if _, err := interp.Eval("missing.bzl", src); err == nil {
				t.Errorf("manifest says %q is missing but it resolved (flip Status to %s?)",
					entry.Name, eval.StatusImplemented)
			}
		})
	}
}
