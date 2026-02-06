// Package eval provides the Starlark evaluation engine for Bazel files.
//
// This package implements evaluation of BUILD and .bzl files following Bazel's semantics.
// It provides separate evaluation paths for:
// - BUILD files: Creates targets and collects declared rules
// - .bzl files: Exports globals (functions, providers, etc.) for loading
package eval

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/albertocavalcante/starlark-go-bazel/loader"
	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Evaluator evaluates Starlark files (BUILD and .bzl).
type Evaluator struct {
	bzlLoader        loader.BzlLoader
	fileLoader       loader.Loader
	predeclaredBzl   starlark.StringDict
	predeclaredBuild starlark.StringDict
	printHandler     func(msg string)
	cache            map[string]*CachedModule
}

// CachedModule holds a cached module evaluation result.
type CachedModule struct {
	Globals starlark.StringDict
	Err     error
}

// Options configures the Evaluator.
type Options struct {
	BzlLoader        loader.BzlLoader
	FileLoader       loader.Loader
	PredeclaredBzl   starlark.StringDict
	PredeclaredBuild starlark.StringDict
	PrintHandler     func(msg string)
}

// New creates a new Evaluator.
func New(opts Options) *Evaluator {
	predeclaredBzl := makeBzlPredeclared()
	for k, v := range opts.PredeclaredBzl {
		predeclaredBzl[k] = v
	}

	predeclaredBuild := makeBuildPredeclared()
	for k, v := range opts.PredeclaredBuild {
		predeclaredBuild[k] = v
	}

	return &Evaluator{
		bzlLoader:        opts.BzlLoader,
		fileLoader:       opts.FileLoader,
		predeclaredBzl:   predeclaredBzl,
		predeclaredBuild: predeclaredBuild,
		printHandler:     opts.PrintHandler,
		cache:            make(map[string]*CachedModule),
	}
}

// BzlResult contains the result of evaluating a .bzl file.
type BzlResult struct {
	Globals starlark.StringDict
}

// BuildResult contains the result of evaluating a BUILD file.
type BuildResult struct {
	Targets map[string]*types.RuleInstance
	Globals starlark.StringDict
	Package string
}

// EvalBzl evaluates a .bzl file and returns its exports.
func (e *Evaluator) EvalBzl(path string, source []byte) (*BzlResult, error) {
	dir := filepath.Dir(path)
	pkg := strings.TrimPrefix(dir, "/")
	if pkg == "." {
		pkg = ""
	}

	thread := &starlark.Thread{
		Name:  path,
		Print: e.makePrintHandler(),
	}

	if e.bzlLoader != nil {
		thread.Load = loader.MakeLoadFunc(e.bzlLoader)
		loader.SetBzlLoader(thread, e.bzlLoader)
	} else {
		thread.Load = e.makeLoadFunc()
	}
	loader.SetCurrentPackage(thread, pkg)

	globals, err := starlark.ExecFile(thread, path, source, e.predeclaredBzl)
	if err != nil {
		return nil, fmt.Errorf("evaluating %s: %w", path, err)
	}

	return &BzlResult{Globals: globals}, nil
}

// EvalBuild evaluates a BUILD file and returns its targets.
func (e *Evaluator) EvalBuild(path string, source []byte) (*BuildResult, error) {
	dir := filepath.Dir(path)
	pkg := strings.TrimPrefix(dir, "/")
	if pkg == "." {
		pkg = ""
	}

	thread := &starlark.Thread{
		Name:  path,
		Print: e.makePrintHandler(),
	}

	if e.bzlLoader != nil {
		thread.Load = loader.MakeLoadFunc(e.bzlLoader)
		loader.SetBzlLoader(thread, e.bzlLoader)
	} else {
		thread.Load = e.makeLoadFunc()
	}
	loader.SetCurrentPackage(thread, pkg)

	targets := make(map[string]*types.RuleInstance)
	thread.SetLocal("targets", targets)

	globals, err := starlark.ExecFile(thread, path, source, e.predeclaredBuild)
	if err != nil {
		return nil, fmt.Errorf("evaluating %s: %w", path, err)
	}

	return &BuildResult{
		Targets: targets,
		Globals: globals,
		Package: pkg,
	}, nil
}

// EvalBzlFile loads and evaluates a .bzl file from the filesystem.
func (e *Evaluator) EvalBzlFile(path string) (*BzlResult, error) {
	if e.fileLoader == nil {
		return nil, fmt.Errorf("no file loader configured")
	}
	source, err := e.fileLoader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", path, err)
	}
	return e.EvalBzl(path, source)
}

// EvalBuildFile loads and evaluates a BUILD file from the filesystem.
func (e *Evaluator) EvalBuildFile(path string) (*BuildResult, error) {
	if e.fileLoader == nil {
		return nil, fmt.Errorf("no file loader configured")
	}
	source, err := e.fileLoader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", path, err)
	}
	return e.EvalBuild(path, source)
}

func (e *Evaluator) makePrintHandler() func(*starlark.Thread, string) {
	return func(_ *starlark.Thread, msg string) {
		if e.printHandler != nil {
			e.printHandler(msg)
		}
	}
}

func (e *Evaluator) makeLoadFunc() func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	return func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		if cached, ok := e.cache[module]; ok {
			return cached.Globals, cached.Err
		}

		if e.fileLoader == nil {
			return nil, fmt.Errorf("no loader configured for module %q", module)
		}

		source, err := e.fileLoader.Load(module)
		if err != nil {
			e.cache[module] = &CachedModule{Err: err}
			return nil, err
		}

		newThread := &starlark.Thread{
			Name:  module,
			Load:  e.makeLoadFunc(),
			Print: thread.Print,
		}

		globals, err := starlark.ExecFile(newThread, module, source, e.predeclaredBzl)
		e.cache[module] = &CachedModule{Globals: globals, Err: err}

		return globals, err
	}
}

func makeBzlPredeclared() starlark.StringDict {
	return starlark.StringDict{
		"Label":    starlark.NewBuiltin("Label", types.LabelBuiltin),
		"provider": starlark.NewBuiltin("provider", providerBuiltin),
		"struct":   starlark.NewBuiltin("struct", starlarkstruct.Make),
		"depset":   starlark.NewBuiltin("depset", types.DepsetBuiltin),
		"rule":     starlark.NewBuiltin("rule", types.RuleBuiltin),
		"attr":     newAttrModule(),
		"True":     starlark.True,
		"False":    starlark.False,
		"None":     starlark.None,
	}
}

func makeBuildPredeclared() starlark.StringDict {
	return starlark.StringDict{
		"Label":         starlark.NewBuiltin("Label", types.LabelBuiltin),
		"struct":        starlark.NewBuiltin("struct", starlarkstruct.Make),
		"depset":        starlark.NewBuiltin("depset", types.DepsetBuiltin),
		"package":       starlark.NewBuiltin("package", PackageBuiltin),
		"licenses":      starlark.NewBuiltin("licenses", LicensesBuiltin),
		"exports_files": starlark.NewBuiltin("exports_files", ExportsFilesBuiltin),
		"glob":          starlark.NewBuiltin("glob", GlobBuiltin),
		"True":          starlark.True,
		"False":         starlark.False,
		"None":          starlark.None,
	}
}

func providerBuiltin(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		doc    string
		fields *starlark.List
		init   starlark.Callable
	)

	if err := starlark.UnpackArgs("provider", args, kwargs,
		"doc?", &doc,
		"fields?", &fields,
		"init?", &init,
	); err != nil {
		return nil, err
	}

	var fieldNames []string
	if fields != nil {
		iter := fields.Iterate()
		defer iter.Done()
		var v starlark.Value
		for iter.Next(&v) {
			s, ok := v.(starlark.String)
			if !ok {
				return nil, fmt.Errorf("provider: fields must be strings, got %s", v.Type())
			}
			fieldNames = append(fieldNames, string(s))
		}
	}

	return types.NewProvider("", fieldNames, doc, init), nil
}

type attrModule struct{}

var _ starlark.HasAttrs = (*attrModule)(nil)

func newAttrModule() *attrModule {
	return &attrModule{}
}

func (m *attrModule) String() string        { return "<module attr>" }
func (m *attrModule) Type() string          { return "module" }
func (m *attrModule) Freeze()               {}
func (m *attrModule) Truth() starlark.Bool  { return true }
func (m *attrModule) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: module") }

func (m *attrModule) Attr(name string) (starlark.Value, error) {
	switch name {
	case "string":
		return starlark.NewBuiltin("attr.string", attrFactory(types.AttrTypeString)), nil
	case "string_list":
		return starlark.NewBuiltin("attr.string_list", attrFactory(types.AttrTypeStringList)), nil
	case "int":
		return starlark.NewBuiltin("attr.int", attrFactory(types.AttrTypeInt)), nil
	case "bool":
		return starlark.NewBuiltin("attr.bool", attrFactory(types.AttrTypeBool)), nil
	case "label":
		return starlark.NewBuiltin("attr.label", attrFactory(types.AttrTypeLabel)), nil
	case "label_list":
		return starlark.NewBuiltin("attr.label_list", attrFactory(types.AttrTypeLabelList)), nil
	case "output":
		return starlark.NewBuiltin("attr.output", attrFactory(types.AttrTypeOutput)), nil
	case "output_list":
		return starlark.NewBuiltin("attr.output_list", attrFactory(types.AttrTypeOutputList)), nil
	case "string_dict":
		return starlark.NewBuiltin("attr.string_dict", attrFactory(types.AttrTypeStringDict)), nil
	default:
		return nil, starlark.NoSuchAttrError(fmt.Sprintf("attr has no attribute %q", name))
	}
}

func (m *attrModule) AttrNames() []string {
	return []string{"bool", "int", "label", "label_list", "output", "output_list", "string", "string_dict", "string_list"}
}

func attrFactory(attrType types.AttrType) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	return func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(args) > 0 {
			return nil, fmt.Errorf("%s: unexpected positional arguments", b.Name())
		}

		desc := &types.AttrDescriptor{
			Type:       attrType,
			AllowEmpty: true,
		}

		for _, kv := range kwargs {
			key := string(kv[0].(starlark.String))
			val := kv[1]

			switch key {
			case "mandatory":
				if b, ok := val.(starlark.Bool); ok {
					desc.Mandatory = bool(b)
				}
			case "default":
				desc.Default = val
			case "doc":
				if s, ok := val.(starlark.String); ok {
					desc.Doc = string(s)
				}
			case "allow_empty":
				if b, ok := val.(starlark.Bool); ok {
					desc.AllowEmpty = bool(b)
				}
			case "allow_files", "allow_single_file", "executable", "providers":
				// Handle these options
			}
		}

		return &attrDescriptorValue{desc: desc}, nil
	}
}

type attrDescriptorValue struct {
	desc *types.AttrDescriptor
}

var _ starlark.Value = (*attrDescriptorValue)(nil)

func (a *attrDescriptorValue) String() string       { return fmt.Sprintf("<attr.%s>", a.desc.Type) }
func (a *attrDescriptorValue) Type() string         { return "Attribute" }
func (a *attrDescriptorValue) Freeze()              {}
func (a *attrDescriptorValue) Truth() starlark.Bool { return true }
func (a *attrDescriptorValue) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: Attribute")
}
func (a *attrDescriptorValue) Descriptor() *types.AttrDescriptor { return a.desc }
