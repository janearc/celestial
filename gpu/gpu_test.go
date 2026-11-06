//go:build darwin

package gpu_test

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"go.uber.org/mock/gomock"

	"github.com/janearc/celestial/gpu"
	"github.com/janearc/celestial/mocks"
)

// picked is the numbers these tests use, from testdata/gpu.json.
type picked struct {
	Count    int     `json:"count"`
	Batches  int     `json:"batches"`
	Doubling float64 `json:"doubling"`
	Scale    float64 `json:"scale"`
	Huge     int     `json:"huge"`
	None     int     `json:"none"`
	One      int     `json:"one"`
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/gpu.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// two kernels: one adding a buffer into another, one scaling a buffer
// by a parameter
const kernels = `
#include <metal_stdlib>
using namespace metal;
kernel void add(device float *into [[buffer(0)]],
                device const float *from [[buffer(1)]],
                uint at [[thread_position_in_grid]]) {
	into[at] += from[at];
}
kernel void scale(device float *into [[buffer(0)]],
                  constant float &by [[buffer(1)]],
                  uint at [[thread_position_in_grid]]) {
	into[at] *= by;
}
`

// session opens the kernels on the gpu, or skips with none.
func session(test *testing.T) *gpu.Session {
	test.Helper()
	made, err := gpu.Open(gpu.Metal(), kernels, "add", "scale")
	if gpu.NoDevice(err) {
		test.Skip(err)
	}
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(made.Close)
	return made
}

// scaled is a scale as the kernel's parameter: a little-endian float.
func scaled(by float64) []byte {
	bits := math.Float32bits(float32(by))
	return binary.LittleEndian.AppendUint32(nil, bits)
}

// a batch runs its dispatches in the order given, on buffers that keep
// what the last batch left and what go wrote between: numbers added in
// and doubled, so many batches over, come to what the same replay in go
// does.
func TestASessionKeepsItsBuffersAcrossBatches(test *testing.T) {
	numbers := pickedNumbers(test)
	made := session(test)
	both, err := made.Buffers(numbers.Count, numbers.Count)
	if err != nil {
		test.Fatal(err)
	}
	into, from := both[numbers.None], both[numbers.One]
	for index := range from.Numbers() {
		from.Numbers()[index] = float32(index)
	}
	for range numbers.Batches {
		if err := made.Dispatch("add", numbers.Count, numbers.One, nil,
			into, from); err != nil {
			test.Fatal(err)
		}
		if err := made.Dispatch("scale", numbers.Count, numbers.One,
			scaled(numbers.Doubling), into); err != nil {
			test.Fatal(err)
		}
		if err := made.Commit(); err != nil {
			test.Fatal(err)
		}
	}
	for index, got := range into.Numbers() {
		if want := replayed(numbers, index); got != want {
			test.Errorf("number %d: %g, want %g", index, got, want)
		}
	}
	if made.GPUTime() <= time.Duration(numbers.None) {
		test.Error("the gpu spent no time on the batches")
	}
}

// replayed is what the batches leave in a number, replayed in go.
func replayed(numbers picked, index int) float32 {
	held := float32(numbers.None)
	for range numbers.Batches {
		held = (held + float32(index)) * float32(numbers.Doubling)
	}
	return held
}

// params are bound after the buffers, as the kernel's next buffer.
func TestParamsFollowTheBuffers(test *testing.T) {
	numbers := pickedNumbers(test)
	made := session(test)
	only, err := made.Buffers(numbers.Count)
	if err != nil {
		test.Fatal(err)
	}
	into := only[numbers.None]
	for index := range into.Numbers() {
		into.Numbers()[index] = float32(numbers.One)
	}
	if err := made.Dispatch("scale", numbers.Count, numbers.One,
		scaled(numbers.Scale), into); err != nil {
		test.Fatal(err)
	}
	if err := made.Commit(); err != nil {
		test.Fatal(err)
	}
	for index, got := range into.Numbers() {
		if got != float32(numbers.Scale) {
			test.Errorf("number %d: %g, want %g", index, got, numbers.Scale)
		}
	}
}

// a commit with nothing dispatched is no work.
func TestAnEmptyBatchCommits(test *testing.T) {
	if err := session(test).Commit(); err != nil {
		test.Error(err)
	}
}

// the anti-tests: what the session and the gpu refuse, and how they
// say so.

// the anti-test: a kernel that does not compile is refused.
func TestAKernelThatDoesNotCompileIsRefused(test *testing.T) {
	_, err := gpu.Open(gpu.Metal(), "kernel void broken(", "broken")
	if gpu.NoDevice(err) {
		test.Skip(err)
	}
	if err == nil || !strings.Contains(err.Error(), "compile") {
		test.Errorf("a kernel that does not compile opened: %v", err)
	}
}

// the anti-test: a kernel the source does not hold is refused.
func TestAKernelTheSourceDoesNotHoldIsRefused(test *testing.T) {
	_, err := gpu.Open(gpu.Metal(), kernels, "add", "none")
	if gpu.NoDevice(err) {
		test.Skip(err)
	}
	if err == nil || !strings.Contains(err.Error(), "none") {
		test.Errorf("a kernel the source does not hold opened: %v", err)
	}
}

// the anti-test: a kernel in doubles does not compile.
// Metal has no 64-bit floats. Every kernel in celestial computes in
// 32 bits because of this, and this test is where that is checked: the
// day it fails, the kernels may be rewritten in doubles.
func TestMetalHasNoDouble(test *testing.T) {
	_, err := gpu.Open(gpu.Metal(), `
#include <metal_stdlib>
kernel void wide(device double *into [[buffer(0)]],
                 uint at [[thread_position_in_grid]]) {
	into[at] = 1.0;
}
`, "wide")
	if gpu.NoDevice(err) {
		test.Skip(err)
	}
	if err == nil {
		test.Fatal("a kernel in doubles compiled; celestial's kernels" +
			" compute in 32 bits on the belief that none can")
	}
}

// the anti-test: a kernel the session has no name for is refused.
func TestAnUnknownKernelIsRefusedAtTheDispatch(test *testing.T) {
	numbers := pickedNumbers(test)
	err := session(test).Dispatch("none", numbers.One, numbers.One, nil)
	if err == nil {
		test.Error("a kernel that is not there was dispatched")
	}
}

// the anti-test: a buffer of no numbers is refused.
func TestABufferOfNoNumbersIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	if _, err := session(test).Buffers(numbers.None); err == nil {
		test.Error("a buffer of no numbers was made")
	}
}

// the anti-test: a buffer bigger than the gpu holds is refused.
func TestABufferBiggerThanTheGPUHoldsIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	if _, err := session(test).Buffers(numbers.Huge); err == nil {
		test.Error("a buffer of terabytes was made")
	}
}

// the anti-test: a closed session refuses everything, and has no time.
func TestAClosedSessionRefusesEverything(test *testing.T) {
	numbers := pickedNumbers(test)
	made := session(test)
	only, err := made.Buffers(numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	made.Close()
	made.Close()
	if _, err := made.Buffers(numbers.One); err == nil {
		test.Error("a closed session made a buffer")
	}
	if err := made.Dispatch("add", numbers.One, numbers.One, nil,
		only[numbers.None]); err == nil {
		test.Error("a closed session took a dispatch")
	}
	if err := made.Commit(); err == nil {
		test.Error("a closed session committed")
	}
	if made.GPUTime() != time.Duration(numbers.None) {
		test.Error("a closed session has gpu time")
	}
}

// the anti-test: another session's buffer is refused, not thrown.
// a buffer of one session handed to another is refused by the gpu,
// not thrown: the other session holds no buffer by that number.
func TestAnotherSessionsBufferIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	first, second := session(test), session(test)
	two, err := first.Buffers(numbers.One, numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	err = second.Dispatch("add", numbers.One, numbers.One, nil,
		two[numbers.One])
	if err == nil || !strings.Contains(err.Error(), "no such buffer") {
		test.Errorf("another session's buffer was dispatched: %v", err)
	}
}

// the anti-test: the gpu refuses what is not open.
// the gpu, asked before anything is open on it, says so.
func TestTheGPURefusesWhatIsNotOpen(test *testing.T) {
	numbers := pickedNumbers(test)
	device := gpu.Metal()
	if _, _, why := device.Buffer(numbers.One); why != "no session" {
		test.Errorf("a buffer with nothing open: %q", why)
	}
	if why := device.Dispatch(numbers.None, nil, nil, numbers.One,
		numbers.One); why != "no session" {
		test.Errorf("a dispatch with nothing open: %q", why)
	}
	if why := device.Commit(); why != "no session" {
		test.Errorf("a commit with nothing open: %q", why)
	}
	if device.Seconds() != float64(numbers.None) {
		test.Error("the gpu has spent time on nothing")
	}
	device.Close()
}

// what a machine without a gpu says is not what one with says: the
// mock device stands in for the one these tests cannot be run on.

// mocked is a mock device under the test's control.
func mocked(test *testing.T) *mocks.MockDevice {
	return mocks.NewMockDevice(gomock.NewController(test))
}

// opened is a mock device that opens, and closes once.
func opened(test *testing.T) *mocks.MockDevice {
	numbers := pickedNumbers(test)
	device := mocked(test)
	device.EXPECT().Open(gomock.Any(), gomock.Any()).Return(false, "")
	device.EXPECT().Close().Times(numbers.One)
	return device
}

// bytes is a device's buffers in go: each is its bytes, numbered none.
type bytes struct{ numbers picked }

// buffer is a buffer of so many bytes.
func (made bytes) buffer(count int) (int, []byte, string) {
	return made.numbers.None, make([]byte, count), ""
}

// the anti-test: no device is its own error, for a test to skip on.
func TestNoDeviceIsItsOwnError(test *testing.T) {
	device := mocked(test)
	device.EXPECT().Open(kernels, []string{"add"}).Return(true,
		"no metal device")
	if _, err := gpu.Open(device, kernels, "add"); !gpu.NoDevice(err) {
		test.Errorf("no device gave %v", err)
	}
}

// the anti-test: a device that cannot compile is said.
func TestADeviceThatCannotCompileIsSaid(test *testing.T) {
	device := mocked(test)
	device.EXPECT().Open(gomock.Any(), gomock.Any()).Return(false,
		"compile: broken")
	_, err := gpu.Open(device, kernels, "add")
	if err == nil || !strings.Contains(err.Error(), "broken") {
		test.Errorf("a device that cannot compile: %v", err)
	}
}

// the anti-test: a buffer the device cannot make is said.
func TestABufferTheDeviceCannotMakeIsSaid(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test)
	device.EXPECT().Buffer(gomock.Any()).Return(numbers.None, nil,
		"could not make a buffer")
	made, err := gpu.Open(device, kernels, "add")
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	_, err = made.Buffers(numbers.One)
	if err == nil || !strings.Contains(err.Error(), "could not make") {
		test.Errorf("a buffer the device could not make: %v", err)
	}
}

// numberSize is the bytes in one of a buffer's numbers, a float32, as
// the session must ask the device for them.
func numberSize() int {
	var number float32
	return int(unsafe.Sizeof(number))
}

// the anti-test: a buffer a byte short is refused.
func TestAShortBufferIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test)
	bytes := numbers.Count * numberSize()
	device.EXPECT().Buffer(bytes).Return(numbers.None,
		make([]byte, bytes-numbers.One), "")
	made, err := gpu.Open(device, kernels, "add")
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	_, err = made.Buffers(numbers.Count)
	if err == nil || !strings.Contains(err.Error(), strconv.Itoa(bytes)) {
		test.Errorf("a buffer a byte short of %d: %v", bytes, err)
	}
}

// the anti-test: a closed session leaves its device alone.
// a closed session asks nothing more of its device: the mock would
// fail the test on any call after Close.
func TestAClosedSessionLeavesItsDeviceAlone(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test)
	device.EXPECT().Buffer(gomock.Any()).DoAndReturn(bytes{numbers}.buffer)
	made, err := gpu.Open(device, kernels, "add")
	if err != nil {
		test.Fatal(err)
	}
	only, err := made.Buffers(numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	made.Close()
	if _, err := made.Buffers(numbers.One); err == nil {
		test.Error("a closed session made a buffer")
	}
	if err := made.Dispatch("add", numbers.One, numbers.One, nil,
		only[numbers.None]); err == nil {
		test.Error("a closed session took a dispatch")
	}
	if err := made.Commit(); err == nil {
		test.Error("a closed session committed")
	}
	if made.GPUTime() != time.Duration(numbers.None) {
		test.Error("a closed session has gpu time")
	}
}

// the anti-test: a dispatch the device refuses is said.
func TestADispatchTheDeviceRefusesIsSaid(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test)
	device.EXPECT().Buffer(gomock.Any()).DoAndReturn(bytes{numbers}.buffer)
	device.EXPECT().Dispatch(numbers.None, []int32{int32(numbers.None)},
		nil, numbers.One, numbers.One).Return("no such kernel")
	made, err := gpu.Open(device, kernels, "add")
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	only, err := made.Buffers(numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	err = made.Dispatch("add", numbers.One, numbers.One, nil,
		only[numbers.None])
	if err == nil || !strings.Contains(err.Error(), "no such kernel") {
		test.Errorf("a dispatch the device refused: %v", err)
	}
}

// the anti-test: a commit the device fails is said.
func TestACommitTheDeviceFailsIsSaid(test *testing.T) {
	device := opened(test)
	device.EXPECT().Commit().Return("run: the command was stopped")
	made, err := gpu.Open(device, kernels, "add")
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	err = made.Commit()
	if err == nil || !strings.Contains(err.Error(), "stopped") {
		test.Errorf("a commit the device failed: %v", err)
	}
}

func TestGPUTimeIsTheDevicesSeconds(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test)
	device.EXPECT().Seconds().Return(numbers.Scale)
	made, err := gpu.Open(device, kernels, "add")
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	want := time.Duration(numbers.Scale * float64(time.Second))
	if got := made.GPUTime(); got != want {
		test.Errorf("the gpu time is %v, want %v", got, want)
	}
}
