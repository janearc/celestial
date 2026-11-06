//go:build darwin

// Package orbit is celestial/orbit's stepping as a kernel on the gpu:
// every star at once, each on its own thread, in 32-bit floats, by the
// same kick, drift, kick. Its reference is celestial/orbit, and its
// tests measure how closely the two agree.
//
// The tests hold the kernel to the reference on passes from a tenth of
// a kiloparsec out, and on a circle a parsec from the centre.
//
// A plunge from kiloparsecs to parsecs, the regime a search for stars
// thrown from the centre is after, wants steps planned for its nearest
// pass and a galactic potential, and is not tested yet.
package orbit

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/janearc/celestial/gpu"
	"github.com/janearc/celestial/grav"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/orbit"
	"github.com/janearc/celestial/research"
)

// Field is a potential as the kernel wants it: its pull in the Metal
// shading language, the numbers that pull reads, and the same potential
// in Go, for the kernel to be held to.
type Field struct {
	source    string
	numbers   []float32
	reference orbit.Potential
}

// NewField is a field from its pull in Metal, which defines
//
//	static inline float3 pull(float3 at, constant float *field)
//
// as the acceleration at a place in kiloparsecs, in kilometres per
// second over one unit of time; the numbers that pull reads as field;
// and the same potential in Go.
func NewField(source string, numbers []float32,
	reference orbit.Potential,
) Field {
	return Field{source: source, numbers: numbers, reference: reference}
}

// Reference is the potential in Go.
func (field Field) Reference() orbit.Potential { return field.reference }

// keplerPull is the pull of one point mass at the centre, in Metal:
// field[gm] is its gm. At the centre it is not a number, and an orbit
// through it runs away.
const keplerPull = `
static inline float3 pull(float3 at, constant float *field) {
	float r = length(at);
	return at * (-field[gm] / (r * r * r));
}
`

// Kepler is the pull of a body at the centre.
func Kepler(body grav.Body) Field {
	reference := orbit.NewKepler(body)
	gm := float32(reference.GM())
	return NewField(keplerPull, []float32{gm}, reference)
}

// The kernel's numbers, each defined, and handed to it through the
// prelude so it and Go read them from one place.
const (
	perStar   = 6 // defined: a star's numbers, its place and motion
	perPlace  = 3 // defined: a place's numbers, and a pull's
	along     = 3 // defined: where a star's motion begins among them
	perPass   = 4 // defined: a pass's numbers in the found buffer
	nearAt    = 0 // defined: where its nearness is among them
	agoAt     = 1 // defined: where its age is among them
	movedAt   = 2 // defined: where the step's share in the motion is
	pulledAt  = 3 // defined: where the step's share in the pull is
	gmAt      = 0 // defined: where a kepler field's gm is in its numbers
	firstStep = 1 // defined: steps are counted from one, as orbit does
	kicks     = 2 // defined by the leapfrog: a kick split in two
)

// The axes' places among a star's numbers, each defined.
const (
	axisX = iota
	axisY
	axisZ
)

// prelude is what the kernel source begins with: Metal's library, and
// the kernel's numbers as Go holds them.
func prelude() string {
	return fmt.Sprintf(`
#include <metal_stdlib>
using namespace metal;
constant uint perStar = %d, along = %d, perPass = %d, perPlace = %d;
constant uint nearAt = %d, agoAt = %d, movedAt = %d, pulledAt = %d;
constant uint gm = %d, firstStep = %d;
constant uint X = %d, Y = %d, Z = %d;
constant float kicks = %d;
`, perStar, along, perPass, perPlace, nearAt, agoAt, movedAt, pulledAt,
		gmAt, firstStep, axisX, axisY, axisZ, kicks)
}

// pulling is the other kernel: a field's pull at each place, once, for
// holding the kernel's pull to the reference's and for asking the
// field.
const pulling = `
kernel void pulls(device const float *places [[buffer(0)]],
                  device float *pulls [[buffer(1)]],
                  constant float *field [[buffer(2)]],
                  uint at [[thread_position_in_grid]]) {
	device const float *place = places + at * perPlace;
	float3 a = pull(float3(place[X], place[Y], place[Z]), field);
	device float *out = pulls + at * perPlace;
	out[X] = a.x; out[Y] = a.y; out[Z] = a.z;
}
`

// stepping is the kernel, after a field's pull. A run is one command's
// worth of steps: their length in units of time, how many to take, and
// how many were taken before, so a pass is dated from the start.
//
// A pass is as orbit has it: how near, how long ago, and the step's
// shares of it in the star's motion and in the pull, taken at the step
// it was found at.
//
// The kernel keeps the motion and the pull at the nearest step it sees
// and works the shares once, at the end, so a step costs what it did
// without them.
//
// The pass so far comes in through the found buffer and goes out the
// same way, so a run of many commands finds one pass. Before the first
// command the pass so far is nowhere, farther than any place, so the
// first command's start is the first pass.
const stepping = `
struct run {
	float dt;
	uint steps;
	uint done;
};

struct passing {
	float near, ago, moved, pulled;
};

struct nearest {
	float near, ago;
	float3 v, a;
};

static inline passing passAt(nearest found, float dt) {
	return passing{found.near, found.ago, length(found.v) * dt / found.near,
	               length(found.a) * dt * dt / found.near};
}

kernel void back(device float *stars [[buffer(0)]],
                 device float *found [[buffer(1)]],
                 constant float *field [[buffer(2)]],
                 constant run &r [[buffer(3)]],
                 uint at [[thread_position_in_grid]]) {
	device float *star = stars + at * perStar;
	device float *pass = found + at * perPass;
	float3 x = float3(star[X], star[Y], star[Z]);
	float3 v = float3(star[along + X], star[along + Y], star[along + Z]);
	passing best = passing{pass[nearAt], pass[agoAt], pass[movedAt],
	                       pass[pulledAt]};
	float3 a = pull(x, field);
	nearest seen = nearest{best.near, best.ago, v, a};
	float d = length(x);
	if (d < seen.near) {
		seen = nearest{d, float(r.done) * r.dt, v, a};
	}
	for (uint k = 0; k < r.steps; k++) {
		v += a * (r.dt / kicks);
		x += v * r.dt;
		a = pull(x, field);
		v += a * (r.dt / kicks);
		d = length(x);
		if (d < seen.near) {
			float ago = float(r.done + k + firstStep) * r.dt;
			seen = nearest{d, ago, v, a};
		}
	}
	if (seen.near < best.near) {
		best = passAt(seen, r.dt);
	}
	star[X] = x.x; star[Y] = x.y; star[Z] = x.z;
	star[along + X] = v.x; star[along + Y] = v.y; star[along + Z] = v.z;
	pass[nearAt] = best.near;
	pass[agoAt] = best.ago;
	pass[movedAt] = best.moved;
	pass[pulledAt] = best.pulled;
}
`

// The buffers as the kernel binds them, each defined by the kernel.
const (
	starsBuffer = iota // every star's place and motion
	foundBuffer        // every star's pass
	fieldBuffer        // the field's numbers
)

// stepsACommand is how many steps one command takes before the batch
// comes back: picked, in the research ledger, and at least one, since
// the ledger holds any number and a command of no steps is no command.
var stepsACommand = commandSteps()

// commandSteps is the ledger's steps a command, held to one at least.
func commandSteps() int {
	picked := int(research.Value("orbit-steps-a-command"))
	return max(one, picked)
}

// The solver's own numbers, each defined.
const (
	one  = 1 // defined: the fewest stars a solver has room for
	none = 0 // defined: a command's first step, modulo a command's steps
)

// Solver is the kernel open on a device over one field, with room for
// so many stars at once.
type Solver struct {
	session *gpu.Session
	field   Field
	stars   gpu.Buffer
	found   gpu.Buffer
	numbers gpu.Buffer
	most    int
	closed  bool
}

// errClosed is a solver used after Close: its buffers are the device's
// no longer, so nothing may be written to them, let alone run.
var errClosed = errors.New("gpu/orbit: the solver is closed")

// Open compiles the kernel over a field on a device, with room for so
// many stars at once. It refuses room for none, and a field whose pull
// does not compile.
func Open(device gpu.Device, field Field, most int) (*Solver, error) {
	if most < one {
		return nil, fmt.Errorf("gpu/orbit: room for %d stars", most)
	}
	source := prelude() + field.source + stepping + pulling
	session, err := gpu.Open(device, source, "back", "pulls")
	if err != nil {
		return nil, err
	}
	numbers := max(one, len(field.numbers))
	made, err := session.Buffers(most*perStar, most*perPass, numbers)
	if err != nil {
		session.Close()
		return nil, err
	}
	copy(made[fieldBuffer].Numbers(), field.numbers)
	return &Solver{
		session: session, field: field,
		stars: made[starsBuffer], found: made[foundBuffer],
		numbers: made[fieldBuffer], most: most,
	}, nil
}

// Field is the field the solver was opened over.
func (solver *Solver) Field() Field { return solver.field }

// Back runs every star's orbit back over a span, in so many steps, and
// reports each one's nearest pass to the centre over them and its state
// at the start of the span.
//
// It refuses a span that is not positive, no steps, more stars than it
// has room for, and a star whose orbit ran away, through the centre or
// past what a float holds.
func (solver *Solver) Back(stars []orbit.State, span measure.Duration,
	steps int,
) ([]orbit.Pass, []orbit.State, error) {
	if solver.closed {
		return nil, nil, errClosed
	}
	dt, err := orbit.Step(span, steps)
	if err != nil {
		return nil, nil, err
	}
	if len(stars) > solver.most {
		return nil, nil, fmt.Errorf("gpu/orbit: %d stars in a solver"+
			" with room for %d", len(stars), solver.most)
	}
	if len(stars) < one {
		return []orbit.Pass{}, []orbit.State{}, nil
	}
	solver.load(stars)
	// every step is visited and a command is dispatched at each one
	// that begins a command: a counted loop over commands, with a
	// mutable step, is a loop one slip makes endless, and this one has
	// no step to slip
	for first := range steps {
		if first%stepsACommand != none {
			continue
		}
		taking := min(stepsACommand, steps-first)
		err := solver.run(len(stars), dt, taking, first)
		if err != nil {
			return nil, nil, err
		}
	}
	return solver.results(len(stars))
}

// load writes the stars to the gpu, their motion reversed to run them
// back, and each one's pass so far as nowhere, for the kernel to find
// the first.
func (solver *Solver) load(stars []orbit.State) {
	in, found := solver.stars.Numbers(), solver.found.Numbers()
	for index, star := range stars {
		x, y, z := star.Numbers().At()
		vx, vy, vz := star.Numbers().Moving()
		place := in[index*perStar:]
		place[axisX] = float32(x)
		place[axisY] = float32(y)
		place[axisZ] = float32(z)
		motion := place[along:]
		motion[axisX] = -float32(vx)
		motion[axisY] = -float32(vy)
		motion[axisZ] = -float32(vz)
		pass := found[index*perPass : (index+one)*perPass]
		clear(pass)
		pass[nearAt] = nowhere
	}
}

// nowhere is a pass not found yet: farther from the centre than any
// place, so the first place the kernel sees is nearer. Defined, as the
// infinity above every float.
var nowhere = float32(math.Inf(above))

// above is math.Inf's sign for the infinity above every number. Defined.
const above = 1

// run is one command: so many steps of the given length for every star,
// after so many already taken.
func (solver *Solver) run(count int, dt float64, steps, done int) error {
	bits := math.Float32bits(float32(dt))
	params := binary.LittleEndian.AppendUint32(nil, bits)
	params = binary.LittleEndian.AppendUint32(params, uint32(steps))
	params = binary.LittleEndian.AppendUint32(params, uint32(done))
	if err := solver.session.Dispatch("back", count, one, params,
		solver.stars, solver.found, solver.numbers); err != nil {
		return err
	}
	return solver.session.Commit()
}

// results reads back each star's pass and its state at the start of the
// span, its motion turned forward again; a star left anywhere that is
// not a number ran away.
func (solver *Solver) results(count int) ([]orbit.Pass, []orbit.State,
	error,
) {
	in, found := solver.stars.Numbers(), solver.found.Numbers()
	passes := make([]orbit.Pass, count)
	states := make([]orbit.State, count)
	var away []int
	for index := range passes {
		place, motion := in[index*perStar:], in[index*perStar+along:]
		state := orbit.StateOf(
			float64(place[axisX]), float64(place[axisY]),
			float64(place[axisZ]), -float64(motion[axisX]),
			-float64(motion[axisY]), -float64(motion[axisZ]))
		if !finite(state) {
			away = append(away, index)
			continue
		}
		pass := found[index*perPass:]
		passes[index] = orbit.NewPass(float64(pass[nearAt]),
			float64(pass[agoAt]), float64(pass[movedAt]),
			float64(pass[pulledAt]))
		states[index] = state
	}
	if len(away) > noStars {
		return nil, nil, NewRunaway(away)
	}
	return passes, states, nil
}

// Runaway is the refusal of a batch in which stars ran away: through
// the centre or past what a float holds, so that a place or a motion
// is not a number. It names the stars by their places in the batch,
// in order.
type Runaway struct{ stars []int }

// NewRunaway is the refusal naming the stars given, as a solver that
// stands in for this one makes it; RanAway reads the stars back.
func NewRunaway(stars []int) Runaway {
	return Runaway{stars: stars}
}

// Error says which stars ran away.
func (away Runaway) Error() string {
	if len(away.stars) == oneStar {
		return fmt.Sprintf("gpu/orbit: star %d ran away",
			away.stars[firstStar])
	}
	return fmt.Sprintf("gpu/orbit: %d stars ran away, the first star %d",
		len(away.stars), away.stars[firstStar])
}

// RanAway is the stars that ran away when a refusal is a Runaway, and
// whether it is one.
func RanAway(err error) ([]int, bool) {
	var away Runaway
	if errors.As(err, &away) {
		return away.stars, true
	}
	return nil, false
}

// The runaways' numbers, each defined.
const (
	noStars   = 0 // defined: no stars ran away
	oneStar   = 1 // defined: one star ran away
	firstStar = 0 // defined: the first of those that did
)

// finite says whether a star's every number is one, and not infinite.
func finite(state orbit.State) bool {
	x, y, z := state.Numbers().At()
	vx, vy, vz := state.Numbers().Moving()
	for _, number := range []float64{x, y, z, vx, vy, vz} {
		if math.IsNaN(number) || math.IsInf(number, either) {
			return false
		}
	}
	return true
}

// either is math.IsInf's sign for an infinity of either sign. Defined.
const either = 0

// Pulls is the field's pull at each place by the solver's kernel, for
// as many places as it has room for stars: a way to hold the kernel's
// pull to the reference's, and to ask the field. It borrows the
// solver's buffers, so a run's results are gone after it.
//
// It refuses a closed solver and more places than it has room for.
func (solver *Solver) Pulls(places []orbit.Place) ([]orbit.Pull, error) {
	if solver.closed {
		return nil, errClosed
	}
	if len(places) > solver.most {
		return nil, fmt.Errorf("gpu/orbit: %d places in a solver with"+
			" room for %d", len(places), solver.most)
	}
	if len(places) < one {
		return []orbit.Pull{}, nil
	}
	in := solver.stars.Numbers()
	for index, place := range places {
		x, y, z := place.Numbers()
		at := in[index*perPlace:]
		at[axisX] = float32(x)
		at[axisY] = float32(y)
		at[axisZ] = float32(z)
	}
	if err := solver.session.Dispatch("pulls", len(places), one, nil,
		solver.stars, solver.found, solver.numbers); err != nil {
		return nil, err
	}
	if err := solver.session.Commit(); err != nil {
		return nil, err
	}
	out := solver.found.Numbers()
	pulls := make([]orbit.Pull, len(places))
	for index := range pulls {
		along := out[index*perPlace:]
		pulls[index] = orbit.NewPull(float64(along[axisX]),
			float64(along[axisY]), float64(along[axisZ]))
	}
	return pulls, nil
}

// GPUTime is how long the gpu has spent on the solver's runs.
func (solver *Solver) GPUTime() time.Duration {
	return solver.session.GPUTime()
}

// Close lets the kernel and its buffers go. Closing twice is harmless.
func (solver *Solver) Close() {
	solver.closed = true
	solver.session.Close()
}
