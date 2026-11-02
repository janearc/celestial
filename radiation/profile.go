package radiation

import (
	"fmt"
	"math"
)

// Profile maps one wavelength interval to another linearly in log wavelength. A
// false-colour profile can use it to place invisible radiation in the visible
// range; libtheme determines the displayed colour.
type Profile struct {
	name           string
	source, render Range
}

// NewProfile constructs a named mapping from source to render range. An invalid
// range returns an error and no profile.
func NewProfile(name string, source, render Range) (*Profile, error) {
	if !valid(source) || !valid(render) {
		return nil, fmt.Errorf("radiation: %s maps %v onto %v, and "+
			"one is no range", name, source, render)
	}
	return &Profile{name: name, source: source, render: render}, nil
}

// String formats the profile for humans with eyeballs.
func (profile Profile) String() string {
	return fmt.Sprintf("%s, %v, shown as %v", profile.name, profile.source,
		profile.render)
}

// Holds checks the half-open source range. The zero Profile contains no
// wavelengths.
func (profile Profile) Holds(nm float64) bool {
	return valid(profile.render) && profile.source.Holds(nm)
}

// Onto maps a source wavelength into the render interval in log space.
// Wavelengths outside the source return ok=false rather than an extrapolated
// answer.
//
// Results remain inside the half-open render range so Back can reverse them. If
// rounding reaches the exclusive long endpoint, Onto uses the preceding float
// instead.
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

// Back reverses Onto, returning ok=false outside the half-open render range or
// when either profile range is invalid.
func (profile Profile) Back(nm float64) (source float64, ok bool) {
	if !valid(profile.source) || !profile.render.Holds(nm) {
		return source, false
	}
	return profile.source.at(profile.render.fraction(nm)), true
}
