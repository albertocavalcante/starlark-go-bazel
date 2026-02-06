package native

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
)

func TestPackageName(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	SetPackageContext(thread, &PackageContext{
		PackagePath: "some/package",
		RepoName:    "",
	})

	result, err := starlark.Call(thread, Module().Members["package_name"], nil, nil)
	if err != nil {
		t.Fatalf("package_name() failed: %v", err)
	}

	if s, ok := starlark.AsString(result); !ok || s != "some/package" {
		t.Errorf("package_name() = %v, want 'some/package'", result)
	}
}

func TestRepositoryName(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	SetPackageContext(thread, &PackageContext{
		PackagePath: "pkg",
		RepoName:    "my_repo",
	})

	result, err := starlark.Call(thread, Module().Members["repository_name"], nil, nil)
	if err != nil {
		t.Fatalf("repository_name() failed: %v", err)
	}

	if s, ok := starlark.AsString(result); !ok || s != "@my_repo" {
		t.Errorf("repository_name() = %v, want '@my_repo'", result)
	}
}

func TestRepoName(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	SetPackageContext(thread, &PackageContext{
		PackagePath: "pkg",
		RepoName:    "my_repo",
	})

	result, err := starlark.Call(thread, Module().Members["repo_name"], nil, nil)
	if err != nil {
		t.Fatalf("repo_name() failed: %v", err)
	}

	if s, ok := starlark.AsString(result); !ok || s != "my_repo" {
		t.Errorf("repo_name() = %v, want 'my_repo'", result)
	}
}

func TestPackageRelativeLabel(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	SetPackageContext(thread, &PackageContext{
		PackagePath: "some/pkg",
		RepoName:    "",
	})

	tests := []struct {
		input string
		want  string
	}{
		{":target", "//some/pkg:target"},
		{"target", "//some/pkg:target"},
		{"//other/pkg:foo", "//other/pkg:foo"},
		{"@ext//pkg:bar", "@ext//pkg:bar"},
	}

	for _, tc := range tests {
		result, err := starlark.Call(thread, Module().Members["package_relative_label"],
			starlark.Tuple{starlark.String(tc.input)}, nil)
		if err != nil {
			t.Errorf("package_relative_label(%q) failed: %v", tc.input, err)
			continue
		}

		label, ok := result.(*types.Label)
		if !ok {
			t.Errorf("package_relative_label(%q) returned %T, want *types.Label", tc.input, result)
			continue
		}

		if got := label.String(); got != tc.want {
			t.Errorf("package_relative_label(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestExistingRule(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	ctx := &PackageContext{
		PackagePath: "pkg",
		Rules:       make(map[string]map[string]starlark.Value),
	}
	ctx.AddRule("my_target", map[string]starlark.Value{
		"kind": starlark.String("cc_library"),
		"srcs": starlark.NewList([]starlark.Value{starlark.String("foo.cc")}),
	})
	SetPackageContext(thread, ctx)

	// Test existing rule
	result, err := starlark.Call(thread, Module().Members["existing_rule"],
		starlark.Tuple{starlark.String("my_target")}, nil)
	if err != nil {
		t.Fatalf("existing_rule('my_target') failed: %v", err)
	}

	view, ok := result.(*ExistingRuleView)
	if !ok {
		t.Fatalf("existing_rule() returned %T, want *ExistingRuleView", result)
	}

	// Check name
	name, found, _ := view.Get(starlark.String("name"))
	if !found || name != starlark.String("my_target") {
		t.Errorf("view['name'] = %v, want 'my_target'", name)
	}

	// Check kind
	kind, found, _ := view.Get(starlark.String("kind"))
	if !found || kind != starlark.String("cc_library") {
		t.Errorf("view['kind'] = %v, want 'cc_library'", kind)
	}

	// Test non-existing rule
	result, err = starlark.Call(thread, Module().Members["existing_rule"],
		starlark.Tuple{starlark.String("nonexistent")}, nil)
	if err != nil {
		t.Fatalf("existing_rule('nonexistent') failed: %v", err)
	}
	if result != starlark.None {
		t.Errorf("existing_rule('nonexistent') = %v, want None", result)
	}
}

func TestExistingRules(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	ctx := &PackageContext{
		PackagePath: "pkg",
		Rules:       make(map[string]map[string]starlark.Value),
	}
	ctx.AddRule("target1", map[string]starlark.Value{"kind": starlark.String("rule1")})
	ctx.AddRule("target2", map[string]starlark.Value{"kind": starlark.String("rule2")})
	SetPackageContext(thread, ctx)

	result, err := starlark.Call(thread, Module().Members["existing_rules"], nil, nil)
	if err != nil {
		t.Fatalf("existing_rules() failed: %v", err)
	}

	view, ok := result.(*ExistingRulesView)
	if !ok {
		t.Fatalf("existing_rules() returned %T, want *ExistingRulesView", result)
	}

	if view.Len() != 2 {
		t.Errorf("len(existing_rules()) = %d, want 2", view.Len())
	}
}

func TestGlob(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "glob_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	for _, name := range []string{"foo.go", "bar.go", "baz.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	thread := &starlark.Thread{Name: "test"}
	SetPackageContext(thread, &PackageContext{
		PackagePath: "pkg",
		PackageDir:  tmpDir,
	})

	// Test glob with include pattern
	result, err := starlark.Call(thread, Module().Members["glob"], nil, []starlark.Tuple{
		{starlark.String("include"), starlark.NewList([]starlark.Value{starlark.String("*.go")})},
	})
	if err != nil {
		t.Fatalf("glob() failed: %v", err)
	}

	list, ok := result.(*starlark.List)
	if !ok {
		t.Fatalf("glob() returned %T, want *starlark.List", result)
	}

	if list.Len() != 2 {
		t.Errorf("glob(['*.go']) returned %d items, want 2", list.Len())
	}

	// Verify sorted order
	first, _ := starlark.AsString(list.Index(0))
	second, _ := starlark.AsString(list.Index(1))
	if first != "bar.go" || second != "foo.go" {
		t.Errorf("glob() results not sorted: got [%s, %s], want [bar.go, foo.go]", first, second)
	}
}

func TestGlobWithExclude(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glob_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, name := range []string{"foo.go", "bar.go", "foo_test.go"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	thread := &starlark.Thread{Name: "test"}
	SetPackageContext(thread, &PackageContext{
		PackagePath: "pkg",
		PackageDir:  tmpDir,
	})

	result, err := starlark.Call(thread, Module().Members["glob"], nil, []starlark.Tuple{
		{starlark.String("include"), starlark.NewList([]starlark.Value{starlark.String("*.go")})},
		{starlark.String("exclude"), starlark.NewList([]starlark.Value{starlark.String("*_test.go")})},
	})
	if err != nil {
		t.Fatalf("glob() failed: %v", err)
	}

	list := result.(*starlark.List)
	if list.Len() != 2 {
		t.Errorf("glob(['*.go'], exclude=['*_test.go']) returned %d items, want 2", list.Len())
	}
}

func TestNoContextError(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}
	// No context set

	funcs := []string{"package_name", "repository_name", "repo_name", "existing_rules"}
	for _, name := range funcs {
		_, err := starlark.Call(thread, Module().Members[name], nil, nil)
		if err == nil {
			t.Errorf("%s() should fail without context", name)
		}
	}
}
