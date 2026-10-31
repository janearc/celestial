package hole

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// GPUBlackHole is a hole and a camera held still beside it, as a shader
// wants them: ShaderParams is the two as float32 constants, and StepRay
// steps a ray on the processor and reports what it keeps, for holding a
// shader's own steps against.
//
// There is no GPU in it. It steps one ray at a time on one core, and
// its batches are loops. SetCamera changes it, so it is not to be
// shared between goroutines while one of them sets the camera.
type GPUBlackHole struct {
	hole      Hole
	cameraPos Vec3
	// the still observer's at cameraPos; not a number until a camera is set
	cameraVelocity Vec4
	cameraFrame    Frame
}

// RayStep is a ray after a step and what it keeps there: a geodesic
// holds Energy, AngularMomentum, CarterConstant and Hamiltonian fixed,
// so their drift from step to step is the stepping's error.
type RayStep struct {
	position        Vec4
	momentum        Vec4
	redshift        float64
	energy          float64
	angularMomentum float64
	carterConstant  float64
	hamiltonian     float64
	radius          float64
}

// Position is the ray's place in spacetime after the step.
func (step RayStep) Position() Vec4 { return step.position }

// Momentum is the ray's covariant momentum after the step.
func (step RayStep) Momentum() Vec4 { return step.momentum }

// Redshift is the photon's energy at the camera over its energy as an
// emitter held still where the ray now is would have it. It is not a
// number before a camera is set, and where nothing is still.
func (step RayStep) Redshift() float64 { return step.redshift }

// Energy is the energy the ray keeps.
func (step RayStep) Energy() float64 { return step.energy }

// AngularMomentum is the angular momentum about the spin axis that the
// ray keeps.
func (step RayStep) AngularMomentum() float64 { return step.angularMomentum }

// CarterConstant is the carter constant the ray keeps.
func (step RayStep) CarterConstant() float64 { return step.carterConstant }

// Hamiltonian is the ray's H, which a geodesic keeps.
func (step RayStep) Hamiltonian() float64 { return step.hamiltonian }

// Radius is the ray's kerr-schild radius after the step.
func (step RayStep) Radius() float64 { return step.radius }

// ShaderParams is the hole and the camera as constants for a shader, in
// float32, which is what a shader has.
//
// Its layout is a contract with the shader's own struct: five floats, a
// packed three, and a four by four at byte 32, 96 bytes in all. Bytes is
// that layout.
type ShaderParams struct {
	mass, spin, horizon, photonSphere, shadow float32
	cameraPos                                 [cameraPlaces]float32
	cameraFrame                               frameRows
}

// frameRows are a frame's four vectors, a row to each, in float32.
type frameRows [dimensions][dimensions]float32

// Mass is the hole's mass.
func (params ShaderParams) Mass() float32 { return params.mass }

// Spin is the hole's spin.
func (params ShaderParams) Spin() float32 { return params.spin }

// Horizon is the outer horizon's radius.
func (params ShaderParams) Horizon() float32 { return params.horizon }

// PhotonSphere is the radius of the photon sphere of a hole with no spin.
func (params ShaderParams) PhotonSphere() float32 { return params.photonSphere }

// Shadow is the radius of the shadow of a hole with no spin.
func (params ShaderParams) Shadow() float32 { return params.shadow }

// CameraPosition is the camera's place in space.
func (params ShaderParams) CameraPosition() [cameraPlaces]float32 {
	return params.cameraPos
}

// CameraFrame is the camera's tetrad, one vector to a row, the observer's
// four-velocity first.
func (params ShaderParams) CameraFrame() [dimensions][dimensions]float32 {
	return params.cameraFrame
}

// Bytes is the constants as the shader's struct holds them: little endian,
// the five scalars first, then the camera's position, then the frame a row
// at a time. It is 96 bytes.
func (params ShaderParams) Bytes() []byte {
	values := []float32{
		params.mass, params.spin, params.horizon,
		params.photonSphere, params.shadow,
	}
	values = append(values, params.cameraPos[:]...)
	for _, row := range params.cameraFrame {
		values = append(values, row[:]...)
	}
	var encoded []byte
	for _, value := range values {
		encoded = binary.LittleEndian.AppendUint32(
			encoded, math.Float32bits(value),
		)
	}
	return encoded
}

// errNothingStill is what ErrNothingStill answers.
var errNothingStill = errors.New("hole: nothing can be held still there")

// ErrNothingStill is SetCamera's refusal of a point where no observer
// can be held still: inside the ergosurface of a spinning hole, or
// inside the horizon of any.
//
// It is a function so that it cannot be changed behind a caller's back;
// errors.Is finds it in what SetCamera returns.
func ErrNothingStill() error { return errNothingStill }

// NewGPUBlackHole is a hole of a mass and a spin with no camera yet, or
// nil if they are no hole's, as NewHole says. SetCamera gives it one.
func NewGPUBlackHole(mass, spin float64) *GPUBlackHole {
	hole := NewHole(mass, spin)
	if hole == nil {
		return nil
	}
	notANumber := math.NaN()
	return &GPUBlackHole{
		hole: *hole,
		cameraVelocity: Vec4{
			time: notANumber, x: notANumber,
			y: notANumber, z: notANumber,
		},
	}
}

// SetCamera puts a camera held still at a point and works out its
// frame, once, for every step and every ShaderParams after it.
//
// Where nothing is still there is no such camera: it answers
// ErrNothingStill and leaves the camera where it was.
func (gb *GPUBlackHole) SetCamera(position Vec3) error {
	velocity := gb.hole.Static(position)
	if math.IsNaN(velocity.time) {
		return fmt.Errorf(
			"%w: (%v, %v, %v), at radius %v of a hole of mass %v "+
				"and spin %v",
			errNothingStill, position.x, position.y, position.z,
			gb.hole.Radius(position), gb.hole.mass, gb.hole.spin,
		)
	}
	gb.cameraPos, gb.cameraVelocity = position, velocity
	gb.cameraFrame = gb.hole.FrameOf(position, velocity)
	return nil
}

// StepRay takes a ray one runge-kutta step of a size, in place, and
// reports where it now is and what it keeps there.
//
// The report costs more than twice what the step does, so a march that
// does not read it is better made with Hole's March or Step.
func (gb *GPUBlackHole) StepRay(ray *Ray, stepSize float64) RayStep {
	stepped := gb.hole.Step(*ray, stepSize)
	*ray = stepped
	emitter := gb.hole.Static(stepped.Space())
	return RayStep{
		position: stepped.position,
		momentum: stepped.momentum,
		redshift: Shift(
			stepped.momentum, gb.cameraVelocity,
			stepped.momentum, emitter,
		),
		energy:          gb.hole.Energy(stepped),
		angularMomentum: gb.hole.Lz(stepped),
		carterConstant:  gb.hole.Carter(stepped),
		hamiltonian:     gb.hole.Hamiltonian(stepped),
		radius:          gb.hole.Radius(stepped.Space()),
	}
}

// StepRaysBatch takes each ray of a slice one step of a size, in place,
// and reports on each in the same order.
//
// The reports go into the slice handed in when it is long enough, so a
// caller that keeps handing back what this returns allocates once; nil
// will do for the first call.
func (gb *GPUBlackHole) StepRaysBatch(
	reports []RayStep, rays []Ray, stepSize float64,
) []RayStep {
	if cap(reports) < len(rays) {
		reports = make([]RayStep, len(rays))
	}
	reports = reports[:len(rays)]
	for i := range rays {
		reports[i] = gb.StepRay(&rays[i], stepSize)
	}
	return reports
}

// Hole is the hole itself, for everything a Hole can be asked: its
// metric, its marches, what its rays keep.
func (gb *GPUBlackHole) Hole() Hole {
	return gb.hole
}

// ShaderParams is the hole and the camera as constants for a shader.
// Before a camera is set the camera's part is all nought.
func (gb *GPUBlackHole) ShaderParams() ShaderParams {
	params := ShaderParams{
		mass:         float32(gb.hole.mass),
		spin:         float32(gb.hole.spin),
		horizon:      float32(gb.hole.Horizon()),
		photonSphere: float32(gb.hole.PhotonSphere()),
		shadow:       float32(gb.hole.Shadow()),
		cameraPos: [cameraPlaces]float32{
			float32(gb.cameraPos.x), float32(gb.cameraPos.y),
			float32(gb.cameraPos.z),
		},
	}
	for i, row := range gb.cameraFrame.rows() {
		for column, part := range row.parts() {
			params.cameraFrame[i][column] = float32(part)
		}
	}
	return params
}
