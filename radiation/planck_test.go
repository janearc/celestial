package radiation_test

import (
	"math"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/radiation"
)

// numbers Planck's tests use, each defined.
const (
	// hemisphere is the solid angle a flat surface's radiance sums to
	// over the sky above it, in steradians. Defined by integrating the
	// cosine over a hemisphere: pi.
	hemisphere = math.Pi
	// stefanPower is the power of temperature in Stefan's law. Defined by
	// the law.
	stefanPower = 4
	// lambdaPower is the power of wavelength in Planck's law. Defined by
	// the law.
	lambdaPower = 5
	// even, odd and thirds are simpson's rule's weights. Defined by the
	// rule.
	even, odd, thirds = 2.0, 4.0, 3.0
	// firstInner is the first point after simpson's rule's start, the
	// first that is weighted. Defined by the rule.
	firstInner = 1
	// whole is one of a thing: one nanometre, or a factor that changes
	// nothing. Defined by counting.
	whole = 1.0
)

// The test and the anti-test: at each point, Planck is CODATA's law, c1L /
// lambda^5 / (exp(c2 / lambda T) - 1), per nanometre. It is not the same law
// per metre, which is a billion times larger.
func TestPlanckIsCODATAsLaw(test *testing.T) {
	law := pickedNumbers(test).Planck
	perNM := radiation.NmToMetres(whole)
	for _, at := range law.Points {
		metres := radiation.NmToMetres(at.NM)
		perMetre := law.C1L / math.Pow(metres, lambdaPower) /
			math.Expm1(law.C2/(metres*at.Kelvin))
		got := radiation.Planck(at.NM, at.Kelvin)
		if !near(got, perMetre*perNM, law.LawTol) {
			test.Errorf("%v nm at %v K: %v, CODATA's law gives %v", at.NM,
				at.Kelvin, got, perMetre*perNM)
		}
		if near(got, perMetre, law.LawTol) {
			test.Errorf("%v nm at %v K is per metre: %v", at.NM, at.Kelvin,
				got)
		}
	}
}

// A black body is brightest at wien's b / T: a part in a thousand to either
// side, it is dimmer.
func TestABlackBodyPeaksWhereWienSays(test *testing.T) {
	law := pickedNumbers(test).Planck
	peak := law.WienB / law.Kelvin / radiation.NmToMetres(whole)
	top := radiation.Planck(peak, law.Kelvin)
	for _, side := range []float64{peak * (whole - law.Either),
		peak * (whole + law.Either)} {
		if lower := radiation.Planck(side, law.Kelvin); !(lower < top) {
			test.Errorf("%v nm is %v, the peak at %v nm only %v", side,
				lower, peak, top)
		}
	}
}

// pi times Planck's radiance, summed over every wavelength, is Stefan's sigma
// T^4.
func TestStefansLawIsPlancksSum(test *testing.T) {
	law := pickedNumbers(test).Planck
	if !(law.Steps > none && law.Steps%twice == none) {
		test.Fatalf("simpson's rule needs an even count, not %d", law.Steps)
	}
	got := hemisphere * planckSum(law.Kelvin, law.FromNM, law.ToNM,
		law.Steps)
	want := law.Sigma * math.Pow(law.Kelvin, stefanPower)
	if !near(got, want, law.StefanTol) {
		test.Errorf("at %v K the sum is %v W m-2, stefan's law %v",
			law.Kelvin, got, want)
	}
}

// planckSum returns the radiance at a temperature summed over wavelength
// between two ends in nanometres, by simpson's rule in even steps of log
// wavelength. in log wavelength, each step carries the radiance times
// the wavelength.
func planckSum(kelvin, fromNM, toNM float64, steps int) float64 {
	start, end := math.Log(fromNM), math.Log(toNM)
	step := (end - start) / float64(steps)
	sum := logStep(start, kelvin) + logStep(end, kelvin)
	for place := firstInner; place < steps; place++ {
		weight := even
		if place%twice != none {
			weight = odd
		}
		sum += weight * logStep(start+float64(place)*step, kelvin)
	}
	return sum * step / thirds
}

// logStep returns the radiance times the wavelength at a log wavelength:
// what one step of log wavelength carries.
func logStep(logNM, kelvin float64) float64 {
	nm := math.Exp(logNM)
	return radiation.Planck(nm, kelvin) * nm
}

// The anti-test: a body at or below zero kelvin gives no light at any
// wavelength, and a temperature that is NaN gives NaN, not a number that
// looks like an answer.
func TestNoLightAtOrBelowZeroKelvin(test *testing.T) {
	law := pickedNumbers(test).Planck
	for _, at := range law.Points {
		for _, kelvin := range law.NoLight {
			if got := radiation.Planck(at.NM, kelvin); got != none {
				test.Errorf("%v nm at %v K gives %v", at.NM, kelvin, got)
			}
		}
		if got := radiation.Planck(at.NM, math.NaN()); !math.IsNaN(got) {
			test.Errorf("%v nm at NaN kelvin gives %v", at.NM, got)
		}
	}
}

// The anti-test, as a property: a hotter body is brighter at every wavelength.
// A law with the temperature on the wrong side of the fraction fails it.
func TestAHotterBodyIsBrighterAtEveryWavelength(test *testing.T) {
	rapid.Check(test, hotter{law: pickedNumbers(test).Planck}.check)
}

// hotter draws two temperatures and a wavelength and compares them.
type hotter struct{ law planckNumbers }

// planckNumbers is the test data's Planck block.
type planckNumbers struct {
	C1L       float64   `json:"c1l"`
	C2        float64   `json:"c2"`
	WienB     float64   `json:"b"`
	Sigma     float64   `json:"sigma"`
	LawTol    float64   `json:"law tolerance"`
	Points    []point   `json:"points"`
	Kelvin    float64   `json:"kelvin"`
	Either    float64   `json:"either side"`
	FromNM    float64   `json:"from nm"`
	ToNM      float64   `json:"to nm"`
	Steps     int       `json:"steps"`
	StefanTol float64   `json:"stefan tolerance"`
	Coldest   float64   `json:"coldest"`
	Hottest   float64   `json:"hottest"`
	Shortest  float64   `json:"shortest nm"`
	Longest   float64   `json:"longest nm"`
	NoLight   []float64 `json:"no light kelvin"`
}

// check is one draw of the property above.
func (warmer hotter) check(test *rapid.T) {
	law := warmer.law
	cool := rapid.Float64Range(law.Coldest, law.Hottest).Draw(test, "cool")
	factor := rapid.Float64Range(whole+law.Either, twice).Draw(test,
		"factor")
	logNM := rapid.Float64Range(math.Log(law.Shortest),
		math.Log(law.Longest)).Draw(test, "log nm")
	nm, warm := math.Exp(logNM), cool*factor
	if !(radiation.Planck(nm, warm) > radiation.Planck(nm, cool)) {
		test.Fatalf("%v nm: %v K gives %v, %v K gives %v", nm, warm,
			radiation.Planck(nm, warm), cool, radiation.Planck(nm, cool))
	}
}
