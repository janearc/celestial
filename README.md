# celestial

physics and maths for stuff bigger than you, in golang.

celestial is what a [virtual
observatory](https://www.ivoa.net/about/what-is-vo.html) is built on.
[nilgai](https://github.com/janearc/nilgai) is the first i built on this
package. it reads chandra, allwise and fermi into one record and composites
them into one image so that i can search for AGN on the other side of the
avoidance zone. [slayer](https://github.com/janearc/slayer) is another project
of mine. builds the observatories that don't exist: any telescope you can
imagine, pointed where you like, anywhere you'd like it to be with the physics
exact enough that what it renders is what that instrument would see. i'm real
curious about a photon ring observatory. or how you'd build a graviton
detector. stuff like that.

briefly, celestial provides:

- units and constants, cited, in SI, with conversions to and from the
  natural units some of the physics prefers
- time scales, the calendar, sidereal time, frames and places
- spectra, magnitudes and the atmosphere
- stars, gravitation, and the kerr hole from libholemath
- LIGO and Virgo's events and estimates, and the HDF5 their strain is
  published in, read here

PACKAGES.md is the map: what lives in each package, and the calls you
build with.
