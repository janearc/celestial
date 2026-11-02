package hole_test

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/hole"
)

// gpuOf returns the GPU hole of a spec, or stops the test.
func gpuOf(test stopper, from spec) *hole.GPUBlackHole {
	test.Helper()
	gpu, err := hole.NewGPUBlackHole(from.Mass, from.Spin)
	if err != nil {
		test.Fatalf("no GPU hole of %v and %v: %v", from.Mass, from.Spin,
			err)
	}
	return gpu
}

// camera is a GPU hole of a spec with a camera at a place, or the test
// stops.
func camera(test stopper, from spec, at place) *hole.GPUBlackHole {
	test.Helper()
	gpu := gpuOf(test, from)
	if err := gpu.SetCamera(vec3(at)); err != nil {
		test.Fatal(err)
	}
	return gpu
}

// A GPU hole is the hole it was given, its mass and its spin. That a mass
// and a spin no hole has make no GPU hole is held by the anti-test below.
func TestNewGPUBlackHoleMakesTheHoleItWasGiven(t *testing.T) {
	numbers := pickedNumbers(t)
	from := numbers.Holes.Fast
	gpu := gpuOf(t, from)
	black := gpu.Hole()
	if black.Mass() != from.Mass || black.Spin() != from.Spin {
		t.Errorf("the GPU hole has mass %v and spin %v", black.Mass(),
			black.Spin())
	}
}

// The anti-test: a mass and a spin that no hole has make no GPU hole:
// NewGPUBlackHole returns nil and NewHole's error.
func TestNewGPUBlackHoleRefusesWhatNoHoleIs(t *testing.T) {
	for _, refused := range pickedNumbers(t).Refused.Cases {
		gpu, err := hole.NewGPUBlackHole(refused.Mass, refused.Spin)
		if err == nil {
			t.Errorf("%s: not refused: %v", refused.Why, err)
		}
		if gpu != nil {
			t.Errorf("%s: made a GPU hole", refused.Why)
		}
	}
}

// The anti-test: a camera can be held still outside the ergosurface and
// nowhere inside it. SetCamera refuses a point inside, says why, and
// leaves the camera it had.
func TestSetCameraRefusesWhereNothingIsStill(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := camera(t, numbers.Holes.Fast, numbers.Places.Eye)
	before := gpu.ShaderParams()
	err := gpu.SetCamera(vec3(numbers.Frame.Ergosphere))
	if !errors.Is(err, hole.ErrNothingStill()) {
		t.Fatalf("a camera inside the ergosurface gave %v", err)
	}
	after := gpu.ShaderParams()
	if string(after.Bytes()) != string(before.Bytes()) {
		t.Errorf("a refused camera moved the camera")
	}
}

// A camera outside the ergosurface is accepted without a word.
func TestSetCameraAcceptsWhereSomethingIsStill(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := gpuOf(t, numbers.Holes.Fast)
	if err := gpu.SetCamera(vec3(numbers.Places.Eye)); err != nil {
		t.Errorf("a camera far out was refused: %v", err)
	}
}

// The anti-test: before a camera is set there is no camera to shift against,
// and the redshift is not a number: not a zero that reads as a shift.
func TestRedshiftBeforeACameraIsNotANumber(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := gpuOf(t, numbers.Holes.Still)
	black := gpu.Hole()
	ray := sent(&black, numbers.Places.Eye, numbers.Frame.Aim)
	report := gpu.StepRay(&ray, numbers.Geodesic.Step)
	if !math.IsNaN(report.Redshift()) {
		t.Errorf("with no camera the redshift is %v", report.Redshift())
	}
}

// The redshift StepRay reports is a still camera's energy over a still
// emitter's where the ray now is: the ratio of the two clocks' rates.
func TestRedshiftBetweenStillObservers(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := camera(t, numbers.Holes.Still, numbers.Places.Eye)
	black := gpu.Hole()
	ray := sent(&black, numbers.Places.Eye, numbers.Frame.Aim)
	var report hole.RayStep
	for range numbers.GPU.Steps {
		report = gpu.StepRay(&ray, numbers.GPU.Step)
	}
	eye := vec3(numbers.Places.Eye)
	want := redshiftRoot(&black, ray.Space()) / redshiftRoot(&black, eye)
	if !within(report.Redshift(), want, numbers.Frame.Tolerance) {
		t.Errorf("redshift %v from radius %v, want %v", report.Redshift(),
			report.Radius(), want)
	}
	if !(report.Radius() <= numbers.GPU.Reached) {
		t.Errorf("the ray only got to radius %v", report.Radius())
	}
}

// StepRay steps the ray in place as Step does, and reports what the ray
// keeps there as the hole reckons it.
func TestStepRayStepsInPlaceAndReports(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := camera(t, numbers.Holes.Kerr, numbers.Places.Eye)
	black := gpu.Hole()
	ray := sent(&black, numbers.Places.Eye, numbers.Frame.Aim)
	want := black.Step(ray, numbers.GPU.Step)
	report := gpu.StepRay(&ray, numbers.GPU.Step)
	if ray != want || report.Position() != want.Position() ||
		report.Momentum() != want.Momentum() {
		t.Errorf("the ray went to %v, not %v", ray.Position(),
			want.Position())
	}
	radius := black.Radius(want.Space())
	for _, pair := range []kept{
		{"energy", black.Energy(want), report.Energy()},
		{"angular momentum", black.Lz(want), report.AngularMomentum()},
		{"carter constant", black.Carter(want), report.CarterConstant()},
		{"hamiltonian", black.Hamiltonian(want), report.Hamiltonian()},
		{"radius", radius, report.Radius()},
	} {
		if pair.before != pair.after {
			t.Errorf("%s: reported %v, the hole says %v", pair.name,
				pair.after, pair.before)
		}
	}
}

// A batch steps each ray as StepRay does, in the order given, and fills the
// slice it is handed when that is long enough, and a new one when it is not.
func TestStepRaysBatchStepsEachRay(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := camera(t, numbers.Holes.Kerr, numbers.Places.Eye)
	black := gpu.Hole()
	first := sent(&black, numbers.Places.Eye, numbers.Frame.Aim)
	second := sent(&black, numbers.Places.Eye, numbers.Frame.Outward)
	step := numbers.GPU.Step
	singles := []hole.Ray{first, second}
	var wants []hole.RayStep
	for index := range singles {
		wants = append(wants, gpu.StepRay(&singles[index], step))
	}
	rays := []hole.Ray{first, second}
	reports := gpu.StepRaysBatch(nil, rays, step)
	if len(reports) != len(rays) {
		t.Fatalf("the batch gave %d reports for %d rays", len(reports),
			len(rays))
	}
	for index := range rays {
		if rays[index] != singles[index] || reports[index] != wants[index] {
			t.Errorf("ray %d went to %v, not %v", index,
				rays[index].Position(), singles[index].Position())
		}
	}
}

// The test and the anti-test: a slice long enough is filled and handed
// back; one too short is left alone, what it held still there, and a new
// one is made. The short slice holds a real report as its marker, since
// a report's parts are the package's own; the rays have stepped on since,
// so a report written over it would differ.
func TestStepRaysBatchFillsTheSliceItIsGiven(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := camera(t, numbers.Holes.Kerr, numbers.Places.Eye)
	black := gpu.Hole()
	ray := sent(&black, numbers.Places.Eye, numbers.Frame.Aim)
	step := numbers.GPU.Step
	roomy := make([]hole.RayStep, numbers.GPU.Count)
	rays := make([]hole.Ray, numbers.GPU.Count)
	for index := range rays {
		rays[index] = ray
	}
	filled := gpu.StepRaysBatch(roomy, rays, step)
	for index := range roomy {
		if roomy[index] != filled[index] || roomy[index] == (hole.RayStep{}) {
			t.Errorf("report %d was not written into the slice given", index)
		}
	}
	marker := filled[none]
	short := []hole.RayStep{marker}
	again := gpu.StepRaysBatch(short, rays, step)
	if short[none] != marker {
		t.Error("a slice too short for the rays was written into")
	}
	if len(again) != len(rays) || &again[none] == &short[none] {
		t.Errorf("%d reports came back, in the short slice: %v", len(again),
			&again[none] == &short[none])
	}
}

// The anti-test: a batch gives one report to a ray and no more. A slice
// longer than the rays is handed back cut to their number, not whole.
func TestStepRaysBatchGivesNoReportsBeyondTheRays(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := camera(t, numbers.Holes.Kerr, numbers.Places.Eye)
	black := gpu.Hole()
	ray := sent(&black, numbers.Places.Eye, numbers.Frame.Aim)
	rays := make([]hole.Ray, numbers.GPU.Count)
	for index := range rays {
		rays[index] = ray
	}
	spare := make([]hole.RayStep, numbers.GPU.Count+numbers.Anti.SpareReports)
	reports := gpu.StepRaysBatch(spare, rays, numbers.GPU.Step)
	if len(reports) != len(rays) {
		t.Errorf("%d rays were given %d reports", len(rays), len(reports))
	}
}

// The anti-test: a camera is refused where nothing is still, whatever the
// hole: inside the horizon of a hole with no spin too, and the camera it
// had is kept.
func TestSetCameraRefusesInsideTheHorizon(t *testing.T) {
	numbers := pickedNumbers(t)
	gpu := camera(t, numbers.Holes.Still, numbers.Places.Eye)
	before := gpu.ShaderParams()
	err := gpu.SetCamera(vec3(numbers.Frame.InsideHorizon))
	if !errors.Is(err, hole.ErrNothingStill()) {
		t.Fatalf("a camera inside the horizon gave %v", err)
	}
	after := gpu.ShaderParams()
	if string(after.Bytes()) != string(before.Bytes()) {
		t.Errorf("a refused camera moved the camera")
	}
}
