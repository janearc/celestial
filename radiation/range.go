package radiation

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sync"
)

// Range is a half-open interval of wavelengths in nanometres: the short
// endpoint is included and the long endpoint excluded.
type Range struct{ short, long float64 }

// NewRange constructs a wavelength interval in nanometres. Both ends must be
// finite, with 0 < short < long; invalid endpoints return an error.
func NewRange(short, long float64) (*Range, error) {
	span := Range{short: short, long: long}
	if !valid(span) {
		return nil, fmt.Errorf("radiation: %g to %g nm is no range",
			short, long)
	}
	return &span, nil
}

// Edges returns the short and long endpoints in nanometres.
func (span Range) Edges() (short, long float64) { return span.short, span.long }

// String formats the wavelength interval for a person reading it.
func (span Range) String() string {
	return fmt.Sprintf("%g to %g nm", span.short, span.long)
}

// nothing is no wavelength. It is defined by the scale.
const nothing = 0.0

// eitherSign is the sign math.IsInf takes to mean either infinity.
// Defined by math.IsInf.
const eitherSign = 0

// valid checks that the endpoints are finite and satisfy 0 < short < long.
func valid(span Range) bool {
	return span.short > nothing && span.long > span.short &&
		!math.IsInf(span.long, eitherSign)
}

// Holds checks wavelength membership, including the short endpoint and
// excluding the long. The zero Range contains nothing.
func (span Range) Holds(nm float64) bool {
	return valid(span) && nm >= span.short && nm < span.long
}

// fraction returns position along the range in log wavelength: zero at the
// short end and one at the long end.
func (span Range) fraction(nm float64) float64 {
	return math.Log(nm/span.short) / math.Log(span.long/span.short)
}

// at reverses fraction, interpolating in log wavelength.
func (span Range) at(fraction float64) float64 {
	return span.short * math.Pow(span.long/span.short, fraction)
}

// rangesText is the named ranges, as ranges.json holds them.
//
//go:embed ranges.json
var rangesText []byte

// edges is a range as ranges.json writes it.
type edges struct {
	Short float64 `json:"short"`
	Long  float64 `json:"long"`
}

// named is ranges.json, read.
type named struct {
	ISO   map[string]edges `json:"iso"`
	Fermi struct {
		Highest float64 `json:"highest"`
		Lowest  float64 `json:"lowest"`
	} `json:"fermi ev"`
}

// readNamed is ranges.json read, or nil and the error that stopped it.
func readNamed() (*named, error) {
	var ranges named
	if err := json.Unmarshal(rangesText, &ranges); err != nil {
		return nil, err
	}
	return &ranges, nil
}

// namedRanges is ranges.json, read once.
var namedRanges = sync.OnceValues(readNamed)

// loaded reads the embedded range data. Invalid data is an internal error and
// panics with the cause.
func loaded() *named {
	ranges, err := namedRanges()
	if err != nil {
		panic(fmt.Errorf("radiation: ranges.json: %w", err))
	}
	return ranges
}

// iso is the range ranges.json holds under a name of iso 21348's. A name
// it does not hold is a mistake in radiation's code, which its tests
// find, so asking for one stops the program and says which.
//
// Each was read on 2026-10-31 in tobiska and nusinov's account of the
// standard's draft (cospar 2004), by its authors, since the published
// standard is not open; ranges.json's about says where.
//
// They overlap, ultraviolet with extreme ultraviolet and radio with
// infrared and microwave, so a wavelength may be in two.
func iso(name string) Range {
	written, held := loaded().ISO[name]
	if !held {
		panic(fmt.Errorf("radiation: ranges.json has no %q", name))
	}
	return Range{short: written.Short, long: written.Long}
}

// Gamma is gamma rays, under ISO 21348's name: see iso.
func Gamma() Range { return iso("gamma ray") }

// XRay is x-rays, hard and soft together, under ISO 21348's name: see iso.
func XRay() Range { return iso("x-ray") }

// ExtremeUltraviolet is the extreme ultraviolet, under ISO 21348's name, as its
// draft gives it: see iso. It overlaps Ultraviolet.
func ExtremeUltraviolet() Range { return iso("extreme ultraviolet") }

// Ultraviolet is the ultraviolet, under ISO 21348's name: see iso. It overlaps
// ExtremeUltraviolet.
func Ultraviolet() Range { return iso("ultraviolet") }

// Infrared is the infrared, under ISO 21348's name: see iso. Radio begins
// inside it.
func Infrared() Range { return iso("infrared") }

// Microwave is microwaves, under ISO 21348's name: see iso. Radio begins inside
// it.
func Microwave() Range { return iso("microwave") }

// Radio is radio waves, under ISO 21348's name: see iso. It begins inside
// Infrared and Microwave.
func Radio() Range { return iso("radio") }

// Fermi returns the mission’s energy coverage as wavelengths. The higher
// energy becomes the shorter wavelength; this differs from the ISO range.
//
// NASA’s High Energy Astrophysics Science Archive Research Center gives
// 10 keV to 300 GeV, read on 2026-10-31. Use this interval when matching
// observations collected by the mission.
func Fermi() Range {
	energies := loaded().Fermi
	return Range{short: FromEnergy(energies.Highest),
		long: FromEnergy(energies.Lowest)}
}
