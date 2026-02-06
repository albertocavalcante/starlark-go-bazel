package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/albertocavalcante/starlark-go-bazel/types"
)

// PrettyPrinter formats Starlark values for display.
type PrettyPrinter struct {
	indent string
	writer io.Writer
}

// NewPrettyPrinter creates a new PrettyPrinter.
func NewPrettyPrinter(w io.Writer) *PrettyPrinter {
	return &PrettyPrinter{
		indent: "  ",
		writer: w,
	}
}

// SetIndent sets the indentation string.
func (p *PrettyPrinter) SetIndent(indent string) {
	p.indent = indent
}

// PrintRule prints a rule class.
func (p *PrettyPrinter) PrintRule(rc *types.RuleClass) error {
	info := IntrospectRule(rc)
	return p.printJSON(info)
}

// PrintProvider prints a provider.
func (p *PrettyPrinter) PrintProvider(prov *types.Provider) error {
	info := IntrospectProvider(prov)
	return p.printJSON(info)
}

// PrintTarget prints a target.
func (p *PrettyPrinter) PrintTarget(ri *types.RuleInstance) error {
	info := IntrospectTarget(ri)
	return p.printJSON(info)
}

// PrintAttr prints an attribute.
func (p *PrettyPrinter) PrintAttr(attr *types.AttrDescriptor, name string) error {
	info := IntrospectAttr(attr)
	output := map[string]any{
		"name": name,
		"info": info,
	}
	return p.printJSON(output)
}

func (p *PrettyPrinter) printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", p.indent)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(p.writer, string(data))
	return err
}

// FormatRuleSummary returns a one-line summary of a rule.
func FormatRuleSummary(rc *types.RuleClass) string {
	var sb strings.Builder
	sb.WriteString(rc.Name())
	sb.WriteString("(")

	attrs := rc.Attrs()
	first := true
	for attrName, attr := range attrs {
		if !first {
			sb.WriteString(", ")
		}
		first = false
		sb.WriteString(attrName)
		if attr.Mandatory {
			sb.WriteString(" [required]")
		}
	}
	sb.WriteString(")")

	if rc.IsExecutable() {
		sb.WriteString(" [executable]")
	}
	if rc.IsTest() {
		sb.WriteString(" [test]")
	}

	return sb.String()
}

// FormatProviderSummary returns a one-line summary of a provider.
func FormatProviderSummary(prov *types.Provider) string {
	var sb strings.Builder
	sb.WriteString(prov.Name())
	sb.WriteString("(")
	sb.WriteString(strings.Join(prov.Fields(), ", "))
	sb.WriteString(")")
	return sb.String()
}

// FormatTargetSummary returns a one-line summary of a target.
func FormatTargetSummary(ri *types.RuleInstance) string {
	return fmt.Sprintf(":%s", ri.Name())
}
