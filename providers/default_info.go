// Package providers implements Bazel's built-in providers.
//
// DefaultInfo implementation based on:
// - bazel/src/main/java/com/google/devtools/build/lib/analysis/DefaultInfo.java
// - bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/DefaultInfoApi.java
package providers

import (
	"fmt"

	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
)

// DefaultInfoProvider is the singleton provider type for DefaultInfo.
// Reference: DefaultInfo.java: PROVIDER field
var DefaultInfoProvider = types.NewProvider("DefaultInfo", []string{
	"files",
	"runfiles",
	"data_runfiles",
	"default_runfiles",
	"executable",
	"files_to_run",
}, "A provider that gives general information about a target's direct and transitive files.", nil)

// DefaultInfo is provided by all targets implicitly and contains all standard fields.
//
// Reference: DefaultInfo.java (DefaultDefaultInfo inner class)
//
// Fields (from DefaultInfoApi.java):
//   - files: depset of File objects representing default outputs to build
//   - runfiles: Runfiles descriptor (legacy, use default_runfiles instead)
//   - data_runfiles: Runfiles for when this target is a data dependency
//   - default_runfiles: Runfiles for when running this target
//   - executable: File to execute for executable/test rules
//   - files_to_run: FilesToRunProvider (not implemented yet)
type DefaultInfo struct {
	// files is a depset of Files representing the default outputs
	// From DefaultInfo.java: files field (Depset)
	files *types.Depset

	// runfiles is the legacy runfiles (use default_runfiles instead)
	// From DefaultInfo.java: runfiles field (statelessRunfiles)
	runfiles *Runfiles

	// dataRunfiles is runfiles for data dependencies
	// From DefaultInfo.java: dataRunfiles field
	dataRunfiles *Runfiles

	// defaultRunfiles is the standard runfiles
	// From DefaultInfo.java: defaultRunfiles field
	defaultRunfiles *Runfiles

	// executable is the file to execute
	// From DefaultInfo.java: executable field (Artifact)
	executable *types.File

	frozen bool
}

var (
	_ starlark.Value    = (*DefaultInfo)(nil)
	_ starlark.HasAttrs = (*DefaultInfo)(nil)
)

// NewDefaultInfo creates a new DefaultInfo.
func NewDefaultInfo() *DefaultInfo {
	return &DefaultInfo{}
}

// String returns the Starlark representation.
func (d *DefaultInfo) String() string {
	return fmt.Sprintf("DefaultInfo(files = %v, default_runfiles = %v)",
		d.files, d.defaultRunfiles)
}

// Type returns "DefaultInfo".
func (d *DefaultInfo) Type() string { return "DefaultInfo" }

// Freeze marks the info as frozen.
func (d *DefaultInfo) Freeze() {
	if d.frozen {
		return
	}
	d.frozen = true
	if d.files != nil {
		d.files.Freeze()
	}
	if d.runfiles != nil {
		d.runfiles.Freeze()
	}
	if d.dataRunfiles != nil {
		d.dataRunfiles.Freeze()
	}
	if d.defaultRunfiles != nil {
		d.defaultRunfiles.Freeze()
	}
	if d.executable != nil {
		d.executable.Freeze()
	}
}

// Truth returns true.
func (d *DefaultInfo) Truth() starlark.Bool { return true }

// Hash returns an error.
func (d *DefaultInfo) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: DefaultInfo")
}

// Attr returns an attribute of the DefaultInfo.
// Reference: DefaultInfoApi.java interface methods
func (d *DefaultInfo) Attr(name string) (starlark.Value, error) {
	switch name {
	case "files":
		// From DefaultInfoApi.java: getFiles()
		// "A depset of File objects representing the default outputs to build"
		if d.files == nil {
			return starlark.None, nil
		}
		return d.files, nil

	case "runfiles":
		// Legacy runfiles field
		if d.runfiles == nil {
			return starlark.None, nil
		}
		return d.runfiles, nil

	case "data_runfiles":
		// From DefaultInfoApi.java: getDataRunfiles()
		// "runfiles descriptor for when this target is a data dependency"
		if d.dataRunfiles == nil {
			return starlark.None, nil
		}
		return d.dataRunfiles, nil

	case "default_runfiles":
		// From DefaultInfoApi.java: getDefaultRunfiles()
		// "runfiles descriptor for when running this target"
		// From DefaultInfo.java: if data_runfiles and default_runfiles are null,
		// return the legacy runfiles field
		if d.dataRunfiles == nil && d.defaultRunfiles == nil {
			if d.runfiles != nil {
				return d.runfiles, nil
			}
			return starlark.None, nil
		}
		if d.defaultRunfiles == nil {
			return starlark.None, nil
		}
		return d.defaultRunfiles, nil

	case "executable":
		// From DefaultInfoApi.java: (via constructor parameter)
		// "File object representing the file to execute"
		if d.executable == nil {
			return starlark.None, nil
		}
		return d.executable, nil

	case "files_to_run":
		// From DefaultInfoApi.java: getFilesToRun()
		// Not fully implemented yet
		return starlark.None, nil

	default:
		return nil, starlark.NoSuchAttrError(fmt.Sprintf("DefaultInfo has no attribute %q", name))
	}
}

// AttrNames returns the list of attribute names.
func (d *DefaultInfo) AttrNames() []string {
	return []string{
		"data_runfiles",
		"default_runfiles",
		"executable",
		"files",
		"files_to_run",
		"runfiles",
	}
}

// Files returns the files depset.
func (d *DefaultInfo) Files() *types.Depset { return d.files }

// SetFiles sets the files depset.
func (d *DefaultInfo) SetFiles(files *types.Depset) {
	if d.frozen {
		return
	}
	d.files = files
}

// Runfiles returns the legacy runfiles.
func (d *DefaultInfo) Runfiles() *Runfiles { return d.runfiles }

// SetRunfiles sets the legacy runfiles.
func (d *DefaultInfo) SetRunfiles(runfiles *Runfiles) {
	if d.frozen {
		return
	}
	d.runfiles = runfiles
}

// DataRunfiles returns the data runfiles.
func (d *DefaultInfo) DataRunfiles() *Runfiles { return d.dataRunfiles }

// SetDataRunfiles sets the data runfiles.
func (d *DefaultInfo) SetDataRunfiles(runfiles *Runfiles) {
	if d.frozen {
		return
	}
	d.dataRunfiles = runfiles
}

// DefaultRunfiles returns the default runfiles.
func (d *DefaultInfo) DefaultRunfiles() *Runfiles { return d.defaultRunfiles }

// SetDefaultRunfiles sets the default runfiles.
func (d *DefaultInfo) SetDefaultRunfiles(runfiles *Runfiles) {
	if d.frozen {
		return
	}
	d.defaultRunfiles = runfiles
}

// Executable returns the executable file.
func (d *DefaultInfo) Executable() *types.File { return d.executable }

// SetExecutable sets the executable file.
func (d *DefaultInfo) SetExecutable(executable *types.File) {
	if d.frozen {
		return
	}
	d.executable = executable
}

// Provider returns the DefaultInfo provider.
func (d *DefaultInfo) Provider() *types.Provider {
	return DefaultInfoProvider
}

// DefaultInfoBuiltin is the Starlark constructor for DefaultInfo.
// Reference: DefaultInfo.java DefaultInfoProvider.constructor() method
//
// Parameters (from DefaultInfoApi.java):
//   - files: depset of File objects (default outputs)
//   - runfiles: Runfiles (legacy, prefer default_runfiles)
//   - data_runfiles: Runfiles for data dependencies
//   - default_runfiles: Runfiles for running the target
//   - executable: File to execute
func DefaultInfoBuiltin(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// DefaultInfo only accepts keyword arguments (no positional args)
	if len(args) > 0 {
		return nil, fmt.Errorf("DefaultInfo: unexpected positional arguments")
	}

	var (
		filesObj           starlark.Value = starlark.None
		runfilesObj        starlark.Value = starlark.None
		dataRunfilesObj    starlark.Value = starlark.None
		defaultRunfilesObj starlark.Value = starlark.None
		executableObj      starlark.Value = starlark.None
	)

	for _, kv := range kwargs {
		key := string(kv[0].(starlark.String))
		switch key {
		case "files":
			filesObj = kv[1]
		case "runfiles":
			runfilesObj = kv[1]
		case "data_runfiles":
			dataRunfilesObj = kv[1]
		case "default_runfiles":
			defaultRunfilesObj = kv[1]
		case "executable":
			executableObj = kv[1]
		default:
			return nil, fmt.Errorf("DefaultInfo: unexpected keyword argument %q", key)
		}
	}

	info := NewDefaultInfo()

	// Validate and set files
	// Reference: DefaultInfo.java constructor - castNoneToNull(Depset.class, files)
	if filesObj != starlark.None {
		files, ok := filesObj.(*types.Depset)
		if !ok {
			return nil, fmt.Errorf("DefaultInfo: files must be a depset, got %s", filesObj.Type())
		}
		info.files = files
	}

	// Parse runfiles objects
	// Reference: DefaultInfo.java constructor validation
	var runfiles, dataRunfiles, defaultRunfiles *Runfiles

	if runfilesObj != starlark.None {
		rf, ok := runfilesObj.(*Runfiles)
		if !ok {
			return nil, fmt.Errorf("DefaultInfo: runfiles must be a runfiles object, got %s", runfilesObj.Type())
		}
		runfiles = rf
	}

	if dataRunfilesObj != starlark.None {
		rf, ok := dataRunfilesObj.(*Runfiles)
		if !ok {
			return nil, fmt.Errorf("DefaultInfo: data_runfiles must be a runfiles object, got %s", dataRunfilesObj.Type())
		}
		dataRunfiles = rf
	}

	if defaultRunfilesObj != starlark.None {
		rf, ok := defaultRunfilesObj.(*Runfiles)
		if !ok {
			return nil, fmt.Errorf("DefaultInfo: default_runfiles must be a runfiles object, got %s", defaultRunfilesObj.Type())
		}
		defaultRunfiles = rf
	}

	// Reference: DefaultInfo.java constructor validation:
	// "Cannot specify the provider 'runfiles' together with 'data_runfiles' or 'default_runfiles'"
	if runfiles != nil && (dataRunfiles != nil || defaultRunfiles != nil) {
		return nil, fmt.Errorf("DefaultInfo: cannot specify 'runfiles' together with 'data_runfiles' or 'default_runfiles'")
	}

	info.runfiles = runfiles
	info.dataRunfiles = dataRunfiles
	info.defaultRunfiles = defaultRunfiles

	// Validate and set executable
	if executableObj != starlark.None {
		exe, ok := executableObj.(*types.File)
		if !ok {
			return nil, fmt.Errorf("DefaultInfo: executable must be a File, got %s", executableObj.Type())
		}
		info.executable = exe
	}

	return info, nil
}

// CreateDefaultInfoEmpty creates an empty DefaultInfo.
// Reference: DefaultInfo.java createEmpty() method
func CreateDefaultInfoEmpty() *DefaultInfo {
	return NewDefaultInfo()
}
