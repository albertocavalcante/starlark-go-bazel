package eval

import (
	"testing"

	"go.starlark.net/starlark"
)

// Plan 08 T2: every manifest entry marked Implemented or Stubbed is
// present in makeBzlPredeclared's output. Catches "added the builtin
// to builtins/, forgot to register it in makeBzlPredeclared" drift —
// the exact failure mode that hid the aspect omission until assay's
// E1c work surfaced it.
func TestPredeclared_ImplementedListResolves(t *testing.T) {
	dict := makeBzlPredeclared()
	for _, entry := range Manifest {
		if entry.Status != StatusImplemented && entry.Status != StatusStubbed {
			continue
		}
		if _, ok := dict[entry.Name]; !ok {
			t.Errorf("makeBzlPredeclared missing %q (status=%q, added=%s, docs=%s)",
				entry.Name, entry.Status, entry.AddedIn, entry.BazelDocsURL)
		}
	}
}

// Plan 08 T4: every Implemented manifest entry is mentioned in
// universeSrc. Without this, an entry can claim implemented status
// while the smoke test (TestPredeclared_UniverseEvalsAtVLatest in
// the eval_test package) never exercises it.
func TestPredeclared_ManifestExercisedByUniverse(t *testing.T) {
	for _, entry := range Manifest {
		if entry.Status != StatusImplemented {
			continue
		}
		if !containsToken(UniverseSrc, entry.Name) {
			t.Errorf("UniverseSrc doesn't exercise %q (manifest claims status=%s)",
				entry.Name, entry.Status)
		}
	}
}

// Guard against shape regressions: a manifest entry whose Kind says
// "module" should be a HasAttrs at runtime, "builtin" should be
// *starlark.Builtin, etc. Catches refactors that change a
// constructor's return type without updating the manifest.
func TestPredeclared_KindShapeMatches(t *testing.T) {
	dict := makeBzlPredeclared()
	for _, entry := range Manifest {
		if entry.Status != StatusImplemented && entry.Status != StatusStubbed {
			continue
		}
		v, ok := dict[entry.Name]
		if !ok {
			continue // covered by TestPredeclared_ImplementedListResolves
		}
		switch entry.Kind {
		case "builtin":
			if _, ok := v.(*starlark.Builtin); !ok {
				t.Errorf("%q: manifest Kind=builtin but runtime is %T", entry.Name, v)
			}
		case "module":
			if _, ok := v.(starlark.HasAttrs); !ok {
				t.Errorf("%q: manifest Kind=module but runtime doesn't implement HasAttrs (%T)", entry.Name, v)
			}
		case "constant":
			// Constants are starlark.Value but neither *Builtin nor
			// HasAttrs. No shape assertion beyond presence.
		}
	}
}

// containsToken matches name as a whole token, not substring — so
// looking for "rule" doesn't match "repository_rule". Whitespace,
// punctuation, and start/end-of-string all count as boundaries.
func containsToken(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] != needle {
			continue
		}
		if i > 0 && isIdentChar(haystack[i-1]) {
			continue
		}
		end := i + len(needle)
		if end < len(haystack) && isIdentChar(haystack[end]) {
			continue
		}
		return true
	}
	return false
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}
