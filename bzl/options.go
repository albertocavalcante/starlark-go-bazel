// Package bzl provides the main user-facing API for evaluating Bazel Starlark files.
package bzl

import "github.com/albertocavalcante/starlark-go-bazel/loader"

// Options configures the interpreter.
type Options struct {
	// WorkspaceRoot is the root of the Bazel workspace.
	WorkspaceRoot string

	// FileSystem for loading files (default: OS filesystem).
	FileSystem loader.FileSystem

	// ExternalRepos maps repository names to paths.
	ExternalRepos map[string]string

	// PrintHandler handles print() output.
	PrintHandler func(msg string)

	// LenientLoad makes load() calls for unresolvable external repos
	// (anything not in ExternalRepos) and for missing files return an
	// empty StringDict + nil error rather than aborting the eval.
	// Use this when extracting structural info (rule attrs, provider
	// fields, etc.) from .bzl files that pull from external Bazel
	// modules you don't want to materialize.
	//
	// Faithful execution mode (default) still errors on unresolvable
	// loads.
	LenientLoad bool
}
