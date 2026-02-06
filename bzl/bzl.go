// Package bzl provides the main user-facing API for evaluating Bazel Starlark files.
package bzl

import (
	"strings"

	"github.com/albertocavalcante/starlark-go-bazel/eval"
	"github.com/albertocavalcante/starlark-go-bazel/loader"
	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
)

// Interpreter is the main entry point for evaluating Bazel Starlark.
type Interpreter struct {
	evaluator *eval.Evaluator
	options   Options
	fsLoader  *loader.FileSystemLoader
}

// New creates a new Interpreter with the given options.
func New(opts Options) *Interpreter {
	if opts.FileSystem == nil {
		opts.FileSystem = loader.NewOSFileSystem(opts.WorkspaceRoot)
	}

	fsLoader := loader.NewFileSystemLoader(opts.FileSystem)

	// Create a BzlFileLoader for loading .bzl files
	bzlLoader := loader.NewBzlFileLoader(
		opts.FileSystem,
		opts.WorkspaceRoot,
		loader.WithRepoMapping(opts.ExternalRepos),
	)

	evalOpts := eval.Options{
		BzlLoader:    bzlLoader,
		FileLoader:   fsLoader,
		PrintHandler: opts.PrintHandler,
	}

	return &Interpreter{
		evaluator: eval.New(evalOpts),
		options:   opts,
		fsLoader:  fsLoader,
	}
}

// Result contains evaluation results.
type Result struct {
	// Globals contains exported symbols (for .bzl files)
	Globals starlark.StringDict
	// Targets contains declared targets (for BUILD files)
	Targets map[string]*types.RuleInstance
}

// EvalFile evaluates a Starlark file (auto-detects .bzl vs BUILD).
func (i *Interpreter) EvalFile(path string) (*Result, error) {
	if i.isBuildFile(path) {
		buildResult, err := i.evaluator.EvalBuildFile(path)
		if err != nil {
			return nil, err
		}
		return &Result{
			Globals: buildResult.Globals,
			Targets: buildResult.Targets,
		}, nil
	}

	bzlResult, err := i.evaluator.EvalBzlFile(path)
	if err != nil {
		return nil, err
	}
	return &Result{
		Globals: bzlResult.Globals,
		Targets: nil,
	}, nil
}

// Eval evaluates Starlark source code.
func (i *Interpreter) Eval(filename string, source []byte) (*Result, error) {
	if i.isBuildFile(filename) {
		buildResult, err := i.evaluator.EvalBuild(filename, source)
		if err != nil {
			return nil, err
		}
		return &Result{
			Globals: buildResult.Globals,
			Targets: buildResult.Targets,
		}, nil
	}

	bzlResult, err := i.evaluator.EvalBzl(filename, source)
	if err != nil {
		return nil, err
	}
	return &Result{
		Globals: bzlResult.Globals,
		Targets: nil,
	}, nil
}

// Options returns the interpreter's options.
func (i *Interpreter) Options() Options {
	return i.options
}

// isBuildFile returns true if the filename indicates a BUILD file.
func (i *Interpreter) isBuildFile(filename string) bool {
	base := filename
	if idx := strings.LastIndex(filename, "/"); idx != -1 {
		base = filename[idx+1:]
	}
	return base == "BUILD" || base == "BUILD.bazel"
}
