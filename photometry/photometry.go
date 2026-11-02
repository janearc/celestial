package photometry

import (
	"maps"
	"slices"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/celestial/research"
)

// Photometer converts a magnitude and its uncertainty in a named band to
// spectral flux density.
type Photometer interface {
	Flux(band string, magnitude, magnitudeError float64) (*Flux, error)
}

// Survey must stay a Photometer: the build fails here if it stops being
// one.
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

// Survey stores named bands with their magnitude-zero flux densities.
type Survey struct {
	name  string
	zeros map[string]float64
}

// wiseBands is the name of each of WISE's bands, and the name in celestial's
// research data of its zero point, in janskys.
var wiseBands = map[string]string{
	"W1": "wise-w1-zero-jy", "W2": "wise-w2-zero-jy",
	"W3": "wise-w3-zero-jy", "W4": "wise-w4-zero-jy",
}

// WISE returns the Wide-field Infrared Survey Explorer’s four bands.
// Vega-magnitude zero points come from its All-Sky Explanatory Supplement
// through the research data.
func WISE() *Survey {
	zeros := make(map[string]float64, len(wiseBands))
	for band, name := range wiseBands {
		zeros[band] = radiation.JanskyToSI(research.Value(name))
	}
	return &Survey{name: "wise", zeros: zeros}
}

// Bands returns the survey’s band names in order.
func (survey *Survey) Bands() []string {
	return slices.Sorted(maps.Keys(survey.zeros))
}

// String returns the survey name.
func (survey *Survey) String() string { return survey.name }
