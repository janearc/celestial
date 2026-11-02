// Package photometry converts magnitudes to spectral flux densities using
// Pogson’s scale and each survey’s published zero points. Five magnitudes
// correspond to a factor of one hundred: F = F0 * 100^(-m/5).
//
// Survey data supplies band names such as W1 and W4. Callers use Photometer so
// catalogue-reader tests can substitute generated mocks while this package
// tests the conversion itself. Fluxes are available in SI (W m^-2 Hz^-1) and
// janskys.
package photometry
