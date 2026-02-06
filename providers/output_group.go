// Package providers implements Bazel's built-in providers.
//
// OutputGroupInfo implementation based on:
// - bazel/src/main/java/com/google/devtools/build/lib/analysis/OutputGroupInfo.java
// - bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/OutputGroupInfoApi.java
package providers

import (
	"fmt"
	"sort"

	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
)

// OutputGroupInfo constants from OutputGroupInfo.java
const (
	// HiddenOutputGroupPrefix marks output groups not reported to the user.
	// Reference: OutputGroupInfo.java HIDDEN_OUTPUT_GROUP_PREFIX
	HiddenOutputGroupPrefix = "_"

	// InternalSuffix marks output groups internal to Bazel.
	// Reference: OutputGroupInfo.java INTERNAL_SUFFIX
	InternalSuffix = "_INTERNAL_"

	// FilesToCompile is the output group for compilation outputs.
	// Reference: OutputGroupInfo.java FILES_TO_COMPILE
	FilesToCompile = "compilation_outputs"

	// CompilationPrerequisites is for compilation prerequisites.
	// Reference: OutputGroupInfo.java COMPILATION_PREREQUISITES
	CompilationPrerequisites = "compilation_prerequisites" + InternalSuffix

	// HiddenTopLevel is for files built but not reported.
	// Reference: OutputGroupInfo.java HIDDEN_TOP_LEVEL
	HiddenTopLevel = HiddenOutputGroupPrefix + "hidden_top_level" + InternalSuffix

	// Validation is for validation action outputs.
	// Reference: OutputGroupInfo.java VALIDATION
	Validation = HiddenOutputGroupPrefix + "validation"

	// ValidationTopLevel is for validation from top-level aspects.
	// Reference: OutputGroupInfo.java VALIDATION_TOP_LEVEL
	ValidationTopLevel = HiddenOutputGroupPrefix + "validation_top_level" + InternalSuffix

	// ValidationTransitive is for overriding validation outputs.
	// Reference: OutputGroupInfo.java VALIDATION_TRANSITIVE
	ValidationTransitive = HiddenOutputGroupPrefix + "validation_transitive"

	// TempFiles is for temporary files (e.g., .i, .d, .s files).
	// Reference: OutputGroupInfo.java TEMP_FILES
	TempFiles = "temp_files" + InternalSuffix

	// Default is the default output group.
	// Reference: OutputGroupInfo.java DEFAULT
	Default = "default"
)

// DefaultOutputGroups is the default set of output groups to build.
// Reference: OutputGroupInfo.java DEFAULT_GROUPS
var DefaultOutputGroups = []string{Default, TempFiles, HiddenTopLevel}

// OutputGroupInfoProvider is the singleton provider type for OutputGroupInfo.
// Reference: OutputGroupInfo.java STARLARK_CONSTRUCTOR
var OutputGroupInfoProvider = types.NewProvider("OutputGroupInfo", nil,
	"A provider that indicates what output groups a rule has.", nil)

// OutputGroupInfo provides artifacts that can be built when the target is
// mentioned on the command line.
//
// Reference: OutputGroupInfo.java
//
// The artifacts are grouped into "output groups". Which output groups are built
// is controlled by the --output_groups command line option.
//
// Output groups starting with an underscore are "not important" - artifacts built
// because such an output group is mentioned are not reported on the output.
type OutputGroupInfo struct {
	// groups maps output group names to depsets of files
	// Reference: OutputGroupInfo.java - uses ImmutableSharedKeyMap<String, NestedSet<Artifact>>
	groups map[string]*types.Depset

	frozen bool
}

var (
	_ starlark.Value     = (*OutputGroupInfo)(nil)
	_ starlark.HasAttrs  = (*OutputGroupInfo)(nil)
	_ starlark.Mapping   = (*OutputGroupInfo)(nil)
	_ starlark.Iterable  = (*OutputGroupInfo)(nil)
	_ starlark.Indexable = (*OutputGroupInfo)(nil)
)

// NewOutputGroupInfo creates a new OutputGroupInfo.
func NewOutputGroupInfo() *OutputGroupInfo {
	return &OutputGroupInfo{
		groups: make(map[string]*types.Depset),
	}
}

// NewOutputGroupInfoWithGroups creates an OutputGroupInfo with the given groups.
func NewOutputGroupInfoWithGroups(groups map[string]*types.Depset) *OutputGroupInfo {
	og := NewOutputGroupInfo()
	for k, v := range groups {
		og.groups[k] = v
	}
	return og
}

// String returns the Starlark representation.
func (o *OutputGroupInfo) String() string {
	return fmt.Sprintf("OutputGroupInfo(%v)", o.groupNames())
}

// Type returns "OutputGroupInfo".
func (o *OutputGroupInfo) Type() string { return "OutputGroupInfo" }

// Freeze marks the info as frozen.
func (o *OutputGroupInfo) Freeze() {
	if o.frozen {
		return
	}
	o.frozen = true
	for _, v := range o.groups {
		v.Freeze()
	}
}

// Truth returns true if there are any groups.
func (o *OutputGroupInfo) Truth() starlark.Bool {
	return starlark.Bool(len(o.groups) > 0)
}

// Hash returns an error (OutputGroupInfo is immutable but not hashable by identity).
// Reference: OutputGroupInfo.java isImmutable() returns true
func (o *OutputGroupInfo) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: OutputGroupInfo")
}

// Attr returns an attribute of the OutputGroupInfo.
// Reference: OutputGroupInfo.java getValue() method
func (o *OutputGroupInfo) Attr(name string) (starlark.Value, error) {
	if ds, ok := o.groups[name]; ok {
		return ds, nil
	}
	return nil, starlark.NoSuchAttrError(fmt.Sprintf("OutputGroupInfo has no output group %q", name))
}

// AttrNames returns the list of attribute names (output group names).
// Reference: OutputGroupInfo.java getFieldNames() method
func (o *OutputGroupInfo) AttrNames() []string {
	return o.groupNames()
}

// groupNames returns sorted group names.
func (o *OutputGroupInfo) groupNames() []string {
	names := make([]string, 0, len(o.groups))
	for k := range o.groups {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Get implements starlark.Mapping (dict-like access).
// Reference: OutputGroupInfo.java getValue() method
func (o *OutputGroupInfo) Get(key starlark.Value) (v starlark.Value, found bool, err error) {
	name, ok := key.(starlark.String)
	if !ok {
		return nil, false, fmt.Errorf("OutputGroupInfo: key must be string, got %s", key.Type())
	}
	if ds, ok := o.groups[string(name)]; ok {
		return ds, true, nil
	}
	return starlark.None, false, nil
}

// Index implements starlark.Indexable (bracket access).
// Reference: OutputGroupInfo.java getIndex() method
func (o *OutputGroupInfo) Index(i int) starlark.Value {
	names := o.groupNames()
	if i < 0 || i >= len(names) {
		return nil
	}
	return starlark.String(names[i])
}

// Len returns the number of output groups.
func (o *OutputGroupInfo) Len() int {
	return len(o.groups)
}

// Iterate implements starlark.Iterable (iterates over group names).
// Reference: OutputGroupInfo.java iterator() method
func (o *OutputGroupInfo) Iterate() starlark.Iterator {
	names := o.groupNames()
	return &outputGroupIterator{names: names}
}

type outputGroupIterator struct {
	names []string
	index int
}

func (it *outputGroupIterator) Next(p *starlark.Value) bool {
	if it.index >= len(it.names) {
		return false
	}
	*p = starlark.String(it.names[it.index])
	it.index++
	return true
}

func (it *outputGroupIterator) Done() {}

// GetOutputGroup returns the artifacts in a particular output group.
// Reference: OutputGroupInfo.java getOutputGroup() method
// The return value is never nil. If the group is not present, returns empty depset.
func (o *OutputGroupInfo) GetOutputGroup(name string) *types.Depset {
	if ds, ok := o.groups[name]; ok {
		return ds
	}
	d, _ := types.NewDepset(types.OrderDefault, nil, nil)
	return d
}

// SetOutputGroup sets an output group.
func (o *OutputGroupInfo) SetOutputGroup(name string, files *types.Depset) {
	if o.frozen {
		return
	}
	o.groups[name] = files
}

// ContainsKey returns true if the output group exists.
// Reference: OutputGroupInfo.java containsKey() method
func (o *OutputGroupInfo) ContainsKey(name string) bool {
	_, ok := o.groups[name]
	return ok
}

// Groups returns all output groups.
func (o *OutputGroupInfo) Groups() map[string]*types.Depset {
	return o.groups
}

// Provider returns the OutputGroupInfo provider.
func (o *OutputGroupInfo) Provider() *types.Provider {
	return OutputGroupInfoProvider
}

// OutputGroupInfoBuiltin is the Starlark constructor for OutputGroupInfo.
// Reference: OutputGroupInfo.java OutputGroupInfoProvider.constructor() method
//
// Takes **kwargs where each key is a group name and values are depsets of files.
//
// Example:
//
//	OutputGroupInfo(
//	    validation = depset([...]),
//	    _hidden_top_level = depset([...]),
//	)
func OutputGroupInfoBuiltin(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// OutputGroupInfo only accepts keyword arguments
	if len(args) > 0 {
		return nil, fmt.Errorf("OutputGroupInfo: unexpected positional arguments")
	}

	info := NewOutputGroupInfo()

	// Process kwargs - each key is an output group name
	// Reference: OutputGroupInfo.java constructor() - iterates over kwargs.entrySet()
	for _, kv := range kwargs {
		groupName := string(kv[0].(starlark.String))
		value := kv[1]

		// Convert value to depset
		// Reference: StarlarkRuleConfiguredTargetUtil.convertToOutputGroupValue()
		var depset *types.Depset

		switch v := value.(type) {
		case *types.Depset:
			depset = v
		case *starlark.List:
			// Convert list to depset
			items := make([]starlark.Value, v.Len())
			for i := range v.Len() {
				item := v.Index(i)
				// Verify it's a File
				if _, ok := item.(*types.File); !ok {
					return nil, fmt.Errorf("OutputGroupInfo: output group %q contains non-File: %s",
						groupName, item.Type())
				}
				items[i] = item
			}
			var err error
			depset, err = types.NewDepset(types.OrderDefault, items, nil)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("OutputGroupInfo: output group %q must be a depset or list of Files, got %s",
				groupName, value.Type())
		}

		info.groups[groupName] = depset
	}

	return info, nil
}

// SingleGroup creates an OutputGroupInfo with a single output group.
// Reference: OutputGroupInfo.java singleGroup() method
func SingleGroup(group string, files *types.Depset) *OutputGroupInfo {
	return NewOutputGroupInfoWithGroups(map[string]*types.Depset{
		group: files,
	})
}

// MergeOutputGroupInfo merges multiple OutputGroupInfo providers.
// Reference: OutputGroupInfo.java merge() method
//
// The set of output groups must be disjoint, except for the validation output group,
// which is always merged.
func MergeOutputGroupInfo(providers []*OutputGroupInfo) (*OutputGroupInfo, error) {
	if len(providers) == 0 {
		return nil, nil
	}
	if len(providers) == 1 {
		return providers[0], nil
	}

	// Build merged groups
	// Reference: OutputGroupInfo.java merge() uses TreeMap for sorted iteration
	builders := make(map[string]*types.Depset)

	for _, provider := range providers {
		for group, files := range provider.groups {
			if existing, ok := builders[group]; ok {
				// Merge the depsets
				merged, err := types.NewDepset(types.OrderDefault, nil, []*types.Depset{existing, files})
				if err != nil {
					return nil, err
				}
				builders[group] = merged
			} else {
				builders[group] = files
			}
		}
	}

	return NewOutputGroupInfoWithGroups(builders), nil
}

// IsHiddenOutputGroup returns true if the output group name starts with underscore.
// Reference: OutputGroupInfo.java HIDDEN_OUTPUT_GROUP_PREFIX usage
func IsHiddenOutputGroup(name string) bool {
	return len(name) > 0 && name[0] == '_'
}

// IsInternalOutputGroup returns true if the output group name ends with _INTERNAL_.
// Reference: OutputGroupInfo.java INTERNAL_SUFFIX usage
func IsInternalOutputGroup(name string) bool {
	return len(name) >= len(InternalSuffix) &&
		name[len(name)-len(InternalSuffix):] == InternalSuffix
}
