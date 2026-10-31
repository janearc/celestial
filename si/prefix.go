package si

// The si's decimal prefixes, from the SI Brochure, 9th edition (BIPM,
// 2019), table 7. A unit has one name; its multiples are these functions
// of it, so a kiloparsec is Kilo(length.Parsec) and a keV is
// Kilo(radiation.Electronvolt). Nothing anywhere names a kilo-anything.
//
// Only the prefixes celestial uses are here. libtheme-css holds the
// whole table; a prefix comes over when a caller needs it.
//
// Each is defined by the brochure's table 7.
const (
	nano  = 1e-9 // defined: 10^-9
	micro = 1e-6 // defined: 10^-6
	milli = 1e-3 // defined: 10^-3
	centi = 1e-2 // defined: 10^-2
	kilo  = 1e3  // defined: 10^3
	mega  = 1e6  // defined: 10^6
	giga  = 1e9  // defined: 10^9
	tera  = 1e12 // defined: 10^12
)

// Nano is a thousand-millionth of the unit.
func Nano(unit float64) float64 { return unit * nano }

// Micro is a millionth of the unit: a micron is Micro(length.Metre).
func Micro(unit float64) float64 { return unit * micro }

// Milli is a thousandth of the unit: a millijansky is a Milli of a
// jansky.
func Milli(unit float64) float64 { return unit * milli }

// Centi is a hundredth of the unit: the centimetre of cgs units.
func Centi(unit float64) float64 { return unit * centi }

// Kilo is a thousand of the unit.
func Kilo(unit float64) float64 { return unit * kilo }

// Mega is a million of the unit: a megaparsec is Mega(length.Parsec).
func Mega(unit float64) float64 { return unit * mega }

// Giga is a thousand million of the unit.
func Giga(unit float64) float64 { return unit * giga }

// Tera is a million million of the unit.
func Tera(unit float64) float64 { return unit * tera }
