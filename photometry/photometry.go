package photometry

import (
	"maps"
	"slices"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/celestial/research"
)

// Photometer turns a magnitude measured in a named band, and its error,
// into the flux density it stands for.
type Photometer interface {
	Flux(band string, magnitude, magnitudeError float64) (*Flux, error)
}

// A Survey is a Photometer: the build fails if it stops being one.
var _ Photometer = (*Survey)(nil)

// Pogson's scale, as defined: five magnitudes are a factor of a hundred.
const (
	// pogsonFactor is the factor in flux of pogsonStep magnitudes.
	// Defined by Pogson (1856): a hundred.
	pogsonFactor = 100
	// pogsonStep is how many magnitudes make pogsonFactor. Defined by
	// Pogson (1856): five.
	pogsonStep = 5
)

// Survey is a survey's bands, each known by the name the survey writes
// and holding the flux density of a magnitude of zero in it.
type Survey struct {
	name  string
	zeros map[string]float64
}

// wiseBands is the name of each of wise's bands, and the name in
// celestial's research data of its zero point, in janskys.
var wiseBands = map[string]string{
	"W1": "wise-w1-zero-jy", "W2": "wise-w2-zero-jy",
	"W3": "wise-w3-zero-jy", "W4": "wise-w4-zero-jy",
}

// WISE is the wide-field infrared survey explorer's four bands, with the
// zero points of its vega magnitudes from its all-sky explanatory
// supplement, as celestial's research data holds them.
func WISE() *Survey {
	zeros := make(map[string]float64, len(wiseBands))
	for band, name := range wiseBands {
		zeros[band] = radiation.JanskyToSI(research.Value(name))
	}
	return &Survey{name: "wise", zeros: zeros}
}

// Bands is the names of the survey's bands, in order.
func (survey *Survey) Bands() []string {
	return slices.Sorted(maps.Keys(survey.zeros))
}

// String is the survey's name.
func (survey *Survey) String() string { return survey.name }
