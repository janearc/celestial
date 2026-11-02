package hole

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// GPUBlackHole is a hole and stationary-camera adapter for shaders.
// ShaderParams exports float32 constants; StepRay supplies CPU reference steps
// and diagnostics for comparison with a shader.
//
// No GPU work runs here. Batches step sequentially on one core. SetCamera
// mutates the adapter, so callers must synchronize concurrent use with camera
// changes. This adapter belongs to the proposed rendering-policy migration to
// Slayer.
type GPUBlackHole struct {
	hole      Hole
	cameraPos Vec3
	// The still observer's at cameraPos; not a number until a camera is set
	cameraVelocity Vec4
	cameraFrame    Frame
}

// RayStep records a stepped ray and its diagnostic invariants. Energy, axial
// angular momentum, Carter constant, and Hamiltonian should remain conserved;
// numerical drift measures stepping error.
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

// Redshift returns energy at the stationary camera divided by energy for a
// stationary emitter at the current point. It is NaN before a camera is set or
// where a stationary emitter cannot exist.
func (step RayStep) Redshift() float64 { return step.redshift }

// Energy is the energy the ray keeps.
func (step RayStep) Energy() float64 { return step.energy }

// AngularMomentum is the angular momentum about the spin axis that the
// ray keeps.
func (step RayStep) AngularMomentum() float64 { return step.angularMomentum }

// CarterConstant is the Carter constant the ray keeps.
func (step RayStep) CarterConstant() float64 { return step.carterConstant }

// Hamiltonian is the ray's H, which a geodesic keeps.
func (step RayStep) Hamiltonian() float64 { return step.hamiltonian }

// Radius is the ray's Kerr-schild radius after the step.
func (step RayStep) Radius() float64 { return step.radius }

// ShaderParams stores float32 geometry and camera constants. Its 96-byte shader
// layout is five scalars, three packed position values, and a 4×4 frame
// beginning at byte 32. Bytes serializes that contract.
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

// Bytes serializes the 96-byte shader layout in little-endian order: five
// scalars, three position values, then four frame rows.
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

// ErrNothingStill identifies a location where SetCamera cannot hold an observer
// stationary: inside the ergosurface or horizon. The function returns a stable
// error value for errors.Is comparisons.
func ErrNothingStill() error { return errNothingStill }

// NewGPUBlackHole validates mass and spin through NewHole and creates an
// adapter without a camera. Invalid geometry returns NewHole’s error.
func NewGPUBlackHole(mass, spin float64) (*GPUBlackHole, error) {
	hole, err := NewHole(mass, spin)
	if err != nil {
		return nil, err
	}
	notANumber := math.NaN()
	return &GPUBlackHole{
		hole: *hole,
		cameraVelocity: Vec4{
			time: notANumber, x: notANumber,
			y: notANumber, z: notANumber,
		},
	}, nil
}

// SetCamera installs a stationary observer and caches its frame for subsequent
// steps and shader parameters. Where that observer cannot exist, it returns
// ErrNothingStill and preserves the previous camera.
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

// StepRay advances a ray in place by one Runge–Kutta interval and reports the
// new state and invariants. Diagnostics cost more than twice the step alone;
// use Hole.Step or March when those reports are unnecessary.
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

// StepRaysBatch advances rays in order and returns one report per ray. It
// reuses reports when capacity permits; pass nil initially, then reuse the
// returned slice to avoid repeated allocation.
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

// Hole returns the underlying black-hole geometry.
func (gb *GPUBlackHole) Hole() Hole {
	return gb.hole
}

// ShaderParams returns float32 constants for the hole and current camera.
// Before SetCamera, the camera fields are zero.
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
