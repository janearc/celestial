package gw_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/janearc/celestial/gw"
	"github.com/janearc/celestial/mocks"
)

// massRatio is a caller's choice the mock stands in for: the lighter of
// a merger's two masses over the heavier. It is the shape a caller takes
// an Estimator in.
func massRatio(event gw.Estimator) (float64, error) {
	first, err := event.Estimate("mass_1_source")
	if err != nil {
		return nothing, err
	}
	second, err := event.Estimate("mass_2_source")
	if err != nil {
		return nothing, err
	}
	return min(first.Value(), second.Value()) /
		max(first.Value(), second.Value()), nil
}

// nothing is no ratio at all. Defined by the scale.
const nothing = 0.0

// A caller that takes an Estimator is tested with the mock: it asks for
// each mass once, and its ratio is the lighter over the heavier. The
// estimates the mock gives are GW150914's own.
//
// The caller is written here, as no caller in celestial takes an
// Estimator yet, so this shows only that the generated mock fits that
// shape.
func TestACallerCanStandTheEstimatorMockIn(test *testing.T) {
	event := gw150914(test)
	heavier := estimate(test, event, "mass_1_source")
	lighter := estimate(test, event, "mass_2_source")
	control := gomock.NewController(test)
	mock := mocks.NewMockEstimator(control)
	mock.EXPECT().Estimate("mass_1_source").Return(heavier, nil).
		Times(single)
	mock.EXPECT().Estimate("mass_2_source").Return(lighter, nil).
		Times(single)
	ratio, err := massRatio(mock)
	want := lighter.Value() / heavier.Value()
	if err != nil || !near(ratio, want, pickedNumbers(test).Tolerance) {
		test.Errorf("the mass ratio is %v, %v; not %v", ratio, err, want)
	}
}

// The anti-test: an estimate the event does not have is refused, and the
// caller asks no further.
func TestACallerPassesOnAnAbsentEstimate(test *testing.T) {
	control := gomock.NewController(test)
	mock := mocks.NewMockEstimator(control)
	absent := errors.New(`gw: GW150914-v3 has no estimate of ` +
		`"mass_1_source"`)
	mock.EXPECT().Estimate("mass_1_source").Return(nil, absent).
		Times(single)
	if _, err := massRatio(mock); !errors.Is(err, absent) {
		test.Errorf("the caller gave %v", err)
	}
}
