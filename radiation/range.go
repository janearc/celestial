package radiation

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
)

// Range is a band of radiation, by wavelength in nanometres: from its
// short end up to but not including its long end.
type Range struct{ short, long float64 }

// errRange is two wavelengths that make no range.
var errRange = errors.New("radiation: no such range")

// ErrRange is two wavelengths that make no range: a short end not above
// nothing, a long end not above the short, or either not a number.
func ErrRange() error { return errRange }

// NewRange is the range from a short wavelength to a long one, in
// nanometres. It refuses a short end not above nothing and a long end
// not above the short or endless, and then gives no range.
func NewRange(short, long float64) (*Range, error) {
	span := Range{short: short, long: long}
	if !valid(span) {
		return nil, fmt.Errorf("%w: %g to %g nm", errRange, short, long)
	}
	return &span, nil
}

// Edges is the range's short and long ends, in nanometres, for a caller
// that draws the band or lays one range beside another.
func (span Range) Edges() (short, long float64) { return span.short, span.long }

// String is the range in nanometres, for a person reading it.
func (span Range) String() string {
	return fmt.Sprintf("%g to %g nm", span.short, span.long)
}

// nothing is no wavelength. It is defined by the scale.
const nothing = 0.0

// eitherSign asks math.IsInf about both infinities. Defined by math.
const eitherSign = 0

// valid is whether a range is a range: it starts above nothing and ends,
// short of endless, above where it starts.
func valid(span Range) bool {
	return span.short > nothing && span.long > span.short &&
		!math.IsInf(span.long, eitherSign)
}

// Holds is whether a wavelength in nanometres is inside the range, from
// its short end up to but not including its long end. What is no range,
// the zero Range among them, holds nothing.
func (span Range) Holds(nm float64) bool {
	return valid(span) && nm >= span.short && nm < span.long
}

// fraction is how far up a range a wavelength is, evenly in the
// logarithm of wavelength: nothing at the short end and whole at the
// long end.
func (span Range) fraction(nm float64) float64 {
	return math.Log(nm/span.short) / math.Log(span.long/span.short)
}

// at is the wavelength a fraction of the way up a range, evenly in the
// logarithm of wavelength: fraction, undone.
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

// readNamed reads ranges.json, or gives none and why.
func readNamed() (*named, error) {
	var ranges named
	if err := json.Unmarshal(rangesText, &ranges); err != nil {
		return nil, err
	}
	return &ranges, nil
}

// namedRanges is ranges.json, read once.
var namedRanges = sync.OnceValues(readNamed)

// loaded is ranges.json, read. The file is radiation's own, built in,
// and one that does not read is a mistake in radiation, which its tests
// find, so it stops the program and says why.
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
// standard is not open; ranges.json's about says where, and that its
// extreme ultraviolet ends at 120 nm where libtheme-css had 121.
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

// Gamma is gamma rays, under iso 21348's name: see iso.
func Gamma() Range { return iso("gamma ray") }

// XRay is x-rays, hard and soft together, under iso 21348's name: see
// iso.
func XRay() Range { return iso("x-ray") }

// ExtremeUltraviolet is the extreme ultraviolet, under iso 21348's name,
// as its draft gives it: see iso. It overlaps Ultraviolet.
func ExtremeUltraviolet() Range { return iso("extreme ultraviolet") }

// Ultraviolet is the ultraviolet, under iso 21348's name: see iso. It
// overlaps ExtremeUltraviolet.
func Ultraviolet() Range { return iso("ultraviolet") }

// Infrared is the infrared, under iso 21348's name: see iso. Radio
// begins inside it.
func Infrared() Range { return iso("infrared") }

// Microwave is microwaves, under iso 21348's name: see iso. Radio begins
// inside it.
func Microwave() Range { return iso("microwave") }

// Radio is radio waves, under iso 21348's name: see iso. It begins
// inside Infrared and Microwave.
func Radio() Range { return iso("radio") }

// Fermi is the fermi mission's range, a little different from iso's:
// work on gamma ray emitting objects sometimes needs it, depending on
// where an observation was collected.
//
// nasa's heasarc gives it in energies, 10 keV to 300 GeV (read on
// 2026-10-31); as a range of wavelengths the higher energy is the short
// end.
func Fermi() Range {
	energies := loaded().Fermi
	return Range{short: FromEnergy(energies.Highest),
		long: FromEnergy(energies.Lowest)}
}
