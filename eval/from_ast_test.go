package eval_test

import (
	"strings"
	"testing"

	"go.starlark.net/syntax"

	"github.com/albertocavalcante/starlark-go-bazel/eval"
)

// EvalBzlFromAST accepts a pre-parsed *syntax.File so callers that
// already parsed the source for unrelated reasons (load scanning,
// AST walks) don't pay for a second parse inside ExecFile. This test
// pins three contracts:
//  1. The pre-parsed path produces the same globals as the
//     source-bytes path.
//  2. A simple repository_rule definition lands in globals.
//  3. Passing nil for the parsed file errors cleanly.
func TestEvalBzlFromAST_MatchesEvalBzl(t *testing.T) {
	src := []byte(`
def _impl(ctx):
    ctx.download(url = "https://example.com/x.tar.gz", sha256 = "abc")

my_repo = repository_rule(implementation = _impl)
`)
	const path = "test.bzl"

	ev := eval.New(eval.Options{})

	// Baseline: existing source-bytes path.
	baseline, err := ev.EvalBzl(path, src)
	if err != nil {
		t.Fatalf("EvalBzl: %v", err)
	}
	if _, ok := baseline.Globals["my_repo"]; !ok {
		t.Fatal("baseline missing my_repo")
	}

	// Pre-parse + new path.
	parsed, err := syntax.LegacyFileOptions().Parse(path, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ev.EvalBzlFromAST(path, parsed)
	if err != nil {
		t.Fatalf("EvalBzlFromAST: %v", err)
	}
	if _, ok := got.Globals["my_repo"]; !ok {
		t.Fatal("FromAST missing my_repo")
	}

	// Both paths should produce the same global names.
	for name := range baseline.Globals {
		if _, ok := got.Globals[name]; !ok {
			t.Errorf("FromAST missing global %q present in baseline", name)
		}
	}
}

func TestEvalBzlFromAST_NilFile(t *testing.T) {
	ev := eval.New(eval.Options{})
	_, err := ev.EvalBzlFromAST("x.bzl", nil)
	if err == nil {
		t.Fatal("expected error on nil file")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil: %v", err)
	}
}
