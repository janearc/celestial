//go:build darwin

package orbit_test

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"

	gpuorbit "github.com/janearc/celestial/gpu/orbit"
	"github.com/janearc/celestial/orbit"
	"github.com/janearc/celestial/orbit/orbittest"
)

// galaxy is the published model with Sgr A* at the centre, as a field
// in Metal, or the test stops.
func galaxy(test *testing.T, numbers picked) gpuorbit.Field {
	test.Helper()
	made, err := orbit.NewMilkyWay(orbit.MWPotential2014(),
		orbittest.Centre(numbers.Galaxy.HoleSuns))
	if err != nil {
		test.Fatal(err)
	}
	return gpuorbit.MilkyWay(*made)
}

// the property: the kernel agrees with the reference on any batch of
// stars in the galaxy's field, as it does in the point mass's, and a
// run that compared too few or found one near the edge of its room
// says so.
func TestTheKernelAgreesWithTheReferenceInTheGalaxy(test *testing.T) {
	numbers := pickedNumbers(test)
	agreeing(test, numbers, galaxy(test, numbers))
}

// parity is the kernel's pull against the reference's, under a test:
// the worst pull as a share of its room.
type parity struct {
	test    *testing.T
	numbers picked
	field   gpuorbit.Field
	made    *gpuorbit.Solver
	worst   float64
}

// the property: the kernel's pull is the reference's at any place, in
// the point mass's field and in the galaxy's, within a share of it; and
// a run that found one near the edge of its room says so.
func TestTheKernelsPullIsTheReferences(test *testing.T) {
	numbers := pickedNumbers(test)
	fields := map[string]gpuorbit.Field{"point mass": centre(numbers),
		"galaxy": galaxy(test, numbers)}
	for name, field := range fields {
		holds := &parity{test: test, numbers: numbers, field: field,
			made: solver(test, field, numbers.Room)}
		rapid.Check(test, holds.atAnyPlaces)
		test.Logf("%s: the worst pull was %.2g of its room", name,
			holds.worst)
		if !(holds.worst <= numbers.Agree.Worst) {
			test.Errorf("%s: the worst pull was %.2g of its room, past %g",
				name, holds.worst, numbers.Agree.Worst)
		}
	}
}

// atAnyPlaces is the parity on one drawn batch of places.
func (holds *parity) atAnyPlaces(draw *rapid.T) {
	drawn := holds.numbers.Drawn
	count := rapid.IntRange(drawn.FewestStars, holds.numbers.Room).Draw(
		draw, "count")
	places := make([]orbit.Place, count)
	for index := range places {
		x, y, z := drawnTriple(draw, drawn.Farthest, "at")
		if orbittest.Size(x, y, z) < drawn.Nearest {
			x += math.Copysign(drawn.Nearest, x)
		}
		places[index] = orbit.PlaceOf(x, y, z)
	}
	pulls, err := holds.made.Pulls(places)
	if err != nil {
		draw.Fatal(err)
	}
	for index, place := range places {
		gx, gy, gz := pulls[index].Along()
		wx, wy, wz := holds.field.Reference().Pull(place).Along()
		off := orbittest.Size(gx-wx, gy-wy, gz-wz)
		room := holds.numbers.Parity.Room * orbittest.Size(wx, wy, wz)
		holds.worst = math.Max(holds.worst, off/room)
		if off > room {
			draw.Errorf("at %v the kernel pulls %v, the reference %v",
				place, pulls[index], holds.field.Reference().Pull(place))
		}
	}
}

// the anti-test: a closed solver refuses to pull, and an open one
// refuses more places than it has room for; no places is no work.
func TestPullsRefuseWhatBackRefuses(test *testing.T) {
	numbers := pickedNumbers(test)
	made := solver(test, centre(numbers), numbers.One)
	if pulls, err := made.Pulls(nil); err != nil ||
		len(pulls) != int(numbers.None) {
		test.Errorf("no places gave %v, %v", pulls, err)
	}
	places := []orbit.Place{orbit.PlaceOf(numbers.Line.AlongX,
		numbers.Line.AlongY, numbers.Line.AlongZ)}
	if _, err := made.Pulls(append(places, places...)); err == nil ||
		!strings.Contains(err.Error(), "room") {
		test.Errorf("two places in a solver with room for one gave %v", err)
	}
	made.Close()
	if _, err := made.Pulls(places); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		test.Errorf("a closed solver pulled: %v", err)
	}
}

// the anti-test: a closed solver writes no place into its buffers and
// asks nothing of its device: the refusal comes first, not after a
// place has been written where the device no longer is.
func TestAClosedSolverPullsNothing(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test, numbers)
	held := &kept{numbers: numbers}
	device.EXPECT().Buffer(gomock.Any()).DoAndReturn(held.buffer).AnyTimes()
	made, err := gpuorbit.Open(device, centre(numbers), numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	made.Close()
	places := []orbit.Place{orbit.PlaceOf(numbers.Line.AlongX,
		numbers.Line.AlongY, numbers.Line.AlongZ)}
	if _, err := made.Pulls(places); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		test.Errorf("a closed solver pulled: %v", err)
	}
	stars, found := held.made[int(numbers.None)], held.made[numbers.One]
	for at, value := range append(stars, found...) {
		if value != uint8(numbers.None) {
			test.Fatalf("byte %d of the buffers is %d after a closed pull",
				at, value)
		}
	}
}

// the anti-test: no places ask nothing of the device.
func TestNoPlacesAskNothingOfTheDevice(test *testing.T) {
	numbers := pickedNumbers(test)
	made, err := gpuorbit.Open(ready(test, numbers), centre(numbers),
		numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	if pulls, err := made.Pulls(nil); err != nil ||
		len(pulls) != int(numbers.None) {
		test.Errorf("no places gave %v, %v", pulls, err)
	}
}

// floatAt is the float so many floats into a buffer's bytes.
func floatAt(buffer []byte, index int, bytes int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(
		buffer[index*bytes:]))
}

// each place is written to the device in the layout's numbers, one
// after another: the second place begins where the first's three end.
func TestEachPlaceIsWrittenInThreeNumbers(test *testing.T) {
	numbers := pickedNumbers(test)
	device := opened(test, numbers)
	held := &kept{numbers: numbers}
	device.EXPECT().Buffer(gomock.Any()).DoAndReturn(held.buffer).AnyTimes()
	device.EXPECT().Dispatch(gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any()).Return("")
	device.EXPECT().Commit().Return("")
	made, err := gpuorbit.Open(device, centre(numbers), numbers.Two)
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	line := numbers.Line
	first := orbit.PlaceOf(line.AlongX, line.AlongY, line.AlongZ)
	second := orbit.PlaceOf(line.AlongY, line.AlongZ, line.AlongX)
	if _, err := made.Pulls([]orbit.Place{first, second}); err != nil {
		test.Fatal(err)
	}
	layout := numbers.Layout
	stars := held.made[int(numbers.None)]
	want := []float64{line.AlongX, line.AlongY, line.AlongZ, line.AlongY,
		line.AlongZ, line.AlongX}
	for index, number := range want {
		if got := floatAt(stars, index, layout.Bytes); got != float32(number) {
			test.Errorf("number %d written to the device is %g, want %g",
				index, got, number)
		}
	}
	if layout.Place*numbers.Two != len(want) {
		test.Errorf("two places are %d numbers in the layout, not %d",
			layout.Place*numbers.Two, len(want))
	}
}

// the anti-test: a dispatch the device refuses stops the pulls before
// their commit, and a pull the device stops is said.
func TestPullsTheDeviceRefusesOrStopsAreSaid(test *testing.T) {
	numbers := pickedNumbers(test)
	places := []orbit.Place{orbit.PlaceOf(numbers.Line.AlongX,
		numbers.Line.AlongY, numbers.Line.AlongZ)}
	refusing := ready(test, numbers)
	refusing.EXPECT().Dispatch(gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any()).Return("no such kernel")
	made, err := gpuorbit.Open(refusing, centre(numbers), numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	defer made.Close()
	if _, err := made.Pulls(places); err == nil {
		test.Error("a dispatch the device refused went on to a commit")
	}
	stopping := ready(test, numbers)
	stopping.EXPECT().Dispatch(gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any()).Return("")
	stopping.EXPECT().Commit().Return("run: the command was stopped")
	stopped, err := gpuorbit.Open(stopping, centre(numbers), numbers.One)
	if err != nil {
		test.Fatal(err)
	}
	defer stopped.Close()
	if _, err := stopped.Pulls(places); err == nil {
		test.Error("a pull the device stopped went on")
	}
}
