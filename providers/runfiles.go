// Package providers implements Bazel's built-in providers.
//
// Runfiles implementation based on:
// - bazel/src/main/java/com/google/devtools/build/lib/analysis/Runfiles.java
// - bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/RunfilesApi.java
package providers

import (
	"fmt"

	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
)

// Runfiles represents a container of information regarding a set of files
// required at runtime by an executable. This object should be passed via
// DefaultInfo to tell the build system about the runfiles needed by the
// outputs produced by the rule.
//
// Conceptually, the runfiles are a map of paths to files, forming a symlink tree.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/analysis/Runfiles.java
type Runfiles struct {
	// prefix is the directory to put all runfiles under (workspace name)
	// From Runfiles.java: prefix field
	prefix string

	// files is the depset of artifacts that should be present in the runfiles directory
	// From Runfiles.java: artifacts field (NestedSet<Artifact>)
	files *types.Depset

	// symlinks maps paths to artifacts for symlinks in the runfiles directory
	// From Runfiles.java: symlinks field (NestedSet<SymlinkEntry>)
	symlinks *types.Depset

	// rootSymlinks maps paths to artifacts for symlinks above the runfiles directory
	// From Runfiles.java: rootSymlinks field (NestedSet<SymlinkEntry>)
	rootSymlinks *types.Depset

	// emptyFilenames is the depset of filenames for empty files to create
	// From Runfiles.java: getEmptyFilenames() method
	emptyFilenames *types.Depset

	frozen bool
}

var (
	_ starlark.Value    = (*Runfiles)(nil)
	_ starlark.HasAttrs = (*Runfiles)(nil)
)

// emptyDepset creates an empty depset.
func emptyDepset() *types.Depset {
	d, _ := types.NewDepset(types.OrderDefault, nil, nil)
	return d
}

// EmptyRunfiles is an empty Runfiles instance.
// Reference: Runfiles.java: EMPTY field
var EmptyRunfiles = &Runfiles{
	prefix:         "",
	files:          emptyDepset(),
	symlinks:       emptyDepset(),
	rootSymlinks:   emptyDepset(),
	emptyFilenames: emptyDepset(),
	frozen:         true,
}

// NewRunfiles creates a new Runfiles with the given prefix (workspace name).
func NewRunfiles(prefix string) *Runfiles {
	return &Runfiles{
		prefix:         prefix,
		files:          emptyDepset(),
		symlinks:       emptyDepset(),
		rootSymlinks:   emptyDepset(),
		emptyFilenames: emptyDepset(),
	}
}

// String returns the Starlark representation.
// Reference: Runfiles.java debugPrint() method
func (r *Runfiles) String() string {
	return fmt.Sprintf("Runfiles(empty_files = %s, files = %s, root_symlinks = %s, symlinks = %s)",
		r.emptyFilenames.String(),
		r.files.String(),
		r.rootSymlinks.String(),
		r.symlinks.String())
}

// Type returns "runfiles".
func (r *Runfiles) Type() string { return "runfiles" }

// Freeze marks the runfiles as frozen.
func (r *Runfiles) Freeze() {
	if r.frozen {
		return
	}
	r.frozen = true
	r.files.Freeze()
	r.symlinks.Freeze()
	r.rootSymlinks.Freeze()
	r.emptyFilenames.Freeze()
}

// Truth returns true if the runfiles is non-empty.
func (r *Runfiles) Truth() starlark.Bool {
	return starlark.Bool(!r.IsEmpty())
}

// Hash returns an error (runfiles are not hashable, but are immutable).
func (r *Runfiles) Hash() (uint32, error) {
	// Runfiles are immutable and Starlark-hashable according to Runfiles.java isImmutable()
	// But for simplicity, we return an error
	return 0, fmt.Errorf("unhashable type: runfiles")
}

// Attr returns an attribute of the runfiles.
// Reference: RunfilesApi.java interface methods
func (r *Runfiles) Attr(name string) (starlark.Value, error) {
	switch name {
	case "files":
		// From RunfilesApi.java: getArtifactsForStarlark()
		// "Returns the set of runfiles as files"
		return r.files, nil

	case "symlinks":
		// From RunfilesApi.java: getSymlinksForStarlark()
		// "Returns the set of symlinks"
		return r.symlinks, nil

	case "root_symlinks":
		// From RunfilesApi.java: getRootSymlinksForStarlark()
		// "Returns the set of root symlinks"
		return r.rootSymlinks, nil

	case "empty_filenames":
		// From RunfilesApi.java: getEmptyFilenamesForStarlark()
		// "Returns names of empty files to create"
		return r.emptyFilenames, nil

	case "merge":
		// From RunfilesApi.java: merge()
		return starlark.NewBuiltin("runfiles.merge", r.mergeMethod), nil

	case "merge_all":
		// From RunfilesApi.java: mergeAll()
		return starlark.NewBuiltin("runfiles.merge_all", r.mergeAllMethod), nil

	default:
		return nil, starlark.NoSuchAttrError(fmt.Sprintf("runfiles has no attribute %q", name))
	}
}

// AttrNames returns the list of attribute names.
func (r *Runfiles) AttrNames() []string {
	return []string{
		"empty_filenames",
		"files",
		"merge",
		"merge_all",
		"root_symlinks",
		"symlinks",
	}
}

// Prefix returns the runfiles prefix (workspace name).
func (r *Runfiles) Prefix() string { return r.prefix }

// Files returns the files depset.
func (r *Runfiles) Files() *types.Depset { return r.files }

// Symlinks returns the symlinks depset.
func (r *Runfiles) Symlinks() *types.Depset { return r.symlinks }

// RootSymlinks returns the root symlinks depset.
func (r *Runfiles) RootSymlinks() *types.Depset { return r.rootSymlinks }

// EmptyFilenames returns the empty filenames depset.
func (r *Runfiles) EmptyFilenames() *types.Depset { return r.emptyFilenames }

// IsEmpty returns true if there are no runfiles.
// Reference: Runfiles.java isEmpty() method
func (r *Runfiles) IsEmpty() bool {
	return !bool(r.files.Truth()) && !bool(r.symlinks.Truth()) && !bool(r.rootSymlinks.Truth())
}

// SetFiles sets the files depset.
func (r *Runfiles) SetFiles(files *types.Depset) {
	if r.frozen {
		return
	}
	r.files = files
}

// SetSymlinks sets the symlinks depset.
func (r *Runfiles) SetSymlinks(symlinks *types.Depset) {
	if r.frozen {
		return
	}
	r.symlinks = symlinks
}

// SetRootSymlinks sets the root symlinks depset.
func (r *Runfiles) SetRootSymlinks(rootSymlinks *types.Depset) {
	if r.frozen {
		return
	}
	r.rootSymlinks = rootSymlinks
}

// SetEmptyFilenames sets the empty filenames depset.
func (r *Runfiles) SetEmptyFilenames(emptyFilenames *types.Depset) {
	if r.frozen {
		return
	}
	r.emptyFilenames = emptyFilenames
}

// Merge returns a new runfiles object that includes all contents of this one and the argument.
// Reference: Runfiles.java merge() method
func (r *Runfiles) Merge(other *Runfiles) (*Runfiles, error) {
	if r.IsEmpty() {
		return other, nil
	}
	if other.IsEmpty() {
		return r, nil
	}

	// Use the prefix from non-empty runfiles
	prefix := r.prefix
	if prefix == "" {
		prefix = other.prefix
	}

	result := NewRunfiles(prefix)

	// Merge files
	files, err := types.NewDepset(types.OrderDefault, nil, []*types.Depset{r.files, other.files})
	if err != nil {
		return nil, err
	}
	result.files = files

	// Merge symlinks
	symlinks, err := types.NewDepset(types.OrderDefault, nil, []*types.Depset{r.symlinks, other.symlinks})
	if err != nil {
		return nil, err
	}
	result.symlinks = symlinks

	// Merge root symlinks
	rootSymlinks, err := types.NewDepset(types.OrderDefault, nil, []*types.Depset{r.rootSymlinks, other.rootSymlinks})
	if err != nil {
		return nil, err
	}
	result.rootSymlinks = rootSymlinks

	// Merge empty filenames
	emptyFilenames, err := types.NewDepset(types.OrderDefault, nil, []*types.Depset{r.emptyFilenames, other.emptyFilenames})
	if err != nil {
		return nil, err
	}
	result.emptyFilenames = emptyFilenames

	return result, nil
}

// mergeMethod implements the Starlark merge() method.
// Reference: RunfilesApi.java merge()
func (r *Runfiles) mergeMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var other *Runfiles
	if err := starlark.UnpackArgs("runfiles.merge", args, kwargs, "other", &other); err != nil {
		return nil, err
	}
	return r.Merge(other)
}

// MergeAll returns a new runfiles object that includes all contents of this one and the sequence.
// Reference: Runfiles.java mergeAll() method
func (r *Runfiles) MergeAll(others []*Runfiles) (*Runfiles, error) {
	// When merging exactly one non-empty Runfiles object, return that object
	var result *Runfiles
	var uniqueNonEmpty *Runfiles

	if !r.IsEmpty() {
		result = r
		uniqueNonEmpty = r
	}

	for _, other := range others {
		if !other.IsEmpty() {
			if result == nil {
				result = other
				uniqueNonEmpty = other
			} else {
				uniqueNonEmpty = nil
				var err error
				result, err = result.Merge(other)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if uniqueNonEmpty != nil {
		return uniqueNonEmpty, nil
	}
	if result != nil {
		return result, nil
	}
	return EmptyRunfiles, nil
}

// mergeAllMethod implements the Starlark merge_all() method.
// Reference: RunfilesApi.java mergeAll()
func (r *Runfiles) mergeAllMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var otherList *starlark.List
	if err := starlark.UnpackArgs("runfiles.merge_all", args, kwargs, "other", &otherList); err != nil {
		return nil, err
	}

	others := make([]*Runfiles, 0, otherList.Len())
	iter := otherList.Iterate()
	defer iter.Done()
	var v starlark.Value
	for iter.Next(&v) {
		rf, ok := v.(*Runfiles)
		if !ok {
			return nil, fmt.Errorf("merge_all: expected runfiles, got %s", v.Type())
		}
		others = append(others, rf)
	}

	return r.MergeAll(others)
}

// RunfilesBuilder helps construct Runfiles objects.
// Reference: Runfiles.Builder in Runfiles.java
type RunfilesBuilder struct {
	prefix       string
	files        []starlark.Value
	symlinks     []*types.SymlinkEntry
	rootSymlinks []*types.SymlinkEntry
	transitive   []*Runfiles
}

// NewRunfilesBuilder creates a new RunfilesBuilder.
func NewRunfilesBuilder(prefix string) *RunfilesBuilder {
	return &RunfilesBuilder{prefix: prefix}
}

// AddFile adds a file to the runfiles.
func (rb *RunfilesBuilder) AddFile(f *types.File) {
	rb.files = append(rb.files, f)
}

// AddFiles adds multiple files to the runfiles.
func (rb *RunfilesBuilder) AddFiles(files []*types.File) {
	for _, f := range files {
		rb.files = append(rb.files, f)
	}
}

// AddTransitiveFiles adds a depset of files.
func (rb *RunfilesBuilder) AddTransitiveFiles(files *types.Depset) {
	for _, v := range files.ToList() {
		rb.files = append(rb.files, v)
	}
}

// AddSymlink adds a symlink.
func (rb *RunfilesBuilder) AddSymlink(link string, target *types.File) {
	rb.symlinks = append(rb.symlinks, types.NewSymlinkEntry(link, target))
}

// AddRootSymlink adds a root symlink.
func (rb *RunfilesBuilder) AddRootSymlink(link string, target *types.File) {
	rb.rootSymlinks = append(rb.rootSymlinks, types.NewSymlinkEntry(link, target))
}

// Merge merges another Runfiles.
func (rb *RunfilesBuilder) Merge(other *Runfiles) {
	rb.transitive = append(rb.transitive, other)
}

// Build creates the Runfiles.
func (rb *RunfilesBuilder) Build() (*Runfiles, error) {
	r := NewRunfiles(rb.prefix)

	// Create depset of files
	var transitiveFiles []*types.Depset
	for _, t := range rb.transitive {
		transitiveFiles = append(transitiveFiles, t.files)
	}
	files, err := types.NewDepset(types.OrderDefault, rb.files, transitiveFiles)
	if err != nil {
		return nil, err
	}
	r.files = files

	// Create depset of symlinks
	symlinkValues := make([]starlark.Value, len(rb.symlinks))
	for i, s := range rb.symlinks {
		symlinkValues[i] = s
	}
	var transitiveSymlinks []*types.Depset
	for _, t := range rb.transitive {
		transitiveSymlinks = append(transitiveSymlinks, t.symlinks)
	}
	symlinks, err := types.NewDepset(types.OrderDefault, symlinkValues, transitiveSymlinks)
	if err != nil {
		return nil, err
	}
	r.symlinks = symlinks

	// Create depset of root symlinks
	rootSymlinkValues := make([]starlark.Value, len(rb.rootSymlinks))
	for i, s := range rb.rootSymlinks {
		rootSymlinkValues[i] = s
	}
	var transitiveRootSymlinks []*types.Depset
	for _, t := range rb.transitive {
		transitiveRootSymlinks = append(transitiveRootSymlinks, t.rootSymlinks)
	}
	rootSymlinks, err := types.NewDepset(types.OrderDefault, rootSymlinkValues, transitiveRootSymlinks)
	if err != nil {
		return nil, err
	}
	r.rootSymlinks = rootSymlinks

	return r, nil
}

// RunfilesBuiltin creates a runfiles object (typically called via ctx.runfiles).
// This is the Starlark constructor for runfiles.
//
// Reference: Runfiles.Builder in Runfiles.java
//
// Parameters (from ctx.runfiles):
//   - files: List of Files to include
//   - transitive_files: Depset of Files to include transitively
//   - symlinks: Dict mapping paths to Files
//   - root_symlinks: Dict mapping paths to Files (at runfiles root)
func RunfilesBuiltin(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		filesList       *starlark.List
		transitiveFiles *types.Depset
		symlinksDict    *starlark.Dict
		rootSymlinksArg *starlark.Dict
	)

	if err := starlark.UnpackArgs("runfiles", args, kwargs,
		"files?", &filesList,
		"transitive_files?", &transitiveFiles,
		"symlinks?", &symlinksDict,
		"root_symlinks?", &rootSymlinksArg,
	); err != nil {
		return nil, err
	}

	// Get workspace name from thread (if available)
	prefix := ""
	if ws := thread.Local("workspace_name"); ws != nil {
		if s, ok := ws.(string); ok {
			prefix = s
		}
	}

	rb := NewRunfilesBuilder(prefix)

	// Add files
	if filesList != nil {
		iter := filesList.Iterate()
		defer iter.Done()
		var v starlark.Value
		for iter.Next(&v) {
			f, ok := v.(*types.File)
			if !ok {
				return nil, fmt.Errorf("runfiles: files must contain File objects, got %s", v.Type())
			}
			rb.AddFile(f)
		}
	}

	// Add transitive files
	if transitiveFiles != nil {
		rb.AddTransitiveFiles(transitiveFiles)
	}

	// Add symlinks
	if symlinksDict != nil {
		for _, item := range symlinksDict.Items() {
			path, ok := item[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("runfiles: symlink keys must be strings")
			}
			target, ok := item[1].(*types.File)
			if !ok {
				return nil, fmt.Errorf("runfiles: symlink values must be File objects, got %s", item[1].Type())
			}
			rb.AddSymlink(string(path), target)
		}
	}

	// Add root symlinks
	if rootSymlinksArg != nil {
		for _, item := range rootSymlinksArg.Items() {
			path, ok := item[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("runfiles: root_symlinks keys must be strings")
			}
			target, ok := item[1].(*types.File)
			if !ok {
				return nil, fmt.Errorf("runfiles: root_symlinks values must be File objects, got %s", item[1].Type())
			}
			rb.AddRootSymlink(string(path), target)
		}
	}

	return rb.Build()
}
