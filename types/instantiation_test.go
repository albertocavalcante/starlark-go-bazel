package types_test

import (
	"testing"

	"go.starlark.net/starlark"

	"github.com/albertocavalcante/starlark-go-bazel/taint"
	"github.com/albertocavalcante/starlark-go-bazel/types"
)

func TestRepositoryRuleFromInstantiation_HappyPath(t *testing.T) {
	impl := starlark.NewBuiltin("noop", func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
		return starlark.None, nil
	})
	rc := types.NewRepositoryRuleClass(impl, nil)
	inst := taint.RuleInstantiation{Rule: rc, Attrs: nil}

	got := types.RepositoryRuleFromInstantiation(inst)
	if got != rc {
		t.Errorf("got %p, want %p", got, rc)
	}
}

func TestRepositoryRuleFromInstantiation_NilRule(t *testing.T) {
	inst := taint.RuleInstantiation{Rule: nil}
	if got := types.RepositoryRuleFromInstantiation(inst); got != nil {
		t.Errorf("got %v, want nil for nil-Rule instantiation", got)
	}
}

func TestRepositoryRuleFromInstantiation_WrongType(t *testing.T) {
	// A captured-via-the-wrong-path value should yield nil rather
	// than panic.
	inst := taint.RuleInstantiation{Rule: starlark.String("not a rule class")}
	if got := types.RepositoryRuleFromInstantiation(inst); got != nil {
		t.Errorf("got %v, want nil for wrong-type Rule", got)
	}
}
