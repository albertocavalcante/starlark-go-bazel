// Package native provides the native module for Bazel's Starlark dialect.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java
package native

import (
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Module returns the native module for Starlark.
// The native module is available during BUILD file evaluation and provides
// access to native rules and helper functions.
//
// Functions provided:
//   - glob(include, exclude, exclude_directories, allow_empty) - Match files
//   - existing_rule(name) - Get a rule by name
//   - existing_rules() - Get all rules in the package
//   - package_name() - Get the current package path
//   - repository_name() - Get the current repository name (with @)
//   - repo_name() - Get the current repository name (without @)
//   - package_relative_label(input) - Convert string to Label
//   - subpackages(include, exclude, allow_empty) - List subpackages
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkNativeModuleApi.java
func Module() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "native",
		Members: starlark.StringDict{
			"glob":                   starlark.NewBuiltin("native.glob", glob),
			"existing_rule":          starlark.NewBuiltin("native.existing_rule", existingRule),
			"existing_rules":         starlark.NewBuiltin("native.existing_rules", existingRules),
			"package_name":           starlark.NewBuiltin("native.package_name", packageName),
			"repository_name":        starlark.NewBuiltin("native.repository_name", repositoryName),
			"repo_name":              starlark.NewBuiltin("native.repo_name", repoName),
			"package_relative_label": starlark.NewBuiltin("native.package_relative_label", packageRelativeLabel),
			"subpackages":            starlark.NewBuiltin("native.subpackages", subpackages),
		},
	}
}

// ModuleMembers returns just the member functions of the native module.
// These can be added to the global scope for BUILD file evaluation.
func ModuleMembers() starlark.StringDict {
	return Module().Members
}
