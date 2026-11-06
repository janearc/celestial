package gpu

import (
	"errors"
	"fmt"
	"runtime"
	"time"
	"unsafe"
)

// errNoDevice is a machine with no gpu Metal can use.
var errNoDevice = errors.New("gpu: no metal device")

// NoDevice says whether an error is for want of a gpu. Tests that need
// the gpu skip on this, and only this: a kernel that fails to compile
// must fail its test, not skip it.
func NoDevice(err error) bool { return errors.Is(err, errNoDevice) }

// errClosed is a session used after Close.
var errClosed = errors.New("gpu: the session is closed")

// The session's own numbers, each defined.
const (
	numberSize = 4 // defined by float32: the bytes in a buffer's number
	one        = 1 // defined: the fewest numbers a buffer holds
)

// Session is many kernels compiled once from one source on a device,
// buffers that live there as long as it does, and dispatches gathered
// into a batch and run as one command: for work that steps many times
// without coming back.
//
// Its buffers are memory the cpu and the gpu share, so Go reads and
// writes them where they lie. They lie there until Close, and nothing
// closes a session but Close: a session left open keeps its buffers,
// and a buffer's numbers read after Close are nobody's.
type Session struct {
	device  Device
	open    bool
	kernels map[string]int
}

// Buffer is a session's buffer of numbers.
type Buffer struct {
	index   int
	numbers []float32
}

// Numbers is the buffer's numbers, where they lie: what Go writes there
// the gpu reads, and what the gpu wrote there Go reads, between batches.
func (buffer Buffer) Numbers() []float32 { return buffer.numbers }

// Open compiles a source and each named kernel in it on a device.
func Open(device Device, source string, kernels ...string) (*Session,
	error) {
	noDevice, why := device.Open(source, kernels)
	if noDevice {
		return nil, errNoDevice
	}
	if why != "" {
		return nil, errors.New("gpu: " + why)
	}
	made := &Session{device: device, open: true, kernels: map[string]int{}}
	for index, name := range kernels {
		made.kernels[name] = index
	}
	return made, nil
}

// Buffers is a new buffer for each count of numbers, all zero; or none
// of them, when one could not be made. The ones made before the one
// that could not be stay on the device until Close.
func (session *Session) Buffers(counts ...int) ([]Buffer, error) {
	made := make([]Buffer, len(counts))
	for index, count := range counts {
		buffer, err := session.buffer(count)
		if err != nil {
			return nil, err
		}
		made[index] = *buffer
	}
	return made, nil
}

// buffer is a new buffer of so many numbers, all zero.
func (session *Session) buffer(count int) (*Buffer, error) {
	if !session.open {
		return nil, errClosed
	}
	if count < one {
		return nil, fmt.Errorf("gpu: a buffer of %d numbers", count)
	}
	index, at, why := session.device.Buffer(count * numberSize)
	if why != "" {
		return nil, errors.New("gpu: " + why)
	}
	// the gpu's own buffer is always the bytes asked for; this is for a
	// device standing in, which is held to the same word
	if len(at) < count*numberSize {
		return nil, fmt.Errorf("gpu: the device made %d bytes of %d",
			len(at), count*numberSize)
	}
	data := unsafe.SliceData(at)
	first := (*float32)(unsafe.Pointer(data))
	return &Buffer{index: index, numbers: unsafe.Slice(first, count)}, nil
}

// Dispatch adds a kernel to the batch, over width by height threads,
// its buffers bound in order and its params after them.
func (session *Session) Dispatch(kernel string, width, height int,
	params []byte, buffers ...Buffer) error {
	if !session.open {
		return errClosed
	}
	which, known := session.kernels[kernel]
	if !known {
		return fmt.Errorf("gpu: no kernel %s in the session", kernel)
	}
	bound := make([]int32, len(buffers))
	for index, buffer := range buffers {
		bound[index] = int32(buffer.index)
	}
	why := session.device.Dispatch(which, bound, params, width, height)
	if why != "" {
		return errors.New("gpu: " + why)
	}
	return nil
}

// Commit runs the batch, in the order it was given, and waits for it.
func (session *Session) Commit() error {
	if !session.open {
		return errClosed
	}
	why := session.device.Commit()
	runtime.KeepAlive(session)
	if why != "" {
		return errors.New("gpu: " + why)
	}
	return nil
}

// GPUTime is how long the gpu has spent on the session's batches, by
// the system's own timestamps on its commands: the time the work took on
// the gpu, which is less than the time a caller waited for it.
func (session *Session) GPUTime() time.Duration {
	if !session.open {
		return time.Duration(none)
	}
	seconds := session.device.Seconds()
	runtime.KeepAlive(session)
	return time.Duration(seconds * float64(time.Second))
}

// none is no time at all. Defined.
const none = 0

// Close lets the session and its buffers go; their numbers are not to be
// read after. Closing twice is harmless.
func (session *Session) Close() {
	if session.open {
		session.device.Close()
		session.open = false
	}
}
