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
}
