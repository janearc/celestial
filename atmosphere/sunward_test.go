//ninja:vow dojo the air between a point and the sun is held to the air along a way rising as the sun does, through sunward and columnOf, which atmosphere does not export
package atmosphere

import "testing"

// wayOf is the way of three parts, or the test stops.
func wayOf(test *testing.T, x, y, z float64) Way {
	test.Helper()
	made, err := NewWay(x, y, z)
	if err != nil {
		test.Fatal(err)
	}
	return *made
}

// From the eye, the air between it and the sun is the air along a way
// rising as the sun rises: the sun's own sine, not one worked out
// otherwise. The sun here is off both x and z, about 35 degrees up.
func TestTheAirToTheSunRisesAsTheSunDoes(test *testing.T) {
	known := measured()
	up := wayOf(test, none, whole, none)
	sun := wayOf(test, whole, whole, whole)
	got, lit := sunward(known.meanRadius, up, sun, none, known.sunPieces)
	want := columnOf(known.meanRadius, sun.y, known.sunPieces).molecules
	if !lit || got != want {
		test.Errorf("to the sun the air is %v, lit %v; along its way %v",
			got, lit, want)
	}
}
