// Package sdk is the shared guest-side runtime for Kosmos WASM plugins written in Go (TinyGo): the
// pointer/length-packed-into-i64 ABI, host-memory alloc/dealloc, and the permission-scoped
// env.http_fetch host import every plugin gets. Mirrors kosmos-plugin-sdk's Rust and AssemblyScript
// packages.
//
// Compile guests with `tinygo build -target=wasm-unknown -gc=leaking` — wasm-unknown is a
// freestanding target with no WASI imports (verified empirically: a minimal build with this
// target and -gc=leaking produces zero imports of its own), and leaking is a bump-style allocator
// that never frees, fine for short-lived per-call buffers. A guest's own main package must define
// //go:wasmexport-tagged alloc/dealloc functions that call Alloc/Dealloc below — the directive
// only produces a real WASM export when applied directly in the compiled entry package.
package sdk

import "unsafe"

//go:wasmimport env http_fetch
func hostHTTPFetch(ptr int32, length int32) int64

func Pack(ptr int32, length int32) int64 {
	return (int64(length) << 32) | int64(uint32(ptr))
}

func UnpackPtr(packed int64) int32 {
	return int32(packed)
}

func UnpackLen(packed int64) int32 {
	return int32(packed >> 32)
}

func Alloc(size int32) int32 {
	if size <= 0 {
		return 0
	}
	buf := make([]byte, size)
	return int32(uintptr(unsafe.Pointer(&buf[0])))
}

// Dealloc is a no-op — -gc=leaking never frees, matching the short-lived nature of a single guest
// call. Kept as a real function (not omitted) so the ABI stays symmetric with the Rust/AssemblyScript
// SDKs, whose allocators do reclaim.
func Dealloc(ptr int32, length int32) {}

func ReadString(ptr int32, length int32) string {
	return unsafe.String((*byte)(unsafe.Pointer(uintptr(ptr))), int(length))
}

func WriteString(s string) int64 {
	b := []byte(s)
	ptr := Alloc(int32(len(b)))
	if len(b) > 0 {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), len(b))
		copy(dst, b)
	}
	return Pack(ptr, int32(len(b)))
}

// HTTPFetch calls the host's permission-scoped http_fetch import. Returns either the real
// response body (host allowed the URL's host) or the host's own "blocked:<host>"/"error:..." text.
func HTTPFetch(url string) string {
	packed := WriteString(url)
	respPacked := hostHTTPFetch(UnpackPtr(packed), UnpackLen(packed))
	return ReadString(UnpackPtr(respPacked), UnpackLen(respPacked))
}
