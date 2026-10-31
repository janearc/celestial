//ninja:vow dojo the sky's own sums are held against ten times as many, through skylight, which atmosphere does not export
package atmosphere

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
)

// fine is how the fineness test picks its look, from
// testdata/atmosphere.json.
type fine struct {
	CO2  float64 `json:"co2 ppm"`
	Fine struct {
		Low       float64 `json:"low degrees"`
		SunUp     float64 `json:"sun up degrees"`
		Eye       float64 `json:"eye metres"`
		Finer     int     `json:"finer"`
		Tolerance float64 `json:"tolerance"`
	} `json:"fine"`
	Blue  float64 `json:"blue nm"`
	Green float64 `json:"green nm"`
	Red   float64 `json:"red nm"`
}

// The sky's sums are fine enough: low over the horizon at moonrise, with
// the sun just up behind, the light summed in the sky's own steps and
// pieces is within a percent of the light summed in ten times as many.
func TestTheSkysSumsAreFineEnough(test *testing.T) {
	var numbers fine
	raw, err := os.ReadFile("testdata/atmosphere.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	low := measure.Degrees(numbers.Fine.Low).InRadians()
	sunUp := measure.Degrees(numbers.Fine.SunUp).InRadians()
	view := Way{y: math.Sin(low), z: math.Cos(low)}
	sun := Way{y: math.Sin(sunUp), z: -math.Cos(sunUp)}
	air, err := DryAir(numbers.CO2)
	if err != nil {
		test.Fatal(err)
	}
	var areas []float64
	for _, nm := range []float64{numbers.Blue, numbers.Green, numbers.Red} {
		wavelength := measure.Metres(si.Nano(nm))
		area, err := air.CrossSection(wavelength)
		if err != nil {
			test.Fatal(err)
		}
		areas = append(areas, area.InSquareMetres())
	}
	known, finer := measured(), numbers.Fine.Finer
	from := known.meanRadius + numbers.Fine.Eye
	coarse := skylight(from, view, sun, areas, known.skySteps,
		known.sunPieces)
	fine := skylight(from, view, sun, areas, finer*known.skySteps,
		finer*known.sunPieces)
	for band := range areas {
		if !(math.Abs(coarse[band]/fine[band]-whole) <= numbers.Fine.Tolerance) {
			test.Errorf("at %d the sky sums to %.6g, and finely %.6g",
				band, coarse[band], fine[band])
		}
	}
}
