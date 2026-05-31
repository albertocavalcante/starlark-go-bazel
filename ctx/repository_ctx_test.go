package ctx_test

import (
	"strings"
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/ctx"
	"github.com/albertocavalcante/starlark-go-bazel/taint"
	"github.com/albertocavalcante/starlark-go-bazel/version"
	"go.starlark.net/starlark"
)

// callImplWithSrc evaluates src (defining `impl` taking the ctx arg)
// and invokes impl. impl must return a value if the test needs to
// inspect it (go.starlark.net freezes globals after ExecFile, so the
// `result = []` + `result.append(...)` pattern doesn't work).
func callImplWithSrc(t *testing.T, src string, rctx starlark.Value) starlark.Value {
	t.Helper()
	thread := &starlark.Thread{Name: "test"}
	globals, err := starlark.ExecFile(thread, "test.bzl", src, nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	impl, ok := globals["impl"].(starlark.Callable)
	if !ok {
		t.Fatal("test src must define `impl`")
	}
	out, err := starlark.Call(thread, impl, starlark.Tuple{rctx}, nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	return out
}

// name / original_name / workspace_root surface concrete strings.
func TestRepositoryCtx_NameAndPath(t *testing.T) {
	rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{
		Name:          "my_repo",
		WorkspaceRoot: "/custom/root",
	})
	got, _ := rctx.Attr("name")
	if s, _ := starlark.AsString(got); s != "my_repo" {
		t.Errorf("name = %q, want my_repo", s)
	}
	got, _ = rctx.Attr("original_name")
	if s, _ := starlark.AsString(got); s != "my_repo" {
		t.Errorf("original_name = %q, want my_repo (defaults to name)", s)
	}
	got, _ = rctx.Attr("workspace_root")
	if s, _ := starlark.AsString(got); s != "/custom/root" {
		t.Errorf("workspace_root = %q", s)
	}
}

// ctx.attr.<name> resolves to supplied values; unknown returns empty.
func TestRepositoryCtx_AttrLookup(t *testing.T) {
	rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{
		Attrs: map[string]starlark.Value{
			"version": starlark.String("1.21.0"),
		},
	})
	attrProxy, err := rctx.Attr("attr")
	if err != nil || attrProxy == nil {
		t.Fatalf("ctx.attr = %v err=%v", attrProxy, err)
	}
	hasAttrs := attrProxy.(starlark.HasAttrs)
	got, _ := hasAttrs.Attr("version")
	if s, _ := starlark.AsString(got); s != "1.21.0" {
		t.Errorf("attr.version = %q", s)
	}
	got, _ = hasAttrs.Attr("missing")
	if s, _ := starlark.AsString(got); s != "" {
		t.Errorf("attr.missing = %q, want empty", s)
	}
}

// ctx.os.{name,arch,environ} surface from options.
func TestRepositoryCtx_OsSurface(t *testing.T) {
	rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{
		OSName: "linux",
		OSArch: "amd64",
		OSEnv:  map[string]string{"FOO": "bar"},
	})
	os, _ := rctx.Attr("os")
	osAttrs := os.(starlark.HasAttrs)
	n, _ := osAttrs.Attr("name")
	if s, _ := starlark.AsString(n); s != "linux" {
		t.Errorf("os.name = %q", s)
	}
	a, _ := osAttrs.Attr("arch")
	if s, _ := starlark.AsString(a); s != "amd64" {
		t.Errorf("os.arch = %q", s)
	}
	env, _ := osAttrs.Attr("environ")
	if env.(*starlark.Dict).Len() != 1 {
		t.Errorf("os.environ len = %d, want 1", env.(*starlark.Dict).Len())
	}
}

// ctx.download with a literal URL captures into Sinks.
func TestRepositoryCtx_DownloadCaptures(t *testing.T) {
	sinks := &taint.Sinks{}
	rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{
		OSName: "linux", OSArch: "amd64",
		Sinks: sinks,
	})
	src := `
def impl(ctx):
    ctx.download(url = "https://example.com/foo.tar.gz", sha256 = "deadbeef")
`
	callImplWithSrc(t, src, rctx)
	if len(sinks.URLs) != 1 {
		t.Fatalf("URLs = %d, want 1", len(sinks.URLs))
	}
	u := sinks.URLs[0]
	if u.URL != "https://example.com/foo.tar.gz" {
		t.Errorf("URL = %q", u.URL)
	}
	if u.SHA256 != "deadbeef" {
		t.Errorf("SHA256 = %q", u.SHA256)
	}
	if u.Platform != "linux/amd64" {
		t.Errorf("Platform = %q", u.Platform)
	}
}

// ctx.download_and_extract captures with strip_prefix.
func TestRepositoryCtx_DownloadAndExtractCaptures(t *testing.T) {
	sinks := &taint.Sinks{}
	rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{Sinks: sinks})
	src := `
def impl(ctx):
    ctx.download_and_extract(
        url = ["https://a/x.tar.gz", "https://b/x.tar.gz"],
        sha256 = "abc",
        strip_prefix = "x-1.0",
    )
`
	callImplWithSrc(t, src, rctx)
	if len(sinks.URLs) != 2 {
		t.Fatalf("URLs = %d, want 2", len(sinks.URLs))
	}
	if sinks.URLs[0].StripPrefix != "x-1.0" || sinks.URLs[1].StripPrefix != "x-1.0" {
		t.Errorf("StripPrefix not set on both URLs")
	}
}

// execute / read / which set the per-fork tainted flag; subsequent
// download URLs inherit Tainted=true.
func TestRepositoryCtx_TaintFlagAfterExecute(t *testing.T) {
	sinks := &taint.Sinks{}
	rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{Sinks: sinks})
	src := `
def impl(ctx):
    ctx.execute(["uname"])
    ctx.download(url = "https://example.com/x.tar.gz", sha256 = "x")
`
	callImplWithSrc(t, src, rctx)
	if !rctx.Tainted() {
		t.Error("Tainted() = false, want true after execute")
	}
	if len(sinks.URLs) != 1 || !sinks.URLs[0].Tainted {
		t.Errorf("URL not tainted: %+v", sinks.URLs)
	}
}

// Download before any opaque op stays clean.
func TestRepositoryCtx_DownloadBeforeExecuteIsClean(t *testing.T) {
	sinks := &taint.Sinks{}
	rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{Sinks: sinks})
	src := `
def impl(ctx):
    ctx.download(url = "https://example.com/x.tar.gz", sha256 = "x")
    ctx.execute(["uname"])
`
	callImplWithSrc(t, src, rctx)
	if len(sinks.URLs) != 1 || sinks.URLs[0].Tainted {
		t.Errorf("URL should be clean (download before execute); got %+v", sinks.URLs)
	}
}

// ctx.getenv: known env var returns value; unknown taints.
func TestRepositoryCtx_GetenvKnownAndUnknown(t *testing.T) {
	rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{
		OSEnv:   map[string]string{"GOPATH": "/home/u/go"},
		Version: version.V9,
	})
	src := `
def impl(ctx):
    return (ctx.getenv("GOPATH"), ctx.getenv("DOES_NOT_EXIST"))
`
	out := callImplWithSrc(t, src, rctx)
	tup, ok := out.(starlark.Tuple)
	if !ok || tup.Len() != 2 {
		t.Fatalf("impl returned %v, want tuple of 2", out)
	}
	if s, _ := starlark.AsString(tup.Index(0)); s != "/home/u/go" {
		t.Errorf("known getenv = %q", s)
	}
	if !rctx.Tainted() {
		t.Errorf("unknown getenv should have tainted ctx")
	}
}

// Version-gated attrs hide when Version < required.
func TestRepositoryCtx_VersionGating(t *testing.T) {
	cases := []struct {
		ver  version.Version
		attr string
		want bool // should the attr be present?
	}{
		{version.V7, "getenv", false},        // Bazel 8+
		{version.V7, "watch", false},         // Bazel 8+
		{version.V7, "repo_metadata", false}, // Bazel 9+
		{version.V8, "getenv", true},
		{version.V8, "watch", true},
		{version.V8, "repo_metadata", false},
		{version.V9, "repo_metadata", true},
		{version.VLatest, "getenv", true},
		{version.VLatest, "repo_metadata", true},
	}
	for _, c := range cases {
		rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{Version: c.ver})
		v, _ := rctx.Attr(c.attr)
		got := v != nil
		if got != c.want {
			t.Errorf("Version=%s Attr(%q) present=%v, want %v", c.ver, c.attr, got, c.want)
		}
	}
}

// AttrNames includes version-gated attrs at the right Version.
func TestRepositoryCtx_AttrNamesIncludesVersionGated(t *testing.T) {
	rctx := ctx.NewRepositoryCtx(ctx.RepositoryCtxOptions{Version: version.V9})
	names := strings.Join(rctx.AttrNames(), ",")
	for _, need := range []string{"getenv", "watch", "repo_metadata"} {
		if !strings.Contains(names, need) {
			t.Errorf("AttrNames missing %q at V9: %s", need, names)
		}
	}
}
