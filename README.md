# kosmos-plugin-sdk-go

Shared guest-side runtime for [Kosmos](https://github.com/kosmos-suite/kosmos) WASM plugins written
in Go (compiled via TinyGo) — the pointer/length-packed-into-i64 ABI, host-memory
`alloc`/`dealloc`, and the permission-scoped `env.http_fetch` host import every plugin gets.
Mirrors the Rust and AssemblyScript SDKs.

A new plugin depends on it via a local `replace` directive during local development:

```
// go.mod
require github.com/kosmos-suite/kosmos-plugin-sdk-go v0.0.0
replace github.com/kosmos-suite/kosmos-plugin-sdk-go => ../../kosmos-plugin-sdk-go
```

Once tagged and pushed, a plugin can depend on it normally instead — `go get
github.com/kosmos-suite/kosmos-plugin-sdk-go` — no `replace` needed.

and, at minimum:

```go
package main

import sdk "github.com/kosmos-suite/kosmos-plugin-sdk-go"

//go:wasmexport alloc
func alloc(size int32) int32 { return sdk.Alloc(size) }

//go:wasmexport dealloc
func dealloc(ptr int32, length int32) { sdk.Dealloc(ptr, length) }

//go:wasmexport myExport
func myExport(argPtr int32, argLen int32) int64 {
    arg := sdk.ReadString(argPtr, argLen)
    return sdk.WriteString("got: " + arg)
}

func main() {}
```

`alloc`/`dealloc` must be defined directly in the guest's own `main` package with their own
`//go:wasmexport` directives (not just imported and called) — that directive only produces a real
WASM export when applied in the compiled entry package.

Compile with:

```shell
tinygo build -o build/plugin.wasm -target=wasm-unknown -gc=leaking -no-debug -opt=z .
```

`wasm-unknown` is a freestanding target with no WASI imports (verified empirically — a minimal
build produces zero imports of its own), matching the sandboxed, import-free-except-`http_fetch`
model the other guest languages use. `-gc=leaking` is a bump-style allocator that never frees,
which is fine given each guest call's buffers live only as long as the WASM instance itself.

TinyGo's `wasm-unknown` output also exports a reactor-style `_initialize` function that must run
once before any other export is safe to call — `plugins.PluginHost` calls it automatically if
present, so guests don't need to do anything about this themselves.

See [kosmos-plugin-examples](https://github.com/kosmos-suite/kosmos-plugin-examples)'s `ping-go`
for a full working guest built on this SDK.
