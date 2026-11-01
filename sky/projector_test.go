package sky_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/mocks"
	"github.com/janearc/celestial/sky"
)

// farthestEast is a caller's choice the mock stands in for: how far east
// of the centre the easternmost of some directions lands. It is the shape
// a picture or a fit takes a Projector in.
func farthestEast(plane sky.Projector, directions []sky.Equatorial) (
	*measure.Angle, error) {
	var most *measure.Angle
	for _, at := range directions {
		offset, err := plane.Onto(at)
		if err != nil {
			return nil, err
		}
		if east := offset.East(); most == nil ||
			east.InRadians() > most.InRadians() {
			most = &east
		}
	}
	return most, nil
}

// twice is a count of two calls. Defined by counting.
const twice = 2

// A caller that takes a Projector is tested with the mock: it asks for
// each direction once, and keeps the easternmost. The caller is written
// here, as no caller in celestial takes a Projector yet, so this shows
// only that the generated mock fits that shape.
func TestACallerCanStandTheProjectorMockIn(test *testing.T) {
	numbers := planeNumbers(test)
	control := gomock.NewController(test)
	plane := mocks.NewMockProjector(control)
	near := toward(test, numbers.OneDegree, nought)
	far := toward(test, numbers.Quarter, nought)
	east := sky.NewOffset(measure.Degrees(numbers.OneDegree),
		measure.Radians(nought))
	plane.EXPECT().Onto(gomock.Any()).Return(&east, nil).Times(twice)
	most, err := farthestEast(plane, []sky.Equatorial{near, far})
	if err != nil || most == nil || *most != east.East() {
		test.Errorf("the easternmost is %v, %v", most, err)
	}
}

// The anti-test: what the projector refuses, the caller refuses, and it
// asks no further.
func TestACallerPassesOnTheProjectorsRefusal(test *testing.T) {
	numbers := planeNumbers(test)
	control := gomock.NewController(test)
	plane := mocks.NewMockProjector(control)
	behind := toward(test, numbers.Quarter, nought)
	refusal := errors.New("sky: a quarter turn or more from the centre")
	plane.EXPECT().Onto(behind).Return(nil, refusal).Times(once)
	_, err := farthestEast(plane, []sky.Equatorial{behind, behind})
	if !errors.Is(err, refusal) {
		test.Errorf("the caller gave %v", err)
	}
}

// once is a count of one call. Defined by counting.
const once = 1
