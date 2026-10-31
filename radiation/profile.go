package radiation

import (
	"fmt"
	"math"
)

// Profile maps one range of wavelengths onto another, evenly in the
// logarithm of wavelength. A false colour profile is made of one: it
// takes radiation the eye cannot see onto light it can.
//
// What colour that is belongs to libtheme; here the profile only says
// where a wavelength lands.
type Profile struct {
	name           string
	source, render Range
}

// NewProfile is the profile of a name that maps a source range onto a
// render range. It refuses either range when it is no range, and then
// gives no profile.
func NewProfile(name string, source, render Range) (*Profile, error) {
	if !valid(source) || !valid(render) {
		return nil, fmt.Errorf("%w: %s maps %v onto %v", errRange, name,
			source, render)
	}
	return &Profile{name: name, source: source, render: render}, nil
}

// String is the profile, for humans with eyeballs.
func (profile Profile) String() string {
	return fmt.Sprintf("%s, %v, shown as %v", profile.name, profile.source,
		profile.render)
}

// Holds is whether a wavelength is inside the profile's source, from its
// short end up to but not including its long end. The zero Profile,
// whose ranges are none, holds nothing.
func (profile Profile) Holds(nm float64) bool {
	return valid(profile.render) && profile.source.Holds(nm)
}

// Onto is where a wavelength in the source lands in the render, evenly
// in the logarithm of wavelength. ok is false, and the wavelength is not
// mapped, when the profile does not hold it: the answer would be a
// number, but not one the profile means.
//
// Onto's answer is always inside the render, short end in and long end
// out, so Back takes back whatever Onto gives. A wavelength a rounding
// below the source's long end can work out to the render's long end
// exactly; Onto gives the float just below it instead.
func (profile Profile) Onto(nm float64) (mapped float64, ok bool) {
	if !profile.Holds(nm) {
		return mapped, false
	}
	mapped = profile.render.at(profile.source.fraction(nm))
	if !profile.render.Holds(mapped) {
		short, long := profile.render.Edges()
		mapped = math.Nextafter(long, short)
	}
	return mapped, true
}

// Back is the reciprocal of Onto: where a wavelength in the render came
// from in the source. ok is false, and the wavelength is not mapped,
// when it is outside the render (its long end is outside, as the
// source's is) or the profile's ranges are none.
func (profile Profile) Back(nm float64) (source float64, ok bool) {
	if !valid(profile.source) || !profile.render.Holds(nm) {
		return source, false
	}
	return profile.source.at(profile.render.fraction(nm)), true
}
