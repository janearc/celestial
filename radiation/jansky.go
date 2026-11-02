package radiation

// jansky is one jansky, in watts a square metre a hertz. The IAU defines
// it as 1e-26 W m^-2 Hz^-1 (the jansky, named for Karl Jansky, adopted in
// 1973 at the IAU's 15th General Assembly). It is defined, not measured.
//
// Read on 2026-10-31 in NRAO's account, "Jansky @90" (2023): the 1973
// assembly resolved that the unit be "equal to 10^-26 W m^-2 Hz^-1". The
// IAU's own transactions for 1973 are not open and were not read.
const jansky = 1e-26

// JanskyToSI is a spectral flux density in janskys as watts a square metre a
// hertz, the unit every exported flux density here is in. The jansky is outside
// the SI: radio and infrared catalogues, WISE's among them, still state flux
// densities in it.
//
// It lives in radiation, not SI, because SI holds the defining constants and
// the prefixes, and a jansky is a unit of radiation: a power per area per
// frequency.
func JanskyToSI(jy float64) float64 { return jy * jansky }
