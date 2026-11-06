// Package gpu runs compute kernels on the system's graphics processor
// through Apple's Metal, for the work celestial does many million times
// over: an orbit for every star, a pixel for every direction.
//
// Kernels compile from their source when a session opens, so building
// celestial needs no Metal compiler toolchain.
//
// Each kernel package under gpu carries a reference in plain Go beside
// its kernel, and its tests state and measure how closely the two agree:
// the processor computes in 32-bit floats and Go in 64, so they agree
// closely and not bit for bit.
//
// Only darwin has Metal, so only darwin has a device to open a session
// on. Everywhere else the package is its documentation and its mock.
package gpu

// Device is what a session runs on: the gpu through its bridge, or a
// stand-in. It holds one source's kernels, opened once, and every call
// after is on them. Each call that can fail says why, and says nothing
// when it did not.
type Device interface {
	// Open compiles a source and its named kernels. When it cannot, it
	// says why, and whether that is for want of a gpu at all.
	Open(source string, kernels []string) (noDevice bool, why string)
	// Buffer makes a shared buffer of so many bytes, zeroed, and is its
	// number and its bytes, where the cpu and the gpu both see them.
	Buffer(bytes int) (index int, at []byte, why string)
	// Dispatch adds a kernel to the batch, its buffers bound from zero
	// in order and the params' bytes after them, over width by height
	// threads.
	Dispatch(kernel int, bound []int32, params []byte, width,
		height int) string
	// Commit runs the batch, in order, and waits for it.
	Commit() string
	// Seconds is how long the gpu has spent on every command run, by
	// the system's own timestamps on them.
	Seconds() float64
	// Close lets the kernels and the buffers go.
	Close()
}
