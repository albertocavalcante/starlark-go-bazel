//go:build js && wasm

package main

import (
	"syscall/js"
)

func main() {
	// Register all API functions
	js.Global().Set("goBzl", js.ValueOf(map[string]any{
		"createInterpreter":  js.FuncOf(createInterpreter),
		"destroyInterpreter": js.FuncOf(destroyInterpreter),
		"evalFile":           js.FuncOf(evalFile),
		"getTargets":         js.FuncOf(getTargets),
		"introspect":         js.FuncOf(introspect),
		"addFile":            js.FuncOf(addFile),
	}))

	// Keep the Go runtime alive
	<-make(chan struct{})
}
