// Package attr implements the Starlark "attr" module for defining rule attribute schemas.
//
// This module provides functions for defining attribute schemas used in rule() and aspect()
// definitions. Each function returns a Descriptor that describes the attribute's type,
// default value, and constraints.
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkAttrModuleApi.java
// Reference: bazel/src/main/java/com/google/devtools/build/lib/analysis/starlark/StarlarkAttrModule.java
package attr

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// Module returns the "attr" module for use in Starlark.
// Reference: StarlarkAttrModuleApi.java
func Module() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "attr",
		Members: starlark.StringDict{
			"string":                  starlark.NewBuiltin("attr.string", attrString),
			"int":                     starlark.NewBuiltin("attr.int", attrInt),
			"bool":                    starlark.NewBuiltin("attr.bool", attrBool),
			"label":                   starlark.NewBuiltin("attr.label", attrLabel),
			"label_list":              starlark.NewBuiltin("attr.label_list", attrLabelList),
			"string_list":             starlark.NewBuiltin("attr.string_list", attrStringList),
			"int_list":                starlark.NewBuiltin("attr.int_list", attrIntList),
			"string_dict":             starlark.NewBuiltin("attr.string_dict", attrStringDict),
			"string_list_dict":        starlark.NewBuiltin("attr.string_list_dict", attrStringListDict),
			"label_keyed_string_dict": starlark.NewBuiltin("attr.label_keyed_string_dict", attrLabelKeyedStringDict),
			"output":                  starlark.NewBuiltin("attr.output", attrOutput),
			"output_list":             starlark.NewBuiltin("attr.output_list", attrOutputList),
		},
	}
}

// attrString implements attr.string().
// Reference: StarlarkAttrModuleApi.java stringAttribute()
// Parameters from reference:
//   - configurable: unbound (for symbolic macros only)
//   - default: "" (empty string)
//   - doc: None
//   - mandatory: False
//   - values: [] (empty list of allowed values)
func attrString(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		defaultVal starlark.Value = starlark.String("")
		doc        starlark.Value = starlark.None
		mandatory  bool           = false
		values     *starlark.List = starlark.NewList(nil)
	)

	if err := starlark.UnpackArgs("attr.string", args, kwargs,
		"default?", &defaultVal,
		"doc?", &doc,
		"mandatory?", &mandatory,
		"values?", &values,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("string", TypeString)
	desc.SetDefault(defaultVal)
	desc.SetMandatory(mandatory)

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	// Handle values constraint
	// Reference: StarlarkAttrModule.java - VALUES_ARG handling
	if values != nil && values.Len() > 0 {
		allowedValues := make([]starlark.Value, values.Len())
		for i := range values.Len() {
			allowedValues[i] = values.Index(i)
		}
		desc.SetValues(allowedValues)
	}

	return desc, nil
}

// attrInt implements attr.int().
// Reference: StarlarkAttrModuleApi.java intAttribute()
// Parameters from reference:
//   - configurable: unbound
//   - default: 0
//   - doc: None
//   - mandatory: False
//   - values: [] (allowed integer values)
func attrInt(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		defaultVal starlark.Int   = starlark.MakeInt(0)
		doc        starlark.Value = starlark.None
		mandatory  bool           = false
		values     *starlark.List = starlark.NewList(nil)
	)

	if err := starlark.UnpackArgs("attr.int", args, kwargs,
		"default?", &defaultVal,
		"doc?", &doc,
		"mandatory?", &mandatory,
		"values?", &values,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("int", TypeInt)
	desc.SetDefault(defaultVal)
	desc.SetMandatory(mandatory)

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	if values != nil && values.Len() > 0 {
		allowedValues := make([]starlark.Value, values.Len())
		for i := range values.Len() {
			allowedValues[i] = values.Index(i)
		}
		desc.SetValues(allowedValues)
	}

	return desc, nil
}

// attrBool implements attr.bool().
// Reference: StarlarkAttrModuleApi.java boolAttribute()
// Parameters from reference:
//   - configurable: unbound
//   - default: False
//   - doc: None
//   - mandatory: False
func attrBool(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		defaultVal bool           = false
		doc        starlark.Value = starlark.None
		mandatory  bool           = false
	)

	if err := starlark.UnpackArgs("attr.bool", args, kwargs,
		"default?", &defaultVal,
		"doc?", &doc,
		"mandatory?", &mandatory,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("bool", TypeBool)
	desc.SetDefault(starlark.Bool(defaultVal))
	desc.SetMandatory(mandatory)

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	return desc, nil
}

// attrLabel implements attr.label().
// Reference: StarlarkAttrModuleApi.java labelAttribute()
// Parameters from reference:
//   - configurable: unbound
//   - default: None (can be Label, String, or function)
//   - doc: None
//   - executable: False
//   - allow_files: None (can be bool or list of extensions)
//   - allow_single_file: None
//   - mandatory: False
//   - providers: []
//   - allow_rules: None (deprecated)
//   - cfg: None (can be "target", "exec", or transition)
//   - aspects: []
func attrLabel(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		defaultVal      starlark.Value = starlark.None
		doc             starlark.Value = starlark.None
		executable      bool           = false
		allowFiles      starlark.Value = starlark.None
		allowSingleFile starlark.Value = starlark.None
		mandatory       bool           = false
		providers       *starlark.List = starlark.NewList(nil)
		allowRules      starlark.Value = starlark.None
		cfg             starlark.Value = starlark.None
		aspects         *starlark.List = starlark.NewList(nil)
	)

	if err := starlark.UnpackArgs("attr.label", args, kwargs,
		"default?", &defaultVal,
		"doc?", &doc,
		"executable?", &executable,
		"allow_files?", &allowFiles,
		"allow_single_file?", &allowSingleFile,
		"mandatory?", &mandatory,
		"providers?", &providers,
		"allow_rules?", &allowRules,
		"cfg?", &cfg,
		"aspects?", &aspects,
	); err != nil {
		return nil, err
	}

	// Reference: StarlarkAttrModule.java - cannot specify both allow_files and allow_single_file
	if allowFiles != starlark.None && allowSingleFile != starlark.None {
		return nil, fmt.Errorf("attr.label: Cannot specify both allow_files and allow_single_file")
	}

	// Reference: StarlarkAttrModule.java - cfg is required when executable=True
	if executable && cfg == starlark.None {
		return nil, fmt.Errorf("attr.label: cfg parameter is mandatory when executable=True is provided")
	}

	desc := NewDescriptor("label", TypeLabel)
	desc.SetDefault(defaultVal)
	desc.SetMandatory(mandatory)
	desc.SetExecutable(executable)

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	// Process allow_files
	// Reference: StarlarkAttrModule.java setAllowedFileTypes()
	if allowFiles != starlark.None {
		af, err := parseAllowFiles(allowFiles)
		if err != nil {
			return nil, fmt.Errorf("attr.label: %w", err)
		}
		desc.SetAllowFiles(af)
	}

	// Process allow_single_file
	if allowSingleFile != starlark.None {
		af, err := parseAllowFiles(allowSingleFile)
		if err != nil {
			return nil, fmt.Errorf("attr.label: %w", err)
		}
		desc.SetAllowSingleFile(af)
	}

	// Process providers
	// Reference: StarlarkAttrModule.java buildProviderPredicate()
	if providers != nil && providers.Len() > 0 {
		pr, err := parseProviders(providers)
		if err != nil {
			return nil, fmt.Errorf("attr.label: %w", err)
		}
		desc.SetProviders(pr)
	}

	// Process cfg
	if cfg != starlark.None {
		if s, ok := cfg.(starlark.String); ok {
			cfgStr := string(s)
			// Reference: StarlarkAttrModule.java convertCfg()
			if cfgStr != "target" && cfgStr != "exec" {
				return nil, fmt.Errorf("attr.label: cfg must be 'target', 'exec', or a transition, got %q", cfgStr)
			}
			desc.SetCfg(cfgStr)
		} else {
			// Could be a transition object - for now store string representation
			desc.SetCfg(cfg.String())
		}
	}

	// Process aspects
	if aspects != nil && aspects.Len() > 0 {
		aspectList := make([]starlark.Value, aspects.Len())
		for i := range aspects.Len() {
			aspectList[i] = aspects.Index(i)
		}
		desc.SetAspects(aspectList)
	}

	// Process allow_rules (deprecated)
	if allowRules != starlark.None {
		if list, ok := allowRules.(*starlark.List); ok {
			rules := make([]string, list.Len())
			for i := range list.Len() {
				if s, ok := list.Index(i).(starlark.String); ok {
					rules[i] = string(s)
				}
			}
			desc.SetAllowRules(rules)
		}
	}

	return desc, nil
}

// attrLabelList implements attr.label_list().
// Reference: StarlarkAttrModuleApi.java labelListAttribute()
// Parameters from reference:
//   - allow_empty: True
//   - configurable: unbound
//   - default: []
//   - doc: None
//   - allow_files: None
//   - allow_rules: None (deprecated)
//   - providers: []
//   - mandatory: False
//   - cfg: None
//   - aspects: []
func attrLabelList(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		allowEmpty starlark.Value = starlark.True
		defaultVal starlark.Value = starlark.NewList(nil)
		doc        starlark.Value = starlark.None
		allowFiles starlark.Value = starlark.None
		allowRules starlark.Value = starlark.None
		providers  *starlark.List = starlark.NewList(nil)
		mandatory  bool           = false
		cfg        starlark.Value = starlark.None
		aspects    *starlark.List = starlark.NewList(nil)
	)

	if err := starlark.UnpackArgs("attr.label_list", args, kwargs,
		"allow_empty?", &allowEmpty,
		"default?", &defaultVal,
		"doc?", &doc,
		"allow_files?", &allowFiles,
		"allow_rules?", &allowRules,
		"providers?", &providers,
		"mandatory?", &mandatory,
		"cfg?", &cfg,
		"aspects?", &aspects,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("label_list", TypeLabelList)
	desc.SetDefault(defaultVal)
	desc.SetMandatory(mandatory)

	// Reference: StarlarkAttrModule.java - ALLOW_EMPTY_ARG handling
	if allowEmpty == starlark.False {
		desc.SetAllowEmpty(false)
	}

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	if allowFiles != starlark.None {
		af, err := parseAllowFiles(allowFiles)
		if err != nil {
			return nil, fmt.Errorf("attr.label_list: %w", err)
		}
		desc.SetAllowFiles(af)
	}

	if providers != nil && providers.Len() > 0 {
		pr, err := parseProviders(providers)
		if err != nil {
			return nil, fmt.Errorf("attr.label_list: %w", err)
		}
		desc.SetProviders(pr)
	}

	if cfg != starlark.None {
		if s, ok := cfg.(starlark.String); ok {
			cfgStr := string(s)
			if cfgStr != "target" && cfgStr != "exec" {
				return nil, fmt.Errorf("attr.label_list: cfg must be 'target', 'exec', or a transition, got %q", cfgStr)
			}
			desc.SetCfg(cfgStr)
		} else {
			desc.SetCfg(cfg.String())
		}
	}

	if aspects != nil && aspects.Len() > 0 {
		aspectList := make([]starlark.Value, aspects.Len())
		for i := range aspects.Len() {
			aspectList[i] = aspects.Index(i)
		}
		desc.SetAspects(aspectList)
	}

	if allowRules != starlark.None {
		if list, ok := allowRules.(*starlark.List); ok {
			rules := make([]string, list.Len())
			for i := range list.Len() {
				if s, ok := list.Index(i).(starlark.String); ok {
					rules[i] = string(s)
				}
			}
			desc.SetAllowRules(rules)
		}
	}

	return desc, nil
}

// attrStringList implements attr.string_list().
// Reference: StarlarkAttrModuleApi.java stringListAttribute()
// Parameters from reference:
//   - mandatory: False
//   - allow_empty: True
//   - configurable: unbound
//   - default: []
//   - doc: None
func attrStringList(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		mandatory  bool           = false
		allowEmpty starlark.Value = starlark.True
		defaultVal starlark.Value = starlark.NewList(nil)
		doc        starlark.Value = starlark.None
	)

	if err := starlark.UnpackArgs("attr.string_list", args, kwargs,
		"mandatory?", &mandatory,
		"allow_empty?", &allowEmpty,
		"default?", &defaultVal,
		"doc?", &doc,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("string_list", TypeStringList)
	desc.SetDefault(defaultVal)
	desc.SetMandatory(mandatory)

	if allowEmpty == starlark.False {
		desc.SetAllowEmpty(false)
	}

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	return desc, nil
}

// attrIntList implements attr.int_list().
// Reference: StarlarkAttrModuleApi.java intListAttribute()
// Parameters from reference:
//   - mandatory: False
//   - allow_empty: True
//   - configurable: unbound
//   - default: []
//   - doc: None
func attrIntList(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		mandatory  bool           = false
		allowEmpty starlark.Value = starlark.True
		defaultVal starlark.Value = starlark.NewList(nil)
		doc        starlark.Value = starlark.None
	)

	if err := starlark.UnpackArgs("attr.int_list", args, kwargs,
		"mandatory?", &mandatory,
		"allow_empty?", &allowEmpty,
		"default?", &defaultVal,
		"doc?", &doc,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("int_list", TypeIntList)
	desc.SetDefault(defaultVal)
	desc.SetMandatory(mandatory)

	if allowEmpty == starlark.False {
		desc.SetAllowEmpty(false)
	}

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	return desc, nil
}

// attrStringDict implements attr.string_dict().
// Reference: StarlarkAttrModuleApi.java stringDictAttribute()
// Parameters from reference:
//   - allow_empty: True
//   - configurable: unbound
//   - default: {}
//   - doc: None
//   - mandatory: False
func attrStringDict(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		allowEmpty starlark.Value = starlark.True
		defaultVal starlark.Value = starlark.NewDict(0)
		doc        starlark.Value = starlark.None
		mandatory  bool           = false
	)

	if err := starlark.UnpackArgs("attr.string_dict", args, kwargs,
		"allow_empty?", &allowEmpty,
		"default?", &defaultVal,
		"doc?", &doc,
		"mandatory?", &mandatory,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("string_dict", TypeStringDict)
	desc.SetDefault(defaultVal)
	desc.SetMandatory(mandatory)

	if allowEmpty == starlark.False {
		desc.SetAllowEmpty(false)
	}

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	return desc, nil
}

// attrStringListDict implements attr.string_list_dict().
// Reference: StarlarkAttrModuleApi.java stringListDictAttribute()
// Parameters from reference:
//   - allow_empty: True
//   - configurable: unbound
//   - default: {}
//   - doc: None
//   - mandatory: False
func attrStringListDict(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		allowEmpty starlark.Value = starlark.True
		defaultVal starlark.Value = starlark.NewDict(0)
		doc        starlark.Value = starlark.None
		mandatory  bool           = false
	)

	if err := starlark.UnpackArgs("attr.string_list_dict", args, kwargs,
		"allow_empty?", &allowEmpty,
		"default?", &defaultVal,
		"doc?", &doc,
		"mandatory?", &mandatory,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("string_list_dict", TypeStringListDict)
	desc.SetDefault(defaultVal)
	desc.SetMandatory(mandatory)

	if allowEmpty == starlark.False {
		desc.SetAllowEmpty(false)
	}

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	return desc, nil
}

// attrLabelKeyedStringDict implements attr.label_keyed_string_dict().
// Reference: StarlarkAttrModuleApi.java labelKeyedStringDictAttribute()
// Parameters from reference:
//   - allow_empty: True
//   - configurable: unbound
//   - default: {}
//   - doc: None
//   - allow_files: None
//   - allow_rules: None (deprecated)
//   - providers: []
//   - mandatory: False
//   - cfg: None
//   - aspects: []
func attrLabelKeyedStringDict(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		allowEmpty starlark.Value = starlark.True
		defaultVal starlark.Value = starlark.NewDict(0)
		doc        starlark.Value = starlark.None
		allowFiles starlark.Value = starlark.None
		allowRules starlark.Value = starlark.None
		providers  *starlark.List = starlark.NewList(nil)
		mandatory  bool           = false
		cfg        starlark.Value = starlark.None
		aspects    *starlark.List = starlark.NewList(nil)
	)

	if err := starlark.UnpackArgs("attr.label_keyed_string_dict", args, kwargs,
		"allow_empty?", &allowEmpty,
		"default?", &defaultVal,
		"doc?", &doc,
		"allow_files?", &allowFiles,
		"allow_rules?", &allowRules,
		"providers?", &providers,
		"mandatory?", &mandatory,
		"cfg?", &cfg,
		"aspects?", &aspects,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("label_keyed_string_dict", TypeLabelKeyedStringDict)
	desc.SetDefault(defaultVal)
	desc.SetMandatory(mandatory)

	if allowEmpty == starlark.False {
		desc.SetAllowEmpty(false)
	}

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	if allowFiles != starlark.None {
		af, err := parseAllowFiles(allowFiles)
		if err != nil {
			return nil, fmt.Errorf("attr.label_keyed_string_dict: %w", err)
		}
		desc.SetAllowFiles(af)
	}

	if providers != nil && providers.Len() > 0 {
		pr, err := parseProviders(providers)
		if err != nil {
			return nil, fmt.Errorf("attr.label_keyed_string_dict: %w", err)
		}
		desc.SetProviders(pr)
	}

	if cfg != starlark.None {
		if s, ok := cfg.(starlark.String); ok {
			cfgStr := string(s)
			if cfgStr != "target" && cfgStr != "exec" {
				return nil, fmt.Errorf("attr.label_keyed_string_dict: cfg must be 'target', 'exec', or a transition, got %q", cfgStr)
			}
			desc.SetCfg(cfgStr)
		} else {
			desc.SetCfg(cfg.String())
		}
	}

	if aspects != nil && aspects.Len() > 0 {
		aspectList := make([]starlark.Value, aspects.Len())
		for i := range aspects.Len() {
			aspectList[i] = aspects.Index(i)
		}
		desc.SetAspects(aspectList)
	}

	if allowRules != starlark.None {
		if list, ok := allowRules.(*starlark.List); ok {
			rules := make([]string, list.Len())
			for i := range list.Len() {
				if s, ok := list.Index(i).(starlark.String); ok {
					rules[i] = string(s)
				}
			}
			desc.SetAllowRules(rules)
		}
	}

	return desc, nil
}

// attrOutput implements attr.output().
// Reference: StarlarkAttrModuleApi.java outputAttribute()
// Parameters from reference:
//   - doc: None
//   - mandatory: False
//
// Note: Output attributes are nonconfigurable
// Reference: StarlarkAttrModule.java createNonconfigurableAttrDescriptor() for output
func attrOutput(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		doc       starlark.Value = starlark.None
		mandatory bool           = false
	)

	if err := starlark.UnpackArgs("attr.output", args, kwargs,
		"doc?", &doc,
		"mandatory?", &mandatory,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("output", TypeOutput)
	desc.SetMandatory(mandatory)

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	return desc, nil
}

// attrOutputList implements attr.output_list().
// Reference: StarlarkAttrModuleApi.java outputListAttribute()
// Parameters from reference:
//   - allow_empty: True
//   - doc: None
//   - mandatory: False
func attrOutputList(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		allowEmpty starlark.Value = starlark.True
		doc        starlark.Value = starlark.None
		mandatory  bool           = false
	)

	if err := starlark.UnpackArgs("attr.output_list", args, kwargs,
		"allow_empty?", &allowEmpty,
		"doc?", &doc,
		"mandatory?", &mandatory,
	); err != nil {
		return nil, err
	}

	desc := NewDescriptor("output_list", TypeOutputList)
	desc.SetDefault(starlark.NewList(nil))
	desc.SetMandatory(mandatory)

	if allowEmpty == starlark.False {
		desc.SetAllowEmpty(false)
	}

	if doc != starlark.None {
		if s, ok := doc.(starlark.String); ok {
			desc.SetDoc(string(s))
		}
	}

	return desc, nil
}

// parseAllowFiles parses the allow_files parameter.
// Reference: StarlarkAttrModule.java setAllowedFileTypes()
// Can be:
// - True: allow any file
// - False: allow no files
// - list of strings: allow files with these extensions
func parseAllowFiles(v starlark.Value) (*AllowFilesValue, error) {
	switch x := v.(type) {
	case starlark.Bool:
		if x {
			return NewAllowFilesAll(), nil
		}
		return NewAllowFilesNone(), nil
	case *starlark.List:
		extensions := make([]string, x.Len())
		for i := range x.Len() {
			s, ok := x.Index(i).(starlark.String)
			if !ok {
				return nil, fmt.Errorf("allow_files element must be a string, got %s", x.Index(i).Type())
			}
			extensions[i] = string(s)
		}
		return NewAllowFilesExtensions(extensions), nil
	case starlark.Tuple:
		extensions := make([]string, len(x))
		for i := range len(x) {
			s, ok := x[i].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("allow_files element must be a string, got %s", x[i].Type())
			}
			extensions[i] = string(s)
		}
		return NewAllowFilesExtensions(extensions), nil
	default:
		return nil, fmt.Errorf("allow_files must be a boolean or a list of strings, got %s", v.Type())
	}
}

// parseProviders parses the providers parameter.
// Reference: StarlarkAttrModule.java buildProviderPredicate()
// The providers parameter can be:
// - A list of providers: [P1, P2] means requires ALL of P1 and P2
// - A list of lists: [[P1, P2], [P3]] means (requires P1 AND P2) OR (requires P3)
func parseProviders(list *starlark.List) (*ProviderRequirement, error) {
	if list.Len() == 0 {
		return nil, nil
	}

	// Check if it's a list of providers or list of lists
	first := list.Index(0)
	if _, isProvider := first.(*types.Provider); isProvider {
		// It's a simple list of providers - wrap in single alternative
		providers := make([]*types.Provider, list.Len())
		for i := range list.Len() {
			p, ok := list.Index(i).(*types.Provider)
			if !ok {
				return nil, fmt.Errorf("providers must be a list of providers or list of lists of providers, got %s at index %d", list.Index(i).Type(), i)
			}
			providers[i] = p
		}
		return NewProviderRequirement([][]*types.Provider{providers}), nil
	}

	// It's a list of lists
	alternatives := make([][]*types.Provider, list.Len())
	for i := range list.Len() {
		inner, ok := list.Index(i).(*starlark.List)
		if !ok {
			return nil, fmt.Errorf("providers must be a list of providers or list of lists of providers, got %s at index %d", list.Index(i).Type(), i)
		}
		providers := make([]*types.Provider, inner.Len())
		for j := range inner.Len() {
			p, ok := inner.Index(j).(*types.Provider)
			if !ok {
				return nil, fmt.Errorf("providers inner list must contain providers, got %s at [%d][%d]", inner.Index(j).Type(), i, j)
			}
			providers[j] = p
		}
		alternatives[i] = providers
	}

	return NewProviderRequirement(alternatives), nil
}
