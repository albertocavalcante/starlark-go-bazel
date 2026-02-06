// Package attr implements the Starlark "attr" module for defining rule attribute schemas.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/analysis/starlark/StarlarkAttrModule.java
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/Attribute.java
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/Type.java
package attr

import (
	"fmt"

	"go.starlark.net/starlark"

	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// Type represents the type of an attribute.
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/Type.java
type Type int

const (
	TypeString Type = iota
	TypeInt
	TypeBool
	TypeLabel
	TypeLabelList
	TypeStringList
	TypeIntList
	TypeStringDict
	TypeStringListDict
	TypeLabelKeyedStringDict
	TypeOutput
	TypeOutputList
)

// String returns a string representation of the type.
func (t Type) String() string {
	switch t {
	case TypeString:
		return "string"
	case TypeInt:
		return "int"
	case TypeBool:
		return "bool"
	case TypeLabel:
		return "label"
	case TypeLabelList:
		return "label_list"
	case TypeStringList:
		return "string_list"
	case TypeIntList:
		return "int_list"
	case TypeStringDict:
		return "string_dict"
	case TypeStringListDict:
		return "string_list_dict"
	case TypeLabelKeyedStringDict:
		return "label_keyed_string_dict"
	case TypeOutput:
		return "output"
	case TypeOutputList:
		return "output_list"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

// DefaultValue returns the default value for the type if no explicit default is provided.
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/Type.java - getDefaultValue()
func (t Type) DefaultValue() starlark.Value {
	switch t {
	case TypeString:
		return starlark.String("")
	case TypeInt:
		return starlark.MakeInt(0)
	case TypeBool:
		return starlark.False
	case TypeLabel:
		return starlark.None
	case TypeLabelList:
		return starlark.NewList(nil)
	case TypeStringList:
		return starlark.NewList(nil)
	case TypeIntList:
		return starlark.NewList(nil)
	case TypeStringDict:
		return starlark.NewDict(0)
	case TypeStringListDict:
		return starlark.NewDict(0)
	case TypeLabelKeyedStringDict:
		return starlark.NewDict(0)
	case TypeOutput:
		return starlark.None
	case TypeOutputList:
		return starlark.NewList(nil)
	default:
		return starlark.None
	}
}

// AllowFilesValue represents the allow_files parameter which can be:
// - bool (True = any file, False = no files)
// - list of strings (file extensions like [".cc", ".cpp"])
// Reference: StarlarkAttrModule.java setAllowedFileTypes()
type AllowFilesValue struct {
	allowAll    bool     // True means any file allowed (allow_files=True)
	allowNone   bool     // True means no files allowed (allow_files=False)
	extensions  []string // List of allowed extensions (e.g., [".cc", ".cpp"])
}

// NewAllowFilesAll creates an AllowFilesValue that allows all files.
func NewAllowFilesAll() *AllowFilesValue {
	return &AllowFilesValue{allowAll: true}
}

// NewAllowFilesNone creates an AllowFilesValue that allows no files.
func NewAllowFilesNone() *AllowFilesValue {
	return &AllowFilesValue{allowNone: true}
}

// NewAllowFilesExtensions creates an AllowFilesValue that allows specific extensions.
func NewAllowFilesExtensions(extensions []string) *AllowFilesValue {
	return &AllowFilesValue{extensions: extensions}
}

// AllowAll returns true if all files are allowed.
func (a *AllowFilesValue) AllowAll() bool { return a.allowAll }

// AllowNone returns true if no files are allowed.
func (a *AllowFilesValue) AllowNone() bool { return a.allowNone }

// Extensions returns the list of allowed file extensions.
func (a *AllowFilesValue) Extensions() []string { return a.extensions }

// ProviderRequirement represents a set of required providers.
// Reference: StarlarkAttrModule.java buildProviderPredicate()
// The providers parameter can be:
// - A list of providers (requires ALL in the list)
// - A list of lists of providers (requires ALL in at least ONE of the inner lists)
type ProviderRequirement struct {
	// alternatives is a list of provider sets.
	// A dependency satisfies the requirement if it provides ALL providers in at least one set.
	alternatives [][]*types.Provider
}

// NewProviderRequirement creates a new ProviderRequirement from alternatives.
func NewProviderRequirement(alternatives [][]*types.Provider) *ProviderRequirement {
	return &ProviderRequirement{alternatives: alternatives}
}

// Alternatives returns the list of provider set alternatives.
func (p *ProviderRequirement) Alternatives() [][]*types.Provider { return p.alternatives }

// Descriptor represents an attribute schema/descriptor returned by attr.* functions.
// Reference: StarlarkAttrModule.java Descriptor class
// This is returned by functions like attr.string(), attr.label(), etc.
// and used when defining rules via rule(attrs={...}).
type Descriptor struct {
	name string // e.g., "string", "label", "label_list"

	typ        Type           // The attribute type
	defaultVal starlark.Value // Default value (can be None for mandatory attrs)
	doc        string         // Documentation string
	mandatory  bool           // Whether the attribute must be specified

	// For string/int with restricted values
	// Reference: Attribute.java AllowedValueSet
	values []starlark.Value

	// For label/label_list attributes
	// Reference: StarlarkAttrModule.java - PROVIDERS_ARG handling
	providers *ProviderRequirement

	// Reference: StarlarkAttrModule.java - ALLOW_FILES_ARG handling
	allowFiles *AllowFilesValue

	// Reference: StarlarkAttrModule.java - ALLOW_SINGLE_FILE_ARG
	allowSingleFile *AllowFilesValue

	// Reference: StarlarkAttrModule.java - ALLOW_EMPTY_ARG
	allowEmpty bool

	// Reference: StarlarkAttrModule.java - CONFIGURATION_ARG
	// Valid values: "target", "exec", or a transition object
	cfg string

	// Reference: StarlarkAttrModule.java - EXECUTABLE_ARG
	executable bool

	// Reference: StarlarkAttrModule.java - ASPECTS_ARG
	aspects []starlark.Value

	// Reference: StarlarkAttrModule.java - ALLOW_RULES_ARG (deprecated)
	allowRules []string

	frozen bool
}

// Compile-time interface checks
var (
	_ starlark.Value    = (*Descriptor)(nil)
	_ starlark.HasAttrs = (*Descriptor)(nil)
)

// NewDescriptor creates a new attribute descriptor.
func NewDescriptor(name string, typ Type) *Descriptor {
	return &Descriptor{
		name:       name,
		typ:        typ,
		defaultVal: typ.DefaultValue(),
		allowEmpty: true, // Default per Bazel reference
	}
}

// String returns the Starlark representation.
// Reference: StarlarkAttrModule.java Descriptor.repr()
func (d *Descriptor) String() string {
	return fmt.Sprintf("<attr.%s>", d.name)
}

// Type returns "Attribute" as the Starlark type name.
// Reference: StarlarkAttrModuleApi.java - @StarlarkBuiltin name = "Attribute"
func (d *Descriptor) Type() string { return "Attribute" }

// Freeze marks the descriptor as frozen.
func (d *Descriptor) Freeze() {
	if d.frozen {
		return
	}
	d.frozen = true
	if d.defaultVal != nil {
		d.defaultVal.Freeze()
	}
	for _, v := range d.values {
		v.Freeze()
	}
	for _, a := range d.aspects {
		a.Freeze()
	}
}

// Truth returns true.
func (d *Descriptor) Truth() starlark.Bool { return true }

// Hash returns an error (descriptors are not hashable).
func (d *Descriptor) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: Attribute")
}

// Attr returns an attribute of the descriptor.
func (d *Descriptor) Attr(name string) (starlark.Value, error) {
	switch name {
	case "default":
		if d.defaultVal != nil {
			return d.defaultVal, nil
		}
		return starlark.None, nil
	case "mandatory":
		return starlark.Bool(d.mandatory), nil
	case "doc":
		if d.doc != "" {
			return starlark.String(d.doc), nil
		}
		return starlark.None, nil
	default:
		return nil, starlark.NoSuchAttrError(fmt.Sprintf("Attribute has no attribute %q", name))
	}
}

// AttrNames returns the list of attribute names.
func (d *Descriptor) AttrNames() []string {
	return []string{"default", "mandatory", "doc"}
}

// Getters for the descriptor fields

// Name returns the descriptor name (e.g., "string", "label").
func (d *Descriptor) Name() string { return d.name }

// AttrType returns the attribute type.
func (d *Descriptor) AttrType() Type { return d.typ }

// Default returns the default value.
func (d *Descriptor) Default() starlark.Value { return d.defaultVal }

// Doc returns the documentation string.
func (d *Descriptor) Doc() string { return d.doc }

// Mandatory returns whether the attribute is mandatory.
func (d *Descriptor) Mandatory() bool { return d.mandatory }

// Values returns the list of allowed values (for string/int with values param).
func (d *Descriptor) Values() []starlark.Value { return d.values }

// Providers returns the provider requirement.
func (d *Descriptor) Providers() *ProviderRequirement { return d.providers }

// AllowFiles returns the allow_files setting.
func (d *Descriptor) AllowFiles() *AllowFilesValue { return d.allowFiles }

// AllowSingleFile returns the allow_single_file setting.
func (d *Descriptor) AllowSingleFile() *AllowFilesValue { return d.allowSingleFile }

// AllowEmpty returns whether empty values are allowed (for list/dict types).
func (d *Descriptor) AllowEmpty() bool { return d.allowEmpty }

// Cfg returns the configuration setting.
func (d *Descriptor) Cfg() string { return d.cfg }

// Executable returns whether the attribute is executable.
func (d *Descriptor) Executable() bool { return d.executable }

// Aspects returns the list of aspects.
func (d *Descriptor) Aspects() []starlark.Value { return d.aspects }

// AllowRules returns the list of allowed rule classes (deprecated).
func (d *Descriptor) AllowRules() []string { return d.allowRules }

// Setters for building descriptors

// SetDefault sets the default value.
func (d *Descriptor) SetDefault(v starlark.Value) { d.defaultVal = v }

// SetDoc sets the documentation string.
func (d *Descriptor) SetDoc(doc string) { d.doc = doc }

// SetMandatory sets whether the attribute is mandatory.
func (d *Descriptor) SetMandatory(m bool) { d.mandatory = m }

// SetValues sets the list of allowed values.
func (d *Descriptor) SetValues(v []starlark.Value) { d.values = v }

// SetProviders sets the provider requirement.
func (d *Descriptor) SetProviders(p *ProviderRequirement) { d.providers = p }

// SetAllowFiles sets the allow_files setting.
func (d *Descriptor) SetAllowFiles(a *AllowFilesValue) { d.allowFiles = a }

// SetAllowSingleFile sets the allow_single_file setting.
func (d *Descriptor) SetAllowSingleFile(a *AllowFilesValue) { d.allowSingleFile = a }

// SetAllowEmpty sets whether empty values are allowed.
func (d *Descriptor) SetAllowEmpty(a bool) { d.allowEmpty = a }

// SetCfg sets the configuration.
func (d *Descriptor) SetCfg(cfg string) { d.cfg = cfg }

// SetExecutable sets whether the attribute is executable.
func (d *Descriptor) SetExecutable(e bool) { d.executable = e }

// SetAspects sets the list of aspects.
func (d *Descriptor) SetAspects(a []starlark.Value) { d.aspects = a }

// SetAllowRules sets the list of allowed rule classes.
func (d *Descriptor) SetAllowRules(r []string) { d.allowRules = r }
