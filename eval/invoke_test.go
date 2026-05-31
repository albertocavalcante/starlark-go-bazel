package eval_test

import (
	"context"
	"sort"
	"testing"

	"go.starlark.net/starlark"

	"github.com/albertocavalcante/starlark-go-bazel/bzl"
	bazelctx "github.com/albertocavalcante/starlark-go-bazel/ctx"
	"github.com/albertocavalcante/starlark-go-bazel/eval"
	"github.com/albertocavalcante/starlark-go-bazel/taint"
	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// captureRule parses src, evaluates it, and returns the named
// RepositoryRuleClass from globals.
func captureRule(t *testing.T, src, name string) *types.RepositoryRuleClass {
	t.Helper()
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	rule, ok := res.Globals[name].(*types.RepositoryRuleClass)
	if !ok {
		t.Fatalf("%s not a *RepositoryRuleClass; got %T", name, res.Globals[name])
	}
	return rule
}

func captureExtension(t *testing.T, src, name string) *types.ModuleExtensionClass {
	t.Helper()
	interp := bzl.New(bzl.Options{})
	res, err := interp.Eval("test.bzl", []byte(src))
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	ext, ok := res.Globals[name].(*types.ModuleExtensionClass)
	if !ok {
		t.Fatalf("%s not a *ModuleExtensionClass; got %T", name, res.Globals[name])
	}
	return ext
}

// Case 1 — literal URL.
func TestInvokeRule_LiteralURL(t *testing.T) {
	src := `
def _impl(ctx):
    ctx.download(url = "https://example.com/foo-1.0.tar.gz", sha256 = "deadbeef")

my_repo = repository_rule(implementation = _impl)
`
	rule := captureRule(t, src, "my_repo")
	inv, err := eval.InvokeRepositoryRule(context.Background(), rule, nil, eval.InvokeOptions{})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if len(inv.URLs) != 1 {
		t.Fatalf("urls = %d (%v), want 1", len(inv.URLs), inv.URLs)
	}
	u := inv.URLs[0]
	if u.URL != "https://example.com/foo-1.0.tar.gz" {
		t.Errorf("URL = %q", u.URL)
	}
	if u.SHA256 != "deadbeef" {
		t.Errorf("SHA256 = %q", u.SHA256)
	}
	if u.RuleName != "my_repo" {
		t.Errorf("RuleName = %q", u.RuleName)
	}
}

// Case 2 — templated URL from ctx.attr.version.
func TestInvokeRule_TemplatedFromAttr(t *testing.T) {
	src := `
def _impl(ctx):
    url = "https://example.com/v{}/foo.tar.gz".format(ctx.attr.version)
    ctx.download(url = url, sha256 = "x")

my_repo = repository_rule(implementation = _impl, attrs = {"version": attr.string()})
`
	rule := captureRule(t, src, "my_repo")
	inv, err := eval.InvokeRepositoryRule(context.Background(), rule, eval.StringAttrs(map[string]string{
		"version": "2.7.0",
	}), eval.InvokeOptions{})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if len(inv.URLs) != 1 || inv.URLs[0].URL != "https://example.com/v2.7.0/foo.tar.gz" {
		t.Errorf("urls = %v", inv.URLs)
	}
}

// Case 3 — platform-branched URL across the 6-fork matrix.
func TestInvokeRule_PlatformFork(t *testing.T) {
	src := `
def _impl(ctx):
    url = "https://example.com/sdk-{}-{}.tar.gz".format(ctx.os.name, ctx.os.arch)
    ctx.download_and_extract(url = url, sha256 = "x")

sdk = repository_rule(implementation = _impl)
`
	rule := captureRule(t, src, "sdk")
	inv, err := eval.InvokeRepositoryRule(context.Background(), rule, nil, eval.InvokeOptions{
		Platforms: taint.DefaultPlatforms(),
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if len(inv.URLs) != 6 {
		t.Fatalf("urls = %d (%v), want 6 (one per platform)", len(inv.URLs), inv.URLs)
	}
	wantPairs := map[string]string{
		"linux/amd64":   "https://example.com/sdk-linux-amd64.tar.gz",
		"linux/arm64":   "https://example.com/sdk-linux-arm64.tar.gz",
		"darwin/amd64":  "https://example.com/sdk-darwin-amd64.tar.gz",
		"darwin/arm64":  "https://example.com/sdk-darwin-arm64.tar.gz",
		"windows/amd64": "https://example.com/sdk-windows-amd64.tar.gz",
		"windows/arm64": "https://example.com/sdk-windows-arm64.tar.gz",
	}
	got := map[string]string{}
	for _, u := range inv.URLs {
		got[u.Platform] = u.URL
	}
	for plat, wantURL := range wantPairs {
		if got[plat] != wantURL {
			t.Errorf("plat %s: got %q, want %q", plat, got[plat], wantURL)
		}
	}
}

// Case 4 — list comprehension over ctx.attr.urls (rules_go's
// sdk.bzl pattern).
func TestInvokeRule_URLListComprehension(t *testing.T) {
	src := `
def _impl(ctx):
    filename = "go1.21.0.linux-amd64.tar.gz"
    urls = [tpl.format(filename) for tpl in ["https://dl.google.com/go/{}", "https://mirror.example.com/go/{}"]]
    ctx.download_and_extract(url = urls, sha256 = "abc")

go_sdk = repository_rule(implementation = _impl)
`
	rule := captureRule(t, src, "go_sdk")
	inv, err := eval.InvokeRepositoryRule(context.Background(), rule, nil, eval.InvokeOptions{})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if len(inv.URLs) != 2 {
		t.Fatalf("urls = %d (%v), want 2", len(inv.URLs), inv.URLs)
	}
	got := []string{inv.URLs[0].URL, inv.URLs[1].URL}
	sort.Strings(got)
	want := []string{
		"https://dl.google.com/go/go1.21.0.linux-amd64.tar.gz",
		"https://mirror.example.com/go/go1.21.0.linux-amd64.tar.gz",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("urls[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// Case 5 — fork errors collected, not propagated. The linux fork
// fails; darwin succeeds; URLs from darwin captured + linux error
// surfaced.
func TestInvokeRule_ForkErrors(t *testing.T) {
	src := `
def _impl(ctx):
    if ctx.os.name == "linux":
        fail("not on linux")
    ctx.download(url = "https://example.com/" + ctx.os.name + ".tar.gz", sha256 = "x")

my_repo = repository_rule(implementation = _impl)
`
	rule := captureRule(t, src, "my_repo")
	inv, err := eval.InvokeRepositoryRule(context.Background(), rule, nil, eval.InvokeOptions{
		Platforms: []taint.Platform{{OS: "linux", Arch: "amd64"}, {OS: "darwin", Arch: "amd64"}},
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if len(inv.URLs) != 1 || inv.URLs[0].URL != "https://example.com/darwin.tar.gz" {
		t.Errorf("urls = %v, want one darwin URL", inv.URLs)
	}
	if len(inv.ForkErrors) != 1 || inv.ForkErrors[0].Platform.OS != "linux" {
		t.Errorf("ForkErrors = %v, want one for linux", inv.ForkErrors)
	}
}

// Dedupe correctness: same URL on every platform collapses to one
// "any" row; URLs that differ across platforms emit per-platform rows
// in sorted order.
func TestInvokeRule_DedupeAnyVsPerPlatform(t *testing.T) {
	src := `
def _impl(ctx):
    # Same URL on every platform — collapses to "any".
    ctx.download(url = "https://example.com/common.tar.gz", sha256 = "x")
    # Platform-specific URL — separate per-platform rows.
    ctx.download(url = "https://example.com/" + ctx.os.name + ".tar.gz", sha256 = "y")

my_repo = repository_rule(implementation = _impl)
`
	rule := captureRule(t, src, "my_repo")
	inv, _ := eval.InvokeRepositoryRule(context.Background(), rule, nil, eval.InvokeOptions{
		Platforms: []taint.Platform{
			{OS: "linux", Arch: "amd64"},
			{OS: "darwin", Arch: "amd64"},
		},
	})

	var anyURL, perPlat []taint.CapturedURL
	for _, u := range inv.URLs {
		if u.Platform == "any" {
			anyURL = append(anyURL, u)
		} else {
			perPlat = append(perPlat, u)
		}
	}
	if len(anyURL) != 1 || anyURL[0].URL != "https://example.com/common.tar.gz" {
		t.Errorf("expected one 'any' row for common.tar.gz; got %v", anyURL)
	}
	if len(perPlat) != 2 {
		t.Errorf("expected 2 per-platform rows; got %v", perPlat)
	}
	// Per-platform rows are emitted in URL-first-seen order (linux ran
	// first because it's index 0 in the Platforms slice). Within each
	// URL, platform keys would be sorted — but with one platform per
	// URL here, that's a no-op. The order assertion below pins
	// fork-iteration determinism.
	got := []string{perPlat[0].Platform, perPlat[1].Platform}
	want := []string{"linux/amd64", "darwin/amd64"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("perPlat[%d].Platform = %q, want %q", i, got[i], want[i])
		}
	}
}

// Module extension dispatch: single tag → single instantiation → URL.
func TestInvokeExtension_SingleTag(t *testing.T) {
	src := `
def _impl(ctx):
    url = "https://example.com/v{}/foo.tar.gz".format(ctx.attr.version)
    ctx.download(url = url, sha256 = "abc")

my_repo = repository_rule(implementation = _impl, attrs = {"version": attr.string()})

def _ext_impl(module_ctx):
    for mod in module_ctx.modules:
        for tag in mod.tags.download:
            my_repo(name = "my_repo_" + tag.version, version = tag.version)

my_ext = module_extension(implementation = _ext_impl)
`
	ext := captureExtension(t, src, "my_ext")
	modules := []bazelctx.ModuleSpec{{
		Name: "root", IsRoot: true,
		Tags: map[string][]bazelctx.TagInstance{
			"download": {
				{Attrs: map[string]starlark.Value{"version": starlark.String("1.21.0")}},
				{Attrs: map[string]starlark.Value{"version": starlark.String("1.22.0")}},
			},
		},
	}}

	res, err := eval.InvokeModuleExtension(context.Background(), ext, modules, eval.InvokeOptions{})
	if err != nil {
		t.Fatalf("invoke ext: %v", err)
	}
	if len(res.Instantiations) != 2 {
		t.Fatalf("instantiations = %d, want 2", len(res.Instantiations))
	}
	if len(res.URLs) != 2 {
		t.Fatalf("urls = %d (%v), want 2", len(res.URLs), res.URLs)
	}
	got := []string{res.URLs[0].URL, res.URLs[1].URL}
	sort.Strings(got)
	want := []string{
		"https://example.com/v1.21.0/foo.tar.gz",
		"https://example.com/v1.22.0/foo.tar.gz",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("urls[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// Module extension dispatch composes with (os, arch) fork: extension
// instantiates one rule; the rule's impl branches on platform; URLs
// fan out.
func TestInvokeExtension_DispatchComposesPlatformFork(t *testing.T) {
	src := `
def _impl(ctx):
    url = "https://example.com/{}-{}-v{}.tar.gz".format(ctx.os.name, ctx.os.arch, ctx.attr.version)
    ctx.download_and_extract(url = url, sha256 = "x")

sdk_repo = repository_rule(implementation = _impl, attrs = {"version": attr.string()})

def _ext_impl(module_ctx):
    for mod in module_ctx.modules:
        for tag in mod.tags.download:
            sdk_repo(name = "sdk_" + tag.version, version = tag.version)

my_ext = module_extension(implementation = _ext_impl)
`
	ext := captureExtension(t, src, "my_ext")
	modules := []bazelctx.ModuleSpec{{
		Name: "root", IsRoot: true,
		Tags: map[string][]bazelctx.TagInstance{
			"download": {{Attrs: map[string]starlark.Value{"version": starlark.String("1.21.0")}}},
		},
	}}

	plats := []taint.Platform{
		{OS: "linux", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
	}
	res, err := eval.InvokeModuleExtension(context.Background(), ext, modules, eval.InvokeOptions{Platforms: plats})
	if err != nil {
		t.Fatalf("invoke ext: %v", err)
	}
	if len(res.URLs) != 2 {
		t.Fatalf("urls = %d (%v), want 2", len(res.URLs), res.URLs)
	}
	want := map[string]bool{
		"https://example.com/linux-amd64-v1.21.0.tar.gz":  false,
		"https://example.com/darwin-arm64-v1.21.0.tar.gz": false,
	}
	for _, u := range res.URLs {
		if _, ok := want[u.URL]; !ok {
			t.Errorf("unexpected url: %s", u.URL)
		}
		want[u.URL] = true
	}
	for url, seen := range want {
		if !seen {
			t.Errorf("missing url: %s", url)
		}
	}
}

// Multi-module dispatch: root + dep both contribute tag instances.
func TestInvokeExtension_MultiModule(t *testing.T) {
	src := `
def _impl(ctx):
    ctx.download(url = "https://example.com/" + ctx.attr.src + ".tar.gz", sha256 = "x")

my_repo = repository_rule(implementation = _impl, attrs = {"src": attr.string()})

def _ext_impl(module_ctx):
    for mod in module_ctx.modules:
        for tag in mod.tags.use:
            my_repo(name = mod.name + "_" + tag.label, src = mod.name)

my_ext = module_extension(implementation = _ext_impl)
`
	ext := captureExtension(t, src, "my_ext")
	modules := []bazelctx.ModuleSpec{
		{Name: "root", IsRoot: true, Tags: map[string][]bazelctx.TagInstance{
			"use": {{Attrs: map[string]starlark.Value{"label": starlark.String("alpha")}}},
		}},
		{Name: "dep", IsRoot: false, Tags: map[string][]bazelctx.TagInstance{
			"use": {{Attrs: map[string]starlark.Value{"label": starlark.String("beta")}}},
		}},
	}
	res, err := eval.InvokeModuleExtension(context.Background(), ext, modules, eval.InvokeOptions{})
	if err != nil {
		t.Fatalf("invoke ext: %v", err)
	}
	if len(res.Instantiations) != 2 {
		t.Fatalf("instantiations = %d, want 2", len(res.Instantiations))
	}
}

// Determinism check: same eval runs twice produces identical URL slices.
func TestInvokeRule_Deterministic(t *testing.T) {
	src := `
def _impl(ctx):
    ctx.download(url = "https://example.com/" + ctx.os.name + ".tar.gz", sha256 = "x")
my_repo = repository_rule(implementation = _impl)
`
	plats := taint.DefaultPlatforms()
	rule := captureRule(t, src, "my_repo")
	var prev []string
	for trial := 0; trial < 5; trial++ {
		inv, _ := eval.InvokeRepositoryRule(context.Background(), rule, nil, eval.InvokeOptions{Platforms: plats})
		curr := make([]string, len(inv.URLs))
		for i, u := range inv.URLs {
			curr[i] = u.Platform + ":" + u.URL
		}
		for i := range prev {
			if i >= len(curr) || prev[i] != curr[i] {
				t.Errorf("trial %d: order changed: prev=%v curr=%v", trial, prev, curr)
				return
			}
		}
		prev = curr
	}
}

// nil rule errors cleanly.
func TestInvokeRule_NilRuleErrors(t *testing.T) {
	_, err := eval.InvokeRepositoryRule(context.Background(), nil, nil, eval.InvokeOptions{})
	if err == nil {
		t.Error("expected error for nil rule")
	}
}
