package hole_test

import (
	"encoding/binary"
	"math"
	"testing"
)

// places is the three of the camera's position. It is defined by space.
const places = 3

// floatAt is the float32 at a byte offset of a shader's struct, little
// endian.
func floatAt(encoded []byte, offset int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(encoded[offset:]))
}

// Before a camera is set the camera's part is all nought, and the hole's is
// the hole's as float32.
func TestShaderParamsBeforeACamera(t *testing.T) {
	numbers := pickedNumbers(t)
	from := numbers.Holes.Fast
	gpu := gpuOf(t, from)
	params := gpu.ShaderParams()
	black := gpu.Hole()
	if params.Mass() != float32(from.Mass) ||
		params.Spin() != float32(from.Spin) ||
		params.Horizon() != float32(black.Horizon()) ||
		params.PhotonSphere() != float32(black.PhotonSphere()) ||
		params.Shadow() != float32(black.Shadow()) {
		t.Errorf("the hole's part of %v is wrong", params)
	}
	var nothing3 [places]float32
	var nothing4 [dimensions][dimensions]float32
	if params.CameraPosition() != nothing3 || params.CameraFrame() != nothing4 {
		t.Errorf("before a camera the camera's part is not nought")
	}
}

// With a camera set the position is the camera's and the frame is the still
// observer's tetrad there, a vector to a row, as float32; and the hole's
// part has not moved.
func TestShaderParamsHoldTheCamera(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := camera(t, numbers.Holes.Fast, numbers.Places.Eye)
	before := gpuOf(t, numbers.Holes.Fast).ShaderParams()
	params := gpu.ShaderParams()
	eye := numbers.Places.Eye
	wantAt := [places]float32{
		float32(eye.AlongX), float32(eye.AlongY), float32(eye.AlongZ),
	}
	if params.CameraPosition() != wantAt {
		t.Errorf("the camera is at %v, want %v", params.CameraPosition(),
			wantAt)
	}
	black := gpu.Hole()
	at := vec3(eye)
	frame := black.FrameOf(at, black.Static(at))
	for row, vector := range axesOf(frame) {
		for column, part := range partsOf(vector) {
			if params.CameraFrame()[row][column] != float32(part) {
				t.Errorf("frame row %d place %d is %v, want %v", row,
					column, params.CameraFrame()[row][column], float32(part))
			}
		}
	}
	if params.Mass() != before.Mass() || params.Shadow() != before.Shadow() {
		t.Errorf("setting a camera moved the hole's part")
	}
}

// Bytes is the shader's struct: the size and the offsets the data names,
// little endian.
func TestShaderParamsBytesAreTheShadersLayout(t *testing.T) {
	numbers := pickedNumbers(t)
	layout := numbers.GPU.Layout
	gpu := camera(t, numbers.Holes.Fast, numbers.Places.Eye)
	params := gpu.ShaderParams()
	encoded := params.Bytes()
	if len(encoded) != layout.Size {
		t.Fatalf("the shader's struct is %d bytes, want %d", len(encoded),
			layout.Size)
	}
	for _, scalar := range []struct {
		name   string
		offset int
		want   float32
	}{
		{"mass", layout.Mass, params.Mass()},
		{"spin", layout.Spin, params.Spin()},
		{"horizon", layout.Horizon, params.Horizon()},
		{"photon sphere", layout.PhotonSphere, params.PhotonSphere()},
		{"shadow", layout.Shadow, params.Shadow()},
	} {
		if got := floatAt(encoded, scalar.offset); got != scalar.want {
			t.Errorf("%s at byte %d is %v, want %v", scalar.name,
				scalar.offset, got, scalar.want)
		}
	}
	for index, want := range params.CameraPosition() {
		got := floatAt(encoded, layout.Camera+index*layout.Float)
		if got != want {
			t.Errorf("camera place %d is %v, want %v", index, got, want)
		}
	}
	for row, vector := range params.CameraFrame() {
		for column, want := range vector {
			offset := layout.Frame + (row*dimensions+column)*layout.Float
			if got := floatAt(encoded, offset); got != want {
				t.Errorf("frame %d, %d at byte %d is %v, want %v", row,
					column, offset, got, want)
			}
		}
	}
}

// The layout has no camera in it before one is set: the camera's bytes are
// nought, and the hole's are not.
func TestShaderParamsBytesBeforeACameraAreNought(t *testing.T) {
	numbers := pickedNumbers(t)
	layout := numbers.GPU.Layout
	from := numbers.Holes.Fast
	gpu := gpuOf(t, from)
	encoded := gpu.ShaderParams().Bytes()
	for offset := layout.Camera; offset < layout.Size; offset++ {
		if encoded[offset] != none {
			t.Fatalf("byte %d is %d before a camera is set", offset,
				encoded[offset])
		}
	}
	if floatAt(encoded, layout.Mass) != float32(from.Mass) {
		t.Errorf("the mass is not in the bytes")
	}
}
