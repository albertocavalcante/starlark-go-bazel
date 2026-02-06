package native

import (
	"fmt"

	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
)

// packageName implements native.package_name().
//
// Returns the name of the package being evaluated, without the repository name.
// For example, in the BUILD file some/package/BUILD, its value will be "some/package".
// The value will always be an empty string for the root package.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#packageName
// Reference: bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkNativeModuleApi.java#packageName
func packageName(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ctx := GetPackageContext(thread)
	if ctx == nil {
		return nil, fmt.Errorf("native.package_name() can only be called during BUILD file evaluation")
	}

	if err := starlark.UnpackArgs("package_name", args, kwargs); err != nil {
		return nil, err
	}

	return starlark.String(ctx.PackagePath), nil
}

// repositoryName implements native.repository_name().
//
// Returns the canonical name of the repository containing the package currently
// being evaluated, with a single at-sign (@) prefixed.
// For example, "@local" for a local_repository, or "@" for the main repository.
//
// Deprecated: Use repo_name() instead, which doesn't have the spurious leading @.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#repositoryName
// Reference: bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkNativeModuleApi.java#repositoryName
func repositoryName(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ctx := GetPackageContext(thread)
	if ctx == nil {
		return nil, fmt.Errorf("native.repository_name() can only be called during BUILD file evaluation")
	}

	if err := starlark.UnpackArgs("repository_name", args, kwargs); err != nil {
		return nil, err
	}

	// For legacy reasons, this is prefixed with a single '@'
	return starlark.String("@" + ctx.RepoName), nil
}

// repoName implements native.repo_name().
//
// Returns the canonical name of the repository containing the package currently
// being evaluated, with no leading at-signs.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#repoName
// Reference: bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkNativeModuleApi.java#repoName
func repoName(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ctx := GetPackageContext(thread)
	if ctx == nil {
		return nil, fmt.Errorf("native.repo_name() can only be called during BUILD file evaluation")
	}

	if err := starlark.UnpackArgs("repo_name", args, kwargs); err != nil {
		return nil, err
	}

	return starlark.String(ctx.RepoName), nil
}

// packageRelativeLabel implements native.package_relative_label(input).
//
// Converts the input string into a Label object, in the context of the package
// currently being initialized. If the input is already a Label, it is returned
// unchanged.
//
// This function may only be called while evaluating a BUILD file.
//
// The result is the same Label value as would be produced by passing the given
// string to a label-valued attribute of a target declared in the BUILD file.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#packageRelativeLabel
// Reference: bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkNativeModuleApi.java#packageRelativeLabel
func packageRelativeLabel(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ctx := GetPackageContext(thread)
	if ctx == nil {
		return nil, fmt.Errorf("native.package_relative_label() can only be called during BUILD file evaluation")
	}

	var input starlark.Value
	if err := starlark.UnpackArgs("package_relative_label", args, kwargs, "input", &input); err != nil {
		return nil, err
	}

	// If input is already a Label, return it unchanged
	if label, ok := input.(*types.Label); ok {
		return label, nil
	}

	// Otherwise, parse as string
	inputStr, ok := starlark.AsString(input)
	if !ok {
		return nil, fmt.Errorf("invalid label in native.package_relative_label: expected string or Label, got %s", input.Type())
	}

	label, err := types.ParseLabelRelative(inputStr, ctx.RepoName, ctx.PackagePath)
	if err != nil {
		return nil, fmt.Errorf("invalid label in native.package_relative_label: %w", err)
	}

	return label, nil
}
