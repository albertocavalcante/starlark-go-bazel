// Package analysis provides introspection and analysis utilities for Bazel Starlark.
package analysis

import (
	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
)

// RuleInfo contains introspection data about a rule.
type RuleInfo struct {
	Name       string               `json:"name"`
	Attrs      map[string]*AttrInfo `json:"attrs"`
	Provides   []string             `json:"provides"`
	Executable bool                 `json:"executable"`
	Test       bool                 `json:"test"`
	Doc        string               `json:"doc,omitempty"`
}

// AttrInfo contains introspection data about an attribute.
type AttrInfo struct {
	Type      string      `json:"type"`
	Mandatory bool        `json:"mandatory"`
	Default   any         `json:"default,omitempty"`
	Doc       string      `json:"doc,omitempty"`
}

// ProviderInfo contains introspection data about a provider.
type ProviderInfo struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
	Doc    string   `json:"doc,omitempty"`
}

// IntrospectRule returns info about a RuleClass.
func IntrospectRule(rc *types.RuleClass) *RuleInfo {
	info := &RuleInfo{
		Name:       rc.Name(),
		Attrs:      make(map[string]*AttrInfo),
		Executable: rc.IsExecutable(),
		Test:       rc.IsTest(),
		Doc:        rc.Doc(),
	}

	for name, attr := range rc.Attrs() {
		info.Attrs[name] = IntrospectAttr(attr)
	}

	// Extract provider names
	for _, p := range rc.Provides() {
		info.Provides = append(info.Provides, p.Name())
	}

	return info
}

// IntrospectAttr returns info about an AttrDescriptor.
func IntrospectAttr(attr *types.AttrDescriptor) *AttrInfo {
	info := &AttrInfo{
		Type:      string(attr.Type),
		Mandatory: attr.Mandatory,
		Doc:       attr.Doc,
	}

	// Convert default value to Go type for JSON serialization
	if attr.Default != nil {
		info.Default = starlarkToGo(attr.Default)
	}

	return info
}

// IntrospectProvider returns info about a Provider.
func IntrospectProvider(p *types.Provider) *ProviderInfo {
	return &ProviderInfo{
		Name:   p.Name(),
		Fields: p.Fields(),
		Doc:    p.Doc(),
	}
}

// starlarkToGo converts a Starlark value to a Go value for JSON serialization.
func starlarkToGo(v starlark.Value) any {
	switch x := v.(type) {
	case starlark.NoneType:
		return nil
	case starlark.Bool:
		return bool(x)
	case starlark.Int:
		if i, ok := x.Int64(); ok {
			return i
		}
		return x.String()
	case starlark.Float:
		return float64(x)
	case starlark.String:
		return string(x)
	case *starlark.List:
		result := make([]any, x.Len())
		for i := range x.Len() {
			result[i] = starlarkToGo(x.Index(i))
		}
		return result
	case starlark.Tuple:
		result := make([]any, len(x))
		for i, v := range x {
			result[i] = starlarkToGo(v)
		}
		return result
	case *starlark.Dict:
		result := make(map[string]any)
		for _, item := range x.Items() {
			key, ok := item[0].(starlark.String)
			if ok {
				result[string(key)] = starlarkToGo(item[1])
			}
		}
		return result
	default:
		return v.String()
	}
}

// TargetInfo contains introspection data about a target.
type TargetInfo struct {
	Name  string                 `json:"name"`
	Rule  string                 `json:"rule"`
	Attrs map[string]any `json:"attrs"`
}

// IntrospectTarget returns info about a RuleInstance.
func IntrospectTarget(ri *types.RuleInstance) *TargetInfo {
	info := &TargetInfo{
		Name:  ri.Name(),
		Rule:  ri.RuleClassName(),
		Attrs: make(map[string]any),
	}

	for _, name := range ri.AttrNames() {
		if v, ok := ri.GetAttrValue(name); ok {
			info.Attrs[name] = starlarkToGo(v)
		}
	}

	return info
}
