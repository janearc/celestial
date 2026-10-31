package photometry_test

import (
	"encoding/json"
	"errors"
	"maps"
	"math"
	"os"
	"slices"
	"testing"

	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"

	"github.com/janearc/celestial/mocks"
	"github.com/janearc/celestial/photometry"
)

// span is a range numbers are drawn from.
type span struct {
	Least float64 `json:"least"`
	Most  float64 `json:"most"`
}

// picked is the numbers these tests use, from testdata/photometry.json.
type picked struct {
	Zeros      map[string]float64 `json:"zeros jy"`
	Five       float64            `json:"five"`
	Hundred    float64            `json:"hundred"`
	Tenth      float64            `json:"tenth"`
	PerMag     float64            `json:"error per magnitude"`
	Tolerance  float64            `json:"tolerance"`
	TooBright  float64            `json:"too bright"`
	NotBands   []string           `json:"not bands"`
	NotErrors  []float64          `json:"not errors"`
	Magnitudes span               `json:"magnitudes"`
	Errors     span               `json:"errors"`
	Steady     []float64          `json:"steady errors"`
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/photometry.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// near is whether a value lands within a share of what is expected.
func near(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Abs(want)
}

// The numbers these tests compare with or ask for.
const (
	nothing  = 0.0 // defined by the scale: no magnitude and no error
	upward   = +1  // defined by math.Inf: the endless above
	downward = -1  // defined by math.Inf: the endless below
	once     = 1   // defined by counting: one call
)

// The survey's bands are wise's four, by the names wise writes.
func TestWISEHasItsFourBands(test *testing.T) {
	zeros := pickedNumbers(test).Zeros
	want := slices.Sorted(maps.Keys(zeros))
	survey := photometry.WISE()
	if got := survey.Bands(); !slices.Equal(got, want) {
		test.Errorf("%s has bands %v, not %v", survey, got, want)
	}
	if name := survey.String(); name != "wise" {
		test.Errorf("wise is called %q", name)
	}
}

// A magnitude of zero is the band's zero point, in janskys as the
// supplement gives it.
func TestMagnitudeZeroIsTheZeroPoint(test *testing.T) {
	numbers := pickedNumbers(test)
	survey := photometry.WISE()
	for band, zero := range numbers.Zeros {
		flux, err := survey.Flux(band, nothing, nothing)
		if err != nil {
			test.Fatal(err)
		}
		if janskys, sigma := flux.InJanskys(); !near(janskys, zero,
			numbers.Tolerance) || sigma != nothing {
			test.Errorf("%s at magnitude zero is %v Jy, error %v", band,
				janskys, sigma)
		}
	}
}

// The anti-test: a band the survey does not have is refused, and gives
// no flux, whether it is a band of another survey, a misspelling or
// nothing.
func TestABandTheSurveyHasNotIsRefused(test *testing.T) {
	survey := photometry.WISE()
	for _, band := range pickedNumbers(test).NotBands {
		flux, err := survey.Flux(band, nothing, nothing)
		if !errors.Is(err, photometry.ErrBand()) || flux != nil {
			test.Errorf("%q gave %v, %v", band, flux, err)
		}
	}
}

// Five magnitudes fainter is a hundredth of the flux, and the error of a
// tenth of a magnitude is a tenth of the natural log of a hundred over
// five, of the flux.
func TestFiveMagnitudesAreAHundred(test *testing.T) {
	numbers := pickedNumbers(test)
	survey := photometry.WISE()
	bright, err := survey.Flux("W1", nothing, numbers.Tenth)
	if err != nil {
		test.Fatal(err)
	}
	faint, err := survey.Flux("W1", numbers.Five, numbers.Tenth)
	if err != nil {
		test.Fatal(err)
	}
	brightSI, brightSigma := bright.InSI()
	faintSI, _ := faint.InSI()
	if !near(brightSI/faintSI, numbers.Hundred, numbers.Tolerance) {
		test.Errorf("five magnitudes is a factor of %v", brightSI/faintSI)
	}
	want := brightSI * numbers.PerMag * numbers.Tenth
	if !near(brightSigma, want, numbers.Tolerance) {
		test.Errorf("a tenth of a magnitude's error is %v, not %v",
			brightSigma, want)
	}
}

// The anti-test: a magnitude's error widens its flux's error and never
// moves its flux. The same magnitude with errors from none to a whole
// magnitude gives the same flux; a Flux that read the error into the
// magnitude would pass the five-magnitude test, whose two errors are the
// same and cancel, and fail this one.
func TestAnErrorNeverMovesTheFlux(test *testing.T) {
	numbers := pickedNumbers(test)
	if len(numbers.Steady) == noErrors {
		test.Fatal("no errors to hold")
	}
	survey := photometry.WISE()
	bare, err := survey.Flux("W1", numbers.Five, nothing)
	if err != nil {
		test.Fatal(err)
	}
	want, _ := bare.InSI()
	for _, sigma := range numbers.Steady {
		flux, err := survey.Flux("W1", numbers.Five, sigma)
		if err != nil {
			test.Fatal(err)
		}
		if got, _ := flux.InSI(); got != want {
			test.Errorf("an error of %v moves the flux to %v from %v",
				sigma, got, want)
		}
	}
}

// noErrors is test data with no errors in it. Defined by counting.
const noErrors = 0

// The anti-test: a magnitude that is not a number or is endless, and an
// error below nothing, not a number or endless, are refused and give no
// flux.
func TestWhatIsNotAMagnitudeIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	survey := photometry.WISE()
	nonsense := []float64{math.NaN(), math.Inf(upward), math.Inf(downward)}
	for _, magnitude := range nonsense {
		flux, err := survey.Flux("W2", magnitude, numbers.Tenth)
		if !errors.Is(err, photometry.ErrMagnitude()) || flux != nil {
			test.Errorf("magnitude %v gave %v, %v", magnitude, flux, err)
		}
	}
	for _, sigma := range append(numbers.NotErrors, nonsense...) {
		flux, err := survey.Flux("W2", numbers.Five, sigma)
		if !errors.Is(err, photometry.ErrMagnitude()) || flux != nil {
			test.Errorf("an error of %v gave %v, %v", sigma, flux, err)
		}
	}
}

// The anti-test: a magnitude so bright its flux is past a float is
// refused, not handed back as an endless flux.
func TestTooBrightForAFloatIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	flux, err := photometry.WISE().Flux("W3", numbers.TooBright, nothing)
	if !errors.Is(err, photometry.ErrBright()) || flux != nil {
		test.Errorf("magnitude %v gave %v, %v", numbers.TooBright, flux,
			err)
	}
}

// Whatever the magnitude and its error, five fainter is a hundredth of
// the flux, the error scales with the flux, and janskys and si agree.
func TestPogsonsScaleHoldsEverywhere(test *testing.T) {
	rapid.Check(test, pogson{numbers: pickedNumbers(test)}.check)
}

// pogson draws magnitudes and errors and holds them to Pogson's scale.
type pogson struct{ numbers picked }

// check draws a band, a magnitude and an error.
func (scale pogson) check(test *rapid.T) {
	numbers := scale.numbers
	survey := photometry.WISE()
	band := rapid.SampledFrom(survey.Bands()).Draw(test, "band")
	magnitude := rapid.Float64Range(numbers.Magnitudes.Least,
		numbers.Magnitudes.Most).Draw(test, "magnitude")
	sigma := rapid.Float64Range(numbers.Errors.Least, numbers.Errors.Most).
		Draw(test, "error")
	here, err := survey.Flux(band, magnitude, sigma)
	if err != nil {
		test.Fatal(err)
	}
	fainter, err := survey.Flux(band, magnitude+numbers.Five, sigma)
	if err != nil {
		test.Fatal(err)
	}
	hereSI, hereSigma := here.InSI()
	fainterSI, fainterSigma := fainter.InSI()
	if !near(hereSI, numbers.Hundred*fainterSI, numbers.Tolerance) {
		test.Fatalf("%s %v is %v, five fainter %v", band, magnitude,
			hereSI, fainterSI)
	}
	if !near(hereSigma, numbers.Hundred*fainterSigma, numbers.Tolerance) {
		test.Fatalf("errors %v and %v do not scale", hereSigma,
			fainterSigma)
	}
	janskys, _ := here.InJanskys()
	if !near(janskys, hereSI/janskyInSI, numbers.Tolerance) {
		test.Fatalf("%v W m-2 Hz-1 is %v Jy", hereSI, janskys)
	}
}

// janskyInSI is one jansky in W m^-2 Hz^-1. Defined by the IAU.
const janskyInSI = 1e-26

// brightest is a reader's choice the mock stands in for: the brightest
// of some magnitudes in one band, as a flux. It is the shape a catalogue
// reader takes a Photometer in, written here: no caller in celestial
// takes one yet, so these two tests show only that the generated mock
// fits that shape, and no change to photometry's own code can fail them.
func brightest(meter photometry.Photometer, band string,
	magnitudes []float64) (*photometry.Flux, error) {
	least := slices.Min(magnitudes)
	return meter.Flux(band, least, nothing)
}

// A reader that takes a Photometer can be tested with the mock: it asks
// for the flux it means to, once, and passes on what it is given.
func TestAReaderCanStandTheMockIn(test *testing.T) {
	numbers := pickedNumbers(test)
	control := gomock.NewController(test)
	meter := mocks.NewMockPhotometer(control)
	want, err := photometry.WISE().Flux("W4", nothing, nothing)
	if err != nil {
		test.Fatal(err)
	}
	meter.EXPECT().Flux("W4", nothing, nothing).Return(want, nil).
		Times(once)
	got, err := brightest(meter, "W4", []float64{numbers.Five, nothing})
	if err != nil || got != want {
		test.Errorf("the reader gave %v, %v", got, err)
	}
}

// The anti-test: what the photometer refuses, the reader refuses too.
func TestAReaderPassesOnARefusal(test *testing.T) {
	control := gomock.NewController(test)
	meter := mocks.NewMockPhotometer(control)
	meter.EXPECT().Flux("W9", nothing, nothing).
		Return(nil, photometry.ErrBand()).Times(once)
	got, err := brightest(meter, "W9", []float64{nothing})
	if !errors.Is(err, photometry.ErrBand()) || got != nil {
		test.Errorf("the reader gave %v, %v", got, err)
	}
}
