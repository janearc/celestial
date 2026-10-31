// Package photometry is magnitudes and the fluxes they stand for:
// Pogson's scale, and the bands of a survey, each with the flux density
// of magnitude zero the survey publishes.
//
// Pogson's scale is what is defined: five magnitudes are a factor of a
// hundred in flux, so a flux is F0 times 100 to the minus m over 5. A
// band is known by the name its survey writes, W1 or W4; there is no
// list of bands in code, only the survey's data.
//
// A catalogue reader takes a Photometer, so its tests stand a mock in
// for it (celestial/mocks) and expect its calls; the photometry is
// tested once, here. Fluxes come out in si units, W m^-2 Hz^-1, and in
// janskys.
package photometry
