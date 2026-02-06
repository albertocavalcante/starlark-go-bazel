# starlark-go-bazel

[![CI](https://github.com/albertocavalcante/starlark-go-bazel/actions/workflows/ci.yml/badge.svg)](https://github.com/albertocavalcante/starlark-go-bazel/actions/workflows/ci.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=albertocavalcante_starlark-go-bazel&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=albertocavalcante_starlark-go-bazel)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=albertocavalcante_starlark-go-bazel&metric=coverage)](https://sonarcloud.io/summary/new_code?id=albertocavalcante_starlark-go-bazel)
[![Go Reference](https://pkg.go.dev/badge/github.com/albertocavalcante/starlark-go-bazel.svg)](https://pkg.go.dev/github.com/albertocavalcante/starlark-go-bazel)
[![Go Report Card](https://goreportcard.com/badge/github.com/albertocavalcante/starlark-go-bazel)](https://goreportcard.com/report/github.com/albertocavalcante/starlark-go-bazel)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Go implementation of Bazel's Starlark dialect. Execute `.bzl` and `BUILD` files with full Bazel builtins, compile to WASM for browser-based tools, and evaluate rules without requiring actual build execution.

## Features

- **Full Bazel Builtins** — `rule()`, `provider()`, `aspect()`, `select()`, `depset()`, `struct()`
- **Attribute Module** — Complete `attr.*` support (string, int, bool, label, label_list, etc.)
- **Native Module** — `native.glob()`, `native.existing_rule()`, `native.package_name()`
- **Rule Context** — Full `ctx` object with `ctx.actions`, `ctx.attr`, `ctx.files`
- **Built-in Providers** — `DefaultInfo`, `OutputGroupInfo`, `Runfiles`
- **WASM Support** — Compile to WebAssembly for browser-based educational tools
- **Introspection** — Analyze rules, providers, and targets programmatically

## Installation

```bash
go get github.com/albertocavalcante/starlark-go-bazel
```

Requires Go 1.24+.

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/albertocavalcante/starlark-go-bazel/bzl"
)

func main() {
    interp := bzl.New(bzl.Options{})

    result, err := interp.Eval("rules.bzl", []byte(`
my_provider = provider(fields = ["x", "y"])

def _impl(ctx):
    out = ctx.actions.declare_file(ctx.label.name + ".txt")
    ctx.actions.write(out, "hello")
    return [DefaultInfo(files = depset([out]))]

my_rule = rule(
    implementation = _impl,
    attrs = {"srcs": attr.label_list()},
)
    `))
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Exported symbols:", result.Globals.Keys())
}
```

## Package Structure

| Package | Description |
|---------|-------------|
| [`bzl`](bzl/) | Main interpreter entry point |
| [`types`](types/) | Core types: Label, Provider, Depset, RuleClass, File |
| [`attr`](attr/) | Attribute module (`attr.string()`, `attr.label()`, etc.) |
| [`builtins`](builtins/) | Built-in functions: `rule()`, `provider()`, `aspect()` |
| [`native`](native/) | Native module: `glob()`, `existing_rule()` |
| [`ctx`](ctx/) | Rule context object |
| [`providers`](providers/) | DefaultInfo, OutputGroupInfo, Runfiles |
| [`eval`](eval/) | Evaluation engine for .bzl and BUILD files |
| [`loader`](loader/) | Module loading with caching and cycle detection |
| [`analysis`](analysis/) | Introspection and pretty-printing utilities |
| [`wasm`](wasm/) | WebAssembly/JavaScript bindings |

## WASM Usage

Build for WebAssembly:

```bash
GOOS=js GOARCH=wasm go build -o main.wasm ./wasm/
```

Use in JavaScript:

```javascript
const go = new Go();
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
    go.run(result.instance);

    const interp = goBzl.createInterpreter();
    goBzl.addFile(interp, "rules.bzl", `
        my_rule = rule(implementation = lambda ctx: [], attrs = {})
    `);

    const result = goBzl.evalFile(interp, "rules.bzl");
    console.log(goBzl.introspect(interp, "my_rule"));
});
```

## Supported Builtins

### Core Functions

| Function | Description |
|----------|-------------|
| `rule()` | Define a rule with implementation and attributes |
| `provider()` | Define a provider schema |
| `aspect()` | Define an aspect |
| `select()` | Configurable attribute values |
| `depset()` | Create efficient nested sets |
| `struct()` | Create immutable structs |
| `Label()` | Parse label strings |

### Attribute Types

| Function | Description |
|----------|-------------|
| `attr.string()` | String attribute |
| `attr.int()` | Integer attribute |
| `attr.bool()` | Boolean attribute |
| `attr.label()` | Single label dependency |
| `attr.label_list()` | List of label dependencies |
| `attr.string_list()` | List of strings |
| `attr.string_dict()` | String to string dictionary |
| `attr.output()` | Declared output file |
| `attr.output_list()` | List of declared outputs |

### Native Functions

| Function | Description |
|----------|-------------|
| `native.glob()` | File pattern matching |
| `native.existing_rule()` | Get rule by name |
| `native.existing_rules()` | Get all rules in package |
| `native.package_name()` | Current package path |
| `native.repository_name()` | Current repository name |

## Dependencies

- [`go.starlark.net`](https://pkg.go.dev/go.starlark.net) — Base Starlark interpreter (pure Go, WASM-compatible)

## License

MIT — see [LICENSE](LICENSE) for details.
