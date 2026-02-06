// Package native provides the native module for Bazel's Starlark dialect.
//
// The native module is available during BUILD file evaluation and provides
// functions like glob(), package_name(), existing_rule(), etc.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java
package native

import (
	"os"
	"path/filepath"

	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
)

// packageContextKey is the key used to store PackageContext in thread locals.
const packageContextKey = "github.com/albertocavalcante/starlark-go-bazel/native.PackageContext"

// PackageContext provides the context needed by native functions.
// It must be stored in the thread via SetPackageContext before calling
// any native functions.
type PackageContext struct {
	// PackagePath is the package path relative to the workspace root.
	// For example, "some/package" for //some/package:target.
	// Empty string for the root package.
	PackagePath string

	// RepoName is the canonical repository name.
	// Empty string for the main repository.
	RepoName string

	// PackageDir is the absolute path to the package directory on disk.
	// Used for glob operations.
	PackageDir string

	// Rules contains the rules defined so far in this package.
	// Maps rule name to rule attributes.
	Rules map[string]map[string]starlark.Value

	// BuildFileLocator is used to determine if a directory is a subpackage.
	// If nil, subpackage detection is disabled.
	BuildFileLocator BuildFileLocator
}

// BuildFileLocator determines if a directory contains a BUILD file.
type BuildFileLocator interface {
	// HasBuildFile returns true if the given directory contains a BUILD file.
	HasBuildFile(dir string) bool
}

// DefaultBuildFileLocator checks for BUILD or BUILD.bazel files.
type DefaultBuildFileLocator struct{}

// HasBuildFile checks for BUILD or BUILD.bazel in the directory.
func (d DefaultBuildFileLocator) HasBuildFile(dir string) bool {
	for _, name := range []string{"BUILD.bazel", "BUILD"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// SetPackageContext stores a PackageContext in the thread for use by native functions.
func SetPackageContext(thread *starlark.Thread, ctx *PackageContext) {
	thread.SetLocal(packageContextKey, ctx)
}

// GetPackageContext retrieves the PackageContext from the thread.
// Returns nil if no context is set.
func GetPackageContext(thread *starlark.Thread) *PackageContext {
	if v := thread.Local(packageContextKey); v != nil {
		return v.(*PackageContext)
	}
	return nil
}

// AddRule adds a rule to the package context.
// Called when a rule is instantiated during BUILD file evaluation.
func (ctx *PackageContext) AddRule(name string, attrs map[string]starlark.Value) {
	if ctx.Rules == nil {
		ctx.Rules = make(map[string]map[string]starlark.Value)
	}
	ctx.Rules[name] = attrs
}

// GetRule returns the attributes of a rule by name, or nil if not found.
func (ctx *PackageContext) GetRule(name string) map[string]starlark.Value {
	if ctx.Rules == nil {
		return nil
	}
	return ctx.Rules[name]
}

// ResolveLabel resolves a label string relative to the current package.
// Returns a *types.Label.
func (ctx *PackageContext) ResolveLabel(input string) (*types.Label, error) {
	return types.ParseLabelRelative(input, ctx.RepoName, ctx.PackagePath)
}
