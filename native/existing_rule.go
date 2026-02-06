package native

import (
	"fmt"
	"sort"

	"go.starlark.net/starlark"
)

// existingRule implements native.existing_rule(name).
//
// Returns an immutable dict-like object that describes the attributes of a rule
// instantiated in this thread's package, or None if no rule instance of that
// name exists.
//
// The result contains entries for:
//   - "name": the rule's name (string)
//   - "kind": the rule class (e.g., "cc_binary")
//   - All non-private attributes (names starting with a letter)
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#existingRule
// Reference: bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkNativeModuleApi.java#existingRule
func existingRule(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ctx := GetPackageContext(thread)
	if ctx == nil {
		return nil, fmt.Errorf("native.existing_rule() can only be called during BUILD file evaluation")
	}

	var name string
	if err := starlark.UnpackArgs("existing_rule", args, kwargs, "name", &name); err != nil {
		return nil, err
	}

	attrs := ctx.GetRule(name)
	if attrs == nil {
		return starlark.None, nil
	}

	return &ExistingRuleView{name: name, attrs: attrs}, nil
}

// existingRules implements native.existing_rules().
//
// Returns an immutable dict-like object describing the rules so far instantiated
// in this thread's package. Each entry maps the name of the rule instance to
// the result that would be returned by existing_rule(name).
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#existingRules
// Reference: bazel/src/main/java/com/google/devtools/build/lib/starlarkbuildapi/StarlarkNativeModuleApi.java#existingRules
func existingRules(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	ctx := GetPackageContext(thread)
	if ctx == nil {
		return nil, fmt.Errorf("native.existing_rules() can only be called during BUILD file evaluation")
	}

	if err := starlark.UnpackArgs("existing_rules", args, kwargs); err != nil {
		return nil, err
	}

	if ctx.Rules == nil {
		return &ExistingRulesView{rules: nil}, nil
	}

	// Take a snapshot of the rules at this point
	snapshot := make(map[string]map[string]starlark.Value, len(ctx.Rules))
	for name, attrs := range ctx.Rules {
		snapshot[name] = attrs
	}

	return &ExistingRulesView{rules: snapshot}, nil
}

// ExistingRuleView is an immutable dict-like view of a single rule's attributes.
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#ExistingRuleView
type ExistingRuleView struct {
	name  string
	attrs map[string]starlark.Value
}

var (
	_ starlark.Value     = (*ExistingRuleView)(nil)
	_ starlark.Mapping   = (*ExistingRuleView)(nil)
	_ starlark.Iterable  = (*ExistingRuleView)(nil)
	_ starlark.HasAttrs  = (*ExistingRuleView)(nil)
	_ starlark.Indexable = (*ExistingRuleView)(nil)
)

// String returns the Starlark representation.
func (v *ExistingRuleView) String() string {
	return fmt.Sprintf("<native.ExistingRuleView for target '%s'>", v.name)
}

// Type returns "existing_rule".
func (v *ExistingRuleView) Type() string { return "existing_rule" }

// Freeze is a no-op since the view is immutable.
func (v *ExistingRuleView) Freeze() {}

// Truth returns true.
func (v *ExistingRuleView) Truth() starlark.Bool { return true }

// Hash returns an error since dicts are not hashable.
func (v *ExistingRuleView) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: existing_rule")
}

// Len returns the number of entries.
func (v *ExistingRuleView) Len() int {
	count := 2 // "name" and "kind" are always present
	for attrName := range v.attrs {
		if isPotentiallyExportableAttribute(attrName) {
			count++
		}
	}
	return count
}

// Index implements indexing: v[key]
func (v *ExistingRuleView) Index(i int) starlark.Value {
	keys := v.keys()
	if i < 0 || i >= len(keys) {
		return nil
	}
	val, _, _ := v.Get(starlark.String(keys[i]))
	return val
}

// Get implements the Mapping interface.
func (v *ExistingRuleView) Get(key starlark.Value) (val starlark.Value, found bool, err error) {
	keyStr, ok := starlark.AsString(key)
	if !ok {
		return nil, false, nil
	}

	switch keyStr {
	case "name":
		return starlark.String(v.name), true, nil
	case "kind":
		if kind, ok := v.attrs["kind"]; ok {
			return kind, true, nil
		}
		// Default kind if not set
		return starlark.String(""), true, nil
	default:
		if !isPotentiallyExportableAttribute(keyStr) {
			return nil, false, nil
		}
		if val, ok := v.attrs[keyStr]; ok {
			return val, true, nil
		}
		return nil, false, nil
	}
}

// Iterate returns an iterator over the keys.
func (v *ExistingRuleView) Iterate() starlark.Iterator {
	return &existingRuleIterator{keys: v.keys(), index: 0}
}

// keys returns all the keys in iteration order.
func (v *ExistingRuleView) keys() []string {
	// "name" and "kind" come first, then attribute names sorted
	keys := []string{"name", "kind"}
	attrKeys := make([]string, 0, len(v.attrs))
	for k := range v.attrs {
		if isPotentiallyExportableAttribute(k) && k != "name" && k != "kind" {
			attrKeys = append(attrKeys, k)
		}
	}
	sort.Strings(attrKeys)
	return append(keys, attrKeys...)
}

// Attr implements starlark.HasAttrs for methods like .get(), .keys(), .values(), .items()
func (v *ExistingRuleView) Attr(name string) (starlark.Value, error) {
	switch name {
	case "get":
		return starlark.NewBuiltin("existing_rule.get", v.getMethod), nil
	case "keys":
		return starlark.NewBuiltin("existing_rule.keys", v.keysMethod), nil
	case "values":
		return starlark.NewBuiltin("existing_rule.values", v.valuesMethod), nil
	case "items":
		return starlark.NewBuiltin("existing_rule.items", v.itemsMethod), nil
	default:
		return nil, nil // nil means no such attribute
	}
}

// AttrNames returns the available method names.
func (v *ExistingRuleView) AttrNames() []string {
	return []string{"get", "items", "keys", "values"}
}

// getMethod implements the .get(key, default=None) method.
func (v *ExistingRuleView) getMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key starlark.Value
	var dflt starlark.Value = starlark.None

	if err := starlark.UnpackArgs("get", args, kwargs, "key", &key, "default?", &dflt); err != nil {
		return nil, err
	}

	val, found, err := v.Get(key)
	if err != nil {
		return nil, err
	}
	if !found {
		return dflt, nil
	}
	return val, nil
}

// keysMethod implements the .keys() method.
func (v *ExistingRuleView) keysMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("keys", args, kwargs); err != nil {
		return nil, err
	}

	keys := v.keys()
	vals := make([]starlark.Value, len(keys))
	for i, k := range keys {
		vals[i] = starlark.String(k)
	}
	return starlark.NewList(vals), nil
}

// valuesMethod implements the .values() method.
func (v *ExistingRuleView) valuesMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("values", args, kwargs); err != nil {
		return nil, err
	}

	keys := v.keys()
	vals := make([]starlark.Value, 0, len(keys))
	for _, k := range keys {
		val, found, _ := v.Get(starlark.String(k))
		if found {
			vals = append(vals, val)
		}
	}
	return starlark.NewList(vals), nil
}

// itemsMethod implements the .items() method.
func (v *ExistingRuleView) itemsMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("items", args, kwargs); err != nil {
		return nil, err
	}

	keys := v.keys()
	items := make([]starlark.Value, 0, len(keys))
	for _, k := range keys {
		val, found, _ := v.Get(starlark.String(k))
		if found {
			items = append(items, starlark.Tuple{starlark.String(k), val})
		}
	}
	return starlark.NewList(items), nil
}

// existingRuleIterator iterates over keys.
type existingRuleIterator struct {
	keys  []string
	index int
}

func (it *existingRuleIterator) Next(p *starlark.Value) bool {
	if it.index >= len(it.keys) {
		return false
	}
	*p = starlark.String(it.keys[it.index])
	it.index++
	return true
}

func (it *existingRuleIterator) Done() {}

// ExistingRulesView is an immutable dict-like view of all rules in a package.
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#ExistingRulesView
type ExistingRulesView struct {
	rules map[string]map[string]starlark.Value
}

var (
	_ starlark.Value    = (*ExistingRulesView)(nil)
	_ starlark.Mapping  = (*ExistingRulesView)(nil)
	_ starlark.Iterable = (*ExistingRulesView)(nil)
	_ starlark.HasAttrs = (*ExistingRulesView)(nil)
)

// String returns the Starlark representation.
func (v *ExistingRulesView) String() string {
	return "<native.ExistingRulesView object>"
}

// Type returns "existing_rules".
func (v *ExistingRulesView) Type() string { return "existing_rules" }

// Freeze is a no-op since the view is immutable.
func (v *ExistingRulesView) Freeze() {}

// Truth returns true.
func (v *ExistingRulesView) Truth() starlark.Bool { return true }

// Hash returns an error since dicts are not hashable.
func (v *ExistingRulesView) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: existing_rules")
}

// Len returns the number of rules.
func (v *ExistingRulesView) Len() int {
	if v.rules == nil {
		return 0
	}
	return len(v.rules)
}

// Get implements the Mapping interface.
func (v *ExistingRulesView) Get(key starlark.Value) (val starlark.Value, found bool, err error) {
	keyStr, ok := starlark.AsString(key)
	if !ok {
		return nil, false, nil
	}

	if v.rules == nil {
		return nil, false, nil
	}

	attrs, ok := v.rules[keyStr]
	if !ok {
		return nil, false, nil
	}

	return &ExistingRuleView{name: keyStr, attrs: attrs}, true, nil
}

// Iterate returns an iterator over the rule names.
func (v *ExistingRulesView) Iterate() starlark.Iterator {
	names := v.sortedNames()
	return &existingRulesIterator{names: names, index: 0}
}

func (v *ExistingRulesView) sortedNames() []string {
	if v.rules == nil {
		return nil
	}
	names := make([]string, 0, len(v.rules))
	for name := range v.rules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Attr implements starlark.HasAttrs for methods.
func (v *ExistingRulesView) Attr(name string) (starlark.Value, error) {
	switch name {
	case "get":
		return starlark.NewBuiltin("existing_rules.get", v.getMethod), nil
	case "keys":
		return starlark.NewBuiltin("existing_rules.keys", v.keysMethod), nil
	case "values":
		return starlark.NewBuiltin("existing_rules.values", v.valuesMethod), nil
	case "items":
		return starlark.NewBuiltin("existing_rules.items", v.itemsMethod), nil
	default:
		return nil, nil
	}
}

// AttrNames returns the available method names.
func (v *ExistingRulesView) AttrNames() []string {
	return []string{"get", "items", "keys", "values"}
}

// getMethod implements the .get(key, default=None) method.
func (v *ExistingRulesView) getMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key starlark.Value
	var dflt starlark.Value = starlark.None

	if err := starlark.UnpackArgs("get", args, kwargs, "key", &key, "default?", &dflt); err != nil {
		return nil, err
	}

	val, found, err := v.Get(key)
	if err != nil {
		return nil, err
	}
	if !found {
		return dflt, nil
	}
	return val, nil
}

// keysMethod implements the .keys() method.
func (v *ExistingRulesView) keysMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("keys", args, kwargs); err != nil {
		return nil, err
	}

	names := v.sortedNames()
	vals := make([]starlark.Value, len(names))
	for i, n := range names {
		vals[i] = starlark.String(n)
	}
	return starlark.NewList(vals), nil
}

// valuesMethod implements the .values() method.
func (v *ExistingRulesView) valuesMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("values", args, kwargs); err != nil {
		return nil, err
	}

	names := v.sortedNames()
	vals := make([]starlark.Value, len(names))
	for i, name := range names {
		vals[i] = &ExistingRuleView{name: name, attrs: v.rules[name]}
	}
	return starlark.NewList(vals), nil
}

// itemsMethod implements the .items() method.
func (v *ExistingRulesView) itemsMethod(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("items", args, kwargs); err != nil {
		return nil, err
	}

	names := v.sortedNames()
	items := make([]starlark.Value, len(names))
	for i, name := range names {
		items[i] = starlark.Tuple{
			starlark.String(name),
			&ExistingRuleView{name: name, attrs: v.rules[name]},
		}
	}
	return starlark.NewList(items), nil
}

// existingRulesIterator iterates over rule names.
type existingRulesIterator struct {
	names []string
	index int
}

func (it *existingRulesIterator) Next(p *starlark.Value) bool {
	if it.index >= len(it.names) {
		return false
	}
	*p = starlark.String(it.names[it.index])
	it.index++
	return true
}

func (it *existingRulesIterator) Done() {}

// isPotentiallyExportableAttribute returns true if the attribute can be exposed
// via existing_rule() and existing_rules().
//
// Reference: bazel/src/main/java/com/google/devtools/build/lib/packages/StarlarkNativeModule.java#isPotentiallyExportableAttribute
func isPotentiallyExportableAttribute(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Do not expose hidden or implicit attributes (those not starting with a letter)
	first := name[0]
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')
}
