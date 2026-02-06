package native

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.starlark.net/starlark"
)

// glob implements native.glob(include, exclude, exclude_directories, allow_empty).
//
// Parameters:
//   - include: list of glob patterns to include (default: [])
//   - exclude: list of glob patterns to exclude (default: [])
//   - exclude_directories: if non-zero (default 1), exclude directories from results
//   - allow_empty: if False, fail when glob matches nothing (default: unbound -> True)
//
// Returns a new, mutable, sorted list of every file in the current package
// that matches at least one pattern in include and does not match any pattern
// in exclude.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#glob
// Reference: bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkNativeModuleApi.java#glob
func glob(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ctx := GetPackageContext(thread)
	if ctx == nil {
		return nil, fmt.Errorf("native.glob() can only be called during BUILD file evaluation")
	}

	var include, exclude *starlark.List
	var excludeDirs starlark.Int = starlark.MakeInt(1) // default: exclude directories
	var allowEmpty starlark.Value = starlark.None      // unbound in Bazel

	if err := starlark.UnpackArgs("glob", args, kwargs,
		"include?", &include,
		"exclude?", &exclude,
		"exclude_directories?", &excludeDirs,
		"allow_empty?", &allowEmpty,
	); err != nil {
		return nil, err
	}

	// Default empty lists
	if include == nil {
		include = starlark.NewList(nil)
	}
	if exclude == nil {
		exclude = starlark.NewList(nil)
	}

	// Convert patterns to string slices
	includePatterns, err := listToStrings(include, "glob include")
	if err != nil {
		return nil, err
	}
	excludePatterns, err := listToStrings(exclude, "glob exclude")
	if err != nil {
		return nil, err
	}

	// Determine if we should include directories
	// exclude_directories=1 (default) means exclude directories (FILES only)
	// exclude_directories=0 means include directories (FILES_AND_DIRS)
	excludeDirsInt, ok := excludeDirs.Int64()
	if !ok {
		return nil, fmt.Errorf("exclude_directories must be an integer")
	}
	includeDirs := excludeDirsInt == 0

	// Determine allow_empty behavior
	// In Bazel, the default is controlled by a semantic flag, but we default to True
	shouldAllowEmpty := true
	if allowEmpty != starlark.None {
		if b, ok := allowEmpty.(starlark.Bool); ok {
			shouldAllowEmpty = bool(b)
		} else {
			return nil, fmt.Errorf("expected boolean for argument `allow_empty`, got `%s`", allowEmpty.Type())
		}
	}

	// Execute the glob
	matches, err := executeGlob(ctx, includePatterns, excludePatterns, includeDirs)
	if err != nil {
		return nil, err
	}

	// Check allow_empty
	if !shouldAllowEmpty && len(matches) == 0 {
		return nil, fmt.Errorf("glob pattern(s) %v matched no files", includePatterns)
	}

	// Sort results (Bazel sorts alphabetically)
	sort.Strings(matches)

	// Convert to Starlark list
	values := make([]starlark.Value, len(matches))
	for i, m := range matches {
		// If the match starts with @, prefix with : to disambiguate from external repo
		// Reference: StarlarkNativeModule.java line 123-127
		if strings.HasPrefix(m, "@") {
			m = ":" + m
		}
		values[i] = starlark.String(m)
	}

	return starlark.NewList(values), nil
}

// subpackages implements native.subpackages(include, exclude, allow_empty).
//
// Parameters:
//   - include: list of glob patterns to include
//   - exclude: list of glob patterns to exclude (default: [])
//   - allow_empty: if False, fail when nothing matches (default: False)
//
// Returns a new mutable list of every direct subpackage of the current package.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#subpackages
func subpackages(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ctx := GetPackageContext(thread)
	if ctx == nil {
		return nil, fmt.Errorf("native.subpackages() can only be called during BUILD file evaluation")
	}

	var include, exclude *starlark.List
	var allowEmpty bool = false // default: False (unlike glob)

	if err := starlark.UnpackArgs("subpackages", args, kwargs,
		"include", &include,
		"exclude?", &exclude,
		"allow_empty?", &allowEmpty,
	); err != nil {
		return nil, err
	}

	if exclude == nil {
		exclude = starlark.NewList(nil)
	}

	includePatterns, err := listToStrings(include, "subpackages include")
	if err != nil {
		return nil, err
	}
	excludePatterns, err := listToStrings(exclude, "subpackages exclude")
	if err != nil {
		return nil, err
	}

	// Find subpackages
	matches, err := executeSubpackagesGlob(ctx, includePatterns, excludePatterns)
	if err != nil {
		return nil, err
	}

	if !allowEmpty && len(matches) == 0 {
		return nil, fmt.Errorf("subpackages pattern(s) %v matched no subpackages", includePatterns)
	}

	sort.Strings(matches)

	values := make([]starlark.Value, len(matches))
	for i, m := range matches {
		values[i] = starlark.String(m)
	}

	return starlark.NewList(values), nil
}

// executeGlob executes a glob operation against the filesystem.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/GlobCache.java
func executeGlob(ctx *PackageContext, include, exclude []string, includeDirs bool) ([]string, error) {
	if ctx.PackageDir == "" {
		return nil, fmt.Errorf("PackageDir not set in context")
	}

	matches := make(map[string]struct{})

	// Process include patterns
	for _, pattern := range include {
		if err := validateGlobPattern(pattern); err != nil {
			return nil, err
		}

		found, err := matchPattern(ctx, pattern, includeDirs)
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			matches[f] = struct{}{}
		}
	}

	// Process exclude patterns
	for _, pattern := range exclude {
		if err := validateGlobPattern(pattern); err != nil {
			return nil, err
		}

		found, err := matchPattern(ctx, pattern, includeDirs)
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			delete(matches, f)
		}
	}

	result := make([]string, 0, len(matches))
	for m := range matches {
		result = append(result, m)
	}
	return result, nil
}

// executeSubpackagesGlob finds subpackages matching the given patterns.
func executeSubpackagesGlob(ctx *PackageContext, include, exclude []string) ([]string, error) {
	if ctx.PackageDir == "" {
		return nil, fmt.Errorf("PackageDir not set in context")
	}

	locator := ctx.BuildFileLocator
	if locator == nil {
		locator = DefaultBuildFileLocator{}
	}

	matches := make(map[string]struct{})

	// Process include patterns
	for _, pattern := range include {
		if err := validateGlobPattern(pattern); err != nil {
			return nil, err
		}

		found, err := matchSubpackagePattern(ctx, pattern, locator)
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			matches[f] = struct{}{}
		}
	}

	// Process exclude patterns
	for _, pattern := range exclude {
		if err := validateGlobPattern(pattern); err != nil {
			return nil, err
		}

		found, err := matchSubpackagePattern(ctx, pattern, locator)
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			delete(matches, f)
		}
	}

	result := make([]string, 0, len(matches))
	for m := range matches {
		result = append(result, m)
	}
	return result, nil
}

// validateGlobPattern validates a glob pattern.
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/GlobCache.java#safeGlobUnsorted
func validateGlobPattern(pattern string) error {
	// Forbidden: '?' wildcard
	if strings.Contains(pattern, "?") {
		return fmt.Errorf("glob pattern '%s' contains forbidden '?' wildcard", pattern)
	}

	// Forbidden: uplevel references
	if strings.Contains(pattern, "..") {
		return fmt.Errorf("glob pattern '%s' contains forbidden '..' reference", pattern)
	}

	// Forbidden: absolute paths
	if strings.HasPrefix(pattern, "/") {
		return fmt.Errorf("glob pattern '%s' cannot be absolute", pattern)
	}

	return nil
}

// matchPattern matches a glob pattern against the filesystem.
func matchPattern(ctx *PackageContext, pattern string, includeDirs bool) ([]string, error) {
	locator := ctx.BuildFileLocator
	if locator == nil {
		locator = DefaultBuildFileLocator{}
	}

	// Use Go's filepath.Glob for simple patterns
	// For recursive patterns (**), we need custom handling
	if strings.Contains(pattern, "**") {
		return matchRecursivePattern(ctx, pattern, includeDirs, locator)
	}

	fullPattern := filepath.Join(ctx.PackageDir, pattern)
	absMatches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
	}

	var result []string
	for _, abs := range absMatches {
		// Get relative path
		rel, err := filepath.Rel(ctx.PackageDir, abs)
		if err != nil {
			continue
		}

		// Skip if it's a directory and we're not including directories
		info, err := os.Stat(abs)
		if err != nil {
			continue
		}
		if info.IsDir() {
			if !includeDirs {
				continue
			}
			// Skip subpackages (directories with BUILD files)
			if locator.HasBuildFile(abs) {
				continue
			}
		}

		// Skip empty relative paths (current directory)
		if rel == "." || rel == "" {
			continue
		}

		result = append(result, rel)
	}

	return result, nil
}

// matchRecursivePattern handles patterns with ** (recursive matching).
func matchRecursivePattern(ctx *PackageContext, pattern string, includeDirs bool, locator BuildFileLocator) ([]string, error) {
	var result []string

	// Split pattern at **
	parts := strings.SplitN(pattern, "**", 2)
	prefix := parts[0]
	suffix := ""
	if len(parts) > 1 {
		suffix = parts[1]
	}

	// Remove trailing slash from prefix
	prefix = strings.TrimSuffix(prefix, "/")

	// Remove leading slash from suffix
	suffix = strings.TrimPrefix(suffix, "/")

	startDir := ctx.PackageDir
	if prefix != "" {
		startDir = filepath.Join(ctx.PackageDir, prefix)
	}

	err := filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Get relative path from package directory
		rel, err := filepath.Rel(ctx.PackageDir, path)
		if err != nil {
			return nil
		}

		// Skip the package directory itself
		if rel == "." {
			return nil
		}

		// Skip subpackages and don't descend into them
		if info.IsDir() && locator.HasBuildFile(path) && path != startDir {
			return filepath.SkipDir
		}

		// If it's a directory and we're not including directories, skip
		if info.IsDir() && !includeDirs {
			return nil
		}

		// Match suffix pattern if present
		if suffix != "" {
			baseName := filepath.Base(path)
			matched, err := filepath.Match(suffix, baseName)
			if err != nil || !matched {
				return nil
			}
		}

		result = append(result, rel)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// matchSubpackagePattern finds subpackages matching a glob pattern.
func matchSubpackagePattern(ctx *PackageContext, pattern string, locator BuildFileLocator) ([]string, error) {
	var result []string

	if strings.Contains(pattern, "**") {
		// Recursive matching for subpackages
		err := filepath.Walk(ctx.PackageDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if !info.IsDir() {
				return nil
			}

			// Skip the package directory itself
			if path == ctx.PackageDir {
				return nil
			}

			// Get relative path
			rel, err := filepath.Rel(ctx.PackageDir, path)
			if err != nil {
				return nil
			}

			// Check if this is a subpackage
			if locator.HasBuildFile(path) {
				result = append(result, rel)
				// Don't recurse into subpackages for their subpackages
				// (subpackages only returns direct subpackages of matching dirs)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Non-recursive pattern
		fullPattern := filepath.Join(ctx.PackageDir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, err
		}

		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil || !info.IsDir() {
				continue
			}

			if locator.HasBuildFile(m) {
				rel, err := filepath.Rel(ctx.PackageDir, m)
				if err != nil {
					continue
				}
				result = append(result, rel)
			}
		}
	}

	return result, nil
}

// listToStrings converts a Starlark list to a Go string slice.
func listToStrings(list *starlark.List, context string) ([]string, error) {
	if list == nil {
		return nil, nil
	}

	result := make([]string, list.Len())
	for i := range list.Len() {
		s, ok := starlark.AsString(list.Index(i))
		if !ok {
			return nil, fmt.Errorf("%s element %d is not a string", context, i)
		}
		result[i] = s
	}
	return result, nil
}
