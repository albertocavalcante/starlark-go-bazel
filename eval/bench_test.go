package eval_test

import (
	"context"
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/bzl"
	"github.com/albertocavalcante/starlark-go-bazel/eval"
	"github.com/albertocavalcante/starlark-go-bazel/taint"
	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// BenchmarkInvokeRule_PlatformFork exercises the hot path: synthetic ctx
// per platform fork, impl invocation, taint sink population, dedupe.
func BenchmarkInvokeRule_PlatformFork(b *testing.B) {
	src := `
def _impl(ctx):
    url = "https://example.com/sdk-{}-{}.tar.gz".format(ctx.os.name, ctx.os.arch)
    ctx.download_and_extract(url = url, sha256 = "x")

sdk = repository_rule(implementation = _impl)
`
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		b.Fatal(err)
	}
	rule := res.Globals["sdk"].(*types.RepositoryRuleClass)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eval.InvokeRepositoryRule(ctx, rule, nil, eval.InvokeOptions{
			Platforms: taint.DefaultPlatforms(),
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkInvokeRule_SinglePlatform is the same impl without the fork
// matrix — isolates per-fork overhead from per-eval overhead.
func BenchmarkInvokeRule_SinglePlatform(b *testing.B) {
	src := `
def _impl(ctx):
    ctx.download(url = "https://example.com/foo.tar.gz", sha256 = "x")

r = repository_rule(implementation = _impl)
`
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		b.Fatal(err)
	}
	rule := res.Globals["r"].(*types.RepositoryRuleClass)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eval.InvokeRepositoryRule(ctx, rule, nil, eval.InvokeOptions{})
		if err != nil {
			b.Fatal(err)
		}
	}
}
