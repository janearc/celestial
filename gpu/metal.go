//go:build darwin

package gpu

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Metal -framework Foundation
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import "unsafe"

// The bridge's own numbers, each defined by bridge.h.
const (
	room     = 1024 // the bytes the bridge may write to say why it failed
	set      = 1    // what the bridge sets a flag to
	passed   = 0    // what a call returns when it did not fail
	noBuffer = -1   // the number of a buffer that was not made
	spare    = 1    // room for one more, so an empty list has an address
)

// metal is the system's gpu through the bridge. Its handle is one
// source's kernels, opened once.
type metal struct{ handle unsafe.Pointer }

// Metal is the system's gpu, for a session to open on.
func Metal() Device { return &metal{} }

// Open compiles a source and its named kernels on the gpu.
func (device *metal) Open(source string, kernels []string) (bool, string) {
	src := C.CString(source)
	defer C.free(unsafe.Pointer(src))
	listed, free := named(kernels)
	defer free()
	var said [room]C.char
	var none C.int
	device.handle = C.celestial_gpu_open(src, listed, C.int(len(kernels)),
		&none, &said[passed], room)
	if device.handle == nil {
		return none == set, C.GoString(&said[passed])
	}
	return false, ""
}

// named is the kernel names as C strings in C memory, and how to let
// them go.
func named(kernels []string) (**C.char, func()) {
	names := make([]*C.char, len(kernels))
	for index, name := range kernels {
		names[index] = C.CString(name)
	}
	size := C.size_t(unsafe.Sizeof(uintptr(passed)))
	listed := (**C.char)(C.malloc(C.size_t(len(names)+spare) * size))
	copy(unsafe.Slice(listed, len(names)), names)
	return listed, letGo{names: names, listed: listed}.free
}

// letGo is C memory the bridge was handed, to free when the call is
// done with it.
type letGo struct {
	names  []*C.char
	listed **C.char
}

// free lets the memory go.
func (memory letGo) free() {
	for _, name := range memory.names {
		C.free(unsafe.Pointer(name))
	}
	C.free(unsafe.Pointer(memory.listed))
}

// Buffer makes a shared buffer of so many bytes on the gpu.
func (device *metal) Buffer(bytes int) (int, []byte, string) {
	var said [room]C.char
	index := C.celestial_gpu_buffer(device.handle, C.long(bytes),
		&said[passed], room)
	if index == noBuffer {
		return noBuffer, nil, C.GoString(&said[passed])
	}
	at := C.celestial_gpu_contents(device.handle, index)
	return int(index), unsafe.Slice((*byte)(at), bytes), ""
}

// Dispatch adds a kernel to the batch, its buffers bound in order and
// its params after them.
func (device *metal) Dispatch(kernel int, bound []int32, params []byte,
	width, height int) string {
	indices := make([]C.int, len(bound)+spare)
	for index, number := range bound {
		indices[index] = C.int(number)
	}
	var given unsafe.Pointer
	if len(params) > passed {
		given = C.CBytes(params)
		defer C.free(given)
	}
	var said [room]C.char
	failed := C.celestial_gpu_dispatch(device.handle, C.int(kernel),
		&indices[passed], C.int(len(bound)), given, C.int(len(params)),
		C.int(width), C.int(height), &said[passed], room)
	if failed != passed {
		return C.GoString(&said[passed])
	}
	return ""
}

// Commit runs the batch and waits for it.
func (device *metal) Commit() string {
	var said [room]C.char
	failed := C.celestial_gpu_commit(device.handle, &said[passed], room)
	if failed != passed {
		return C.GoString(&said[passed])
	}
	return ""
}

// Seconds is how long the gpu has spent on the kernels' commands.
func (device *metal) Seconds() float64 {
	return float64(C.celestial_gpu_seconds(device.handle))
}

// Close lets the kernels and the buffers go. Closing twice is harmless.
func (device *metal) Close() {
	C.celestial_gpu_close(device.handle)
	device.handle = nil
}
