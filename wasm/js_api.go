//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/albertocavalcante/starlark-go-bazel/analysis"
	"github.com/albertocavalcante/starlark-go-bazel/bzl"
	"github.com/albertocavalcante/starlark-go-bazel/types"
	"go.starlark.net/starlark"
)

// interpreterState holds interpreter and filesystem together.
type interpreterState struct {
	interp *bzl.Interpreter
	fs     *MemoryFS
}

var interpreters = make(map[int]*interpreterState)
var nextID = 1

// createInterpreter creates a new interpreter instance.
// Returns the interpreter ID.
func createInterpreter(this js.Value, args []js.Value) any {
	memFS := NewMemoryFS()
	opts := bzl.Options{
		FileSystem: memFS,
	}

	interp := bzl.New(opts)
	id := nextID
	nextID++
	interpreters[id] = &interpreterState{
		interp: interp,
		fs:     memFS,
	}

	return id
}

// addFile adds a file to the interpreter's filesystem.
// Args: (interpID, path, content)
func addFile(this js.Value, args []js.Value) any {
	if len(args) < 3 {
		return errorResult("addFile requires (interpID, path, content)")
	}

	id := args[0].Int()
	path := args[1].String()
	content := args[2].String()

	state, ok := interpreters[id]
	if !ok {
		return errorResult("invalid interpreter ID")
	}

	state.fs.AddFile(path, []byte(content))
	return successResultSimple("file added")
}

// evalFile evaluates a Starlark file.
// Args: (interpID, filename, source)
func evalFile(this js.Value, args []js.Value) any {
	if len(args) < 3 {
		return errorResult("evalFile requires (interpID, filename, source)")
	}

	id := args[0].Int()
	filename := args[1].String()
	source := args[2].String()

	state, ok := interpreters[id]
	if !ok {
		return errorResult("invalid interpreter ID")
	}

	// Add the source to the filesystem
	state.fs.AddFile(filename, []byte(source))

	result, err := state.interp.Eval(filename, []byte(source))
	if err != nil {
		return errorResult(err.Error())
	}

	return successResultWithData(result)
}

// getTargets returns the targets from the last BUILD file evaluation.
// Args: (interpID)
func getTargets(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("getTargets requires (interpID)")
	}

	id := args[0].Int()
	_, ok := interpreters[id]
	if !ok {
		return errorResult("invalid interpreter ID")
	}

	// Note: In a real implementation, we'd need to track the last result
	// For now, return empty
	return map[string]any{
		"success": true,
		"targets": []any{},
	}
}

// introspect introspects a value from the globals.
// Args: (interpID, filename, source, name)
func introspect(this js.Value, args []js.Value) any {
	if len(args) < 4 {
		return errorResult("introspect requires (interpID, filename, source, name)")
	}

	id := args[0].Int()
	filename := args[1].String()
	source := args[2].String()
	name := args[3].String()

	state, ok := interpreters[id]
	if !ok {
		return errorResult("invalid interpreter ID")
	}

	// Evaluate to get globals
	result, err := state.interp.Eval(filename, []byte(source))
	if err != nil {
		return errorResult(err.Error())
	}

	// Find the named value
	val, ok := result.Globals[name]
	if !ok {
		return errorResult("name not found in globals: " + name)
	}

	// Introspect based on type
	switch v := val.(type) {
	case *types.Provider:
		info := analysis.IntrospectProvider(v)
		data, _ := json.Marshal(info)
		return map[string]any{
			"success": true,
			"type":    "provider",
			"data":    string(data),
		}
	case *types.RuleClass:
		info := analysis.IntrospectRule(v)
		data, _ := json.Marshal(info)
		return map[string]any{
			"success": true,
			"type":    "rule",
			"data":    string(data),
		}
	default:
		return map[string]any{
			"success":   true,
			"type":      val.Type(),
			"stringRep": val.String(),
		}
	}
}

// destroyInterpreter destroys an interpreter instance.
// Args: (interpID)
func destroyInterpreter(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("destroyInterpreter requires (interpID)")
	}

	id := args[0].Int()
	if _, ok := interpreters[id]; !ok {
		return errorResult("invalid interpreter ID")
	}

	delete(interpreters, id)
	return successResultSimple("interpreter destroyed")
}

// errorResult creates an error result.
func errorResult(msg string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   msg,
	}
}

// successResultSimple creates a simple success result.
func successResultSimple(msg string) map[string]any {
	return map[string]any{
		"success": true,
		"message": msg,
	}
}

// successResultWithData creates a success result with evaluation data.
func successResultWithData(r *bzl.Result) map[string]any {
	result := map[string]any{
		"success": true,
	}

	// Convert globals
	globals := make(map[string]any)
	for name, val := range r.Globals {
		globals[name] = map[string]any{
			"type":   val.Type(),
			"string": val.String(),
		}
	}
	result["globals"] = globals

	// Convert targets
	if r.Targets != nil {
		targets := make(map[string]any)
		for name, target := range r.Targets {
			info := analysis.IntrospectTarget(target)
			targets[name] = info
		}
		result["targets"] = targets
	}

	return result
}

// starlarkToJS converts a Starlark value to a JS-friendly representation.
func starlarkToJS(v starlark.Value) any {
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
			result[i] = starlarkToJS(x.Index(i))
		}
		return result
	case starlark.Tuple:
		result := make([]any, len(x))
		for i, v := range x {
			result[i] = starlarkToJS(v)
		}
		return result
	case *starlark.Dict:
		result := make(map[string]any)
		for _, item := range x.Items() {
			key, ok := item[0].(starlark.String)
			if ok {
				result[string(key)] = starlarkToJS(item[1])
			}
		}
		return result
	default:
		return v.String()
	}
}
