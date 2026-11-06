package galactocentric_test

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/janearc/celestial/galactocentric"
	"github.com/janearc/celestial/length"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/si"
	"github.com/janearc/celestial/sky"
)

// picked is the numbers these tests use, from
// testdata/galactocentric.json.
type picked struct {
	Sun struct {
		Distance     float64 `json:"distance kpc"`
		Height       float64 `json:"height kpc"`
		TowardCentre float64 `json:"toward centre"`
		RoomKpc      float64 `json:"room kpc"`
		RoomSpeed    float64 `json:"room km/s"`
	}
	SgrA struct {
		Ascension         float64
		Declination       float64
		LongitudeMotion   float64 `json:"longitude motion"`
		LatitudeMotion    float64 `json:"latitude motion"`
		AscensionMotion   float64 `json:"ascension motion"`
		DeclinationMotion float64 `json:"declination motion"`
		RoomKpc           float64 `json:"room kpc"`
		RoomSpeed         float64 `json:"room km/s"`
	} `json:"sgr a"`
	AUAYear   float64 `json:"au a year"`
	AURoom    float64 `json:"au room"`
	IAUCentre struct {
		RoomKpc float64 `json:"room kpc"`
	} `json:"iau centre"`
	Pole struct {
		Distance float64 `json:"distance kpc"`
		RoomKpc  float64 `json:"room kpc"`
	}
	HalfWay struct {
		Share     float64
		Radial    float64 `json:"radial km/s"`
		RoomKpc   float64 `json:"room kpc"`
		RoomSpeed float64 `json:"room km/s"`
	} `json:"half way"`
	Drawn struct {
		NearestMas  float64 `json:"nearest mas"`
		FarthestMas float64 `json:"farthest mas"`
		FastestMas  float64 `json:"fastest mas a year"`
		FastestKmS  float64 `json:"fastest km/s"`
		Turn        float64
		Pole        float64
	}
	PropertyRoom float64 `json:"property room"`
	Refused      struct {
		Parallax, Radial, Distance, Height, Declination, Speed float64
	}
	None float64
}

// pickedNumbers reads the test numbers, or the test stops.
func pickedNumbers(test *testing.T) picked {
	test.Helper()
	var numbers picked
	raw, err := os.ReadFile("testdata/galactocentric.json")
	if err != nil {
		test.Fatal(err)
	}
	if err := json.Unmarshal(raw, &numbers); err != nil {
		test.Fatal(err)
	}
	return numbers
}

// The tests' own numbers, each defined.
const (
	masInArcsecond = 1000 // defined: a thousand milliarcseconds in one
	whole          = 1.0  // defined: the whole of a share
	nothing        = 0.0  // defined: no length, no speed
	halves         = 2    // defined: a chord is twice the sine of half its angle
)

// kiloparsec is the metres in one.
var kiloparsec = si.Kilo(length.Parsec)

// published is the ledger's frame, or the test stops.
func published(test *testing.T) *galactocentric.Frame {
	test.Helper()
	frame, err := galactocentric.Published()
	if err != nil {
		test.Fatal(err)
	}
	return frame
}

// kpc is a coordinate as kiloparsecs along each axis.
func kpc(place measure.Coord) (x, y, z float64) {
	by := place.Minus(measure.Origin())
	return by.AlongX().InMetres() / kiloparsec,
		by.AlongY().InMetres() / kiloparsec,
		by.AlongZ().InMetres() / kiloparsec
}

// kms is a velocity as kilometres per second along each axis.
func kms(moving measure.Velocity) (x, y, z float64) {
	return moving.AlongX().InKilometresPerSecond(),
		moving.AlongY().InKilometresPerSecond(),
		moving.AlongZ().InKilometresPerSecond()
}

// size is the length of a triple.
func size(x, y, z float64) float64 { return math.Sqrt(x*x + y*y + z*z) }

// apart is the angle in radians between two directions on the sky, by
// the spherical law of cosines.
func apart(one, other sky.Equatorial) float64 {
	oneUp, otherUp := one.Declination().InRadians(),
		other.Declination().InRadians()
	round := one.Ascension().InRadians() - other.Ascension().InRadians()
	return math.Acos(math.Sin(oneUp)*math.Sin(otherUp) +
		math.Cos(oneUp)*math.Cos(otherUp)*math.Cos(round))
}

// gap is the angle in radians between two directions on the sky, from
// the chord between their unit vectors: true for the smallest angles,
// which the law of cosines loses in rounding.
func gap(one, other sky.Equatorial) float64 {
	oneX, oneY, oneZ := unit(one)
	otherX, otherY, otherZ := unit(other)
	chord := size(oneX-otherX, oneY-otherY, oneZ-otherZ)
	return halves * math.Asin(chord/halves)
}

// unit is the unit vector toward a direction on the sky.
func unit(seen sky.Equatorial) (x, y, z float64) {
	round, up := seen.Ascension().InRadians(), seen.Declination().InRadians()
	return math.Cos(up) * math.Cos(round), math.Cos(up) * math.Sin(round),
		math.Sin(up)
}

// within says whether a number is within room of another.
func within(got, want, room float64) bool {
	return math.Abs(got-want) <= room
}

// stopping is a test or a property's draw that can be stopped.
type stopping interface {
	Helper()
	Fatal(args ...any)
}

// equatorial is a direction on the sky from degrees, or the test stops.
func equatorial(test stopping, ascension,
	declination float64) sky.Equatorial {
	test.Helper()
	seen, err := sky.NewEquatorial(measure.Degrees(ascension),
		measure.Degrees(declination))
	if err != nil {
		test.Fatal(err)
	}
	return *seen
}

// parallaxOf is the parallax of a star so many kiloparsecs away.
func parallaxOf(distance float64) measure.Angle {
	return measure.Arcseconds(whole / distance / masInArcsecond)
}

// placed is a star's place and motion in a frame, or the test stops.
func placed(test stopping, frame *galactocentric.Frame,
	seen sky.Equatorial, parallax measure.Angle,
	motion galactocentric.ProperMotion, radial float64) (measure.Coord,
	measure.Velocity) {
	test.Helper()
	place, moving, err := frame.Of(seen, parallax, motion, radial)
	if err != nil {
		test.Fatal(err)
	}
	return *place, *moving
}

// the published frame is the ledger's: the sun as far from the centre
// and as high above the plane as measured, moving toward the centre as
// measured, and along the rotation and toward the pole as sgr a*'s
// apparent motion reflects, at an astronomical unit a julian year for
// each milliarcsecond a year at a kiloparsec.
func TestThePublishedFrameIsTheLedgers(test *testing.T) {
	numbers := pickedNumbers(test)
	frame := published(test)
	sun := numbers.Sun
	place, moving := frame.Sun()
	x, y, z := kpc(place)
	wantX := -math.Sqrt(sun.Distance*sun.Distance - sun.Height*sun.Height)
	if !within(x, wantX, sun.RoomKpc) || !within(y, nothing, sun.RoomKpc) ||
		!within(z, sun.Height, sun.RoomKpc) {
		test.Errorf("the sun is at %g %g %g kpc, want %g 0 %g", x, y, z, wantX,
			sun.Height)
	}
	reflected := numbers.AUAYear * sun.Distance
	vx, vy, vz := kms(moving)
	wantY := -reflected * numbers.SgrA.LongitudeMotion
	wantZ := -reflected * numbers.SgrA.LatitudeMotion
	room := sun.RoomSpeed + numbers.AURoom*reflected
	if !within(vx, sun.TowardCentre, sun.RoomSpeed) || !within(vy, wantY,
		room) || !within(vz, wantZ, room) {
		test.Errorf("the sun moves at %g %g %g km/s, want %g %g %g", vx, vy,
			vz, sun.TowardCentre, wantY, wantZ)
	}
}

// the test and the anti-test: sgr a* as reid and brunthaler see it,
// with the equatorial components of its apparent motion and no radial
// velocity, is at the centre, at rest along the rotation and toward
// the pole to within what the paper's two fits of one motion agree,
// and moving toward the centre only as the sun does: the frame's one
// motion it does not take from sgr a*.
func TestSgrAStarIsAtRestAtTheCentre(test *testing.T) {
	numbers := pickedNumbers(test)
	frame := published(test)
	sgr := numbers.SgrA
	place, moving := placed(test, frame, equatorial(test, sgr.Ascension,
		sgr.Declination), parallaxOf(numbers.Sun.Distance),
		galactocentric.NewProperMotion(sgr.AscensionMotion,
			sgr.DeclinationMotion), numbers.None)
	x, y, z := kpc(place)
	if !within(size(x, y, z), nothing, sgr.RoomKpc) {
		test.Errorf("sgr a* is at %g %g %g kpc, not the centre", x, y, z)
	}
	vx, vy, vz := kms(moving)
	if !within(vx, numbers.Sun.TowardCentre, sgr.RoomSpeed) ||
		!within(vy, nothing, sgr.RoomSpeed) || !within(vz, nothing,
		sgr.RoomSpeed) {
		test.Errorf("sgr a* moves at %g %g %g km/s, want %g 0 0", vx, vy, vz,
			numbers.Sun.TowardCentre)
	}
}

// the anti-test: the iau's galactic centre, where longitude and
// latitude are nothing, is not sgr a*: at the sun's distance it lands
// as far from it as that distance times the angle between the two on
// the sky, ten parsecs, and not at it.
func TestTheIAUsCentreIsTenParsecsFromSgrAStar(test *testing.T) {
	numbers := pickedNumbers(test)
	frame := published(test)
	centre, err := sky.NewGalactic(measure.Degrees(numbers.None),
		measure.Degrees(numbers.None))
	if err != nil {
		test.Fatal(err)
	}
	place, _ := placed(test, frame, centre.Equatorial(),
		parallaxOf(numbers.Sun.Distance), galactocentric.NewProperMotion(
			numbers.None, numbers.None), numbers.None)
	sgr := equatorial(test, numbers.SgrA.Ascension, numbers.SgrA.Declination)
	want := numbers.Sun.Distance * apart(centre.Equatorial(), sgr)
	off := size(kpc(place))
	if !within(off, want, numbers.IAUCentre.RoomKpc) {
		test.Errorf("the iau's centre is %g kpc from sgr a*, want %g", off,
			want)
	}
}

// the test and the anti-test: a star a kiloparsec toward the north
// galactic pole rises above the sun by its distance, leaning toward
// the centre by the tilt that puts the sun at its height and by sgr
// a*'s own latitude, since the x axis runs to sgr a* and the iau's
// plane does not: a parsec or two, so it is not straight up.
func TestAStarTowardThePoleRisesByItsDistance(test *testing.T) {
	numbers := pickedNumbers(test)
	frame := published(test)
	pole, err := sky.NewGalactic(measure.Degrees(numbers.None),
		measure.Degrees(numbers.Drawn.Pole))
	if err != nil {
		test.Fatal(err)
	}
	far := numbers.Pole.Distance
	place, _ := placed(test, frame, pole.Equatorial(), parallaxOf(far),
		galactocentric.NewProperMotion(numbers.None, numbers.None),
		numbers.None)
	sun, _ := frame.Sun()
	sunX, sunY, sunZ := kpc(sun)
	x, y, z := kpc(place)
	sgr := equatorial(test, numbers.SgrA.Ascension, numbers.SgrA.Declination)
	leaning := math.Asin(numbers.Sun.Height/numbers.Sun.Distance) +
		sgr.Galactic().Latitude().InRadians()
	room := numbers.Pole.RoomKpc
	if !within(x-sunX, far*math.Sin(leaning), room) ||
		!within(y, sunY, room) || !within(z-sunZ, far*math.Cos(leaning), room) {
		test.Errorf("a star %g kpc toward the pole is at %g %g %g kpc, the"+
			" sun at %g %g %g, leaning %g", far, x, y, z, sunX, sunY, sunZ,
			leaning)
	}
}

// the test and the anti-test: a star half way to sgr a*, moving across
// the sky as sgr a* does and 400 km/s away, is where the hand puts it
// from the frame's own numbers. its place is half the sun's, along the
// line of sight: half the sun's distance toward the centre, half the
// sun's height. it moves at the sun's motion toward the centre plus the
// radial velocity along the line of sight; at half the sun's motion
// along the rotation and toward the pole, since sgr a*'s proper motion
// at half its distance is half its speed against the sun, reflected;
// less the radial velocity's share of the tilt, toward the pole.
//
// each axis shows: the radial velocity's sign in x (negated, the star
// comes at 400 km/s instead of going); the y axis in y (mirrored, the
// sun's motion along the rotation is added to the star's instead of
// taken from it); the tilt in z, in the place and in the radial
// velocity's kilometre a second (flipped, both change sign).
func TestAStarHalfWayToSgrAStarIsWhereTheHandPutsIt(test *testing.T) {
	numbers := pickedNumbers(test)
	frame := published(test)
	sun, sgr, half := numbers.Sun, numbers.SgrA, numbers.HalfWay
	place, moving := placed(test, frame, equatorial(test, sgr.Ascension,
		sgr.Declination), parallaxOf(sun.Distance*half.Share),
		galactocentric.NewProperMotion(sgr.AscensionMotion,
			sgr.DeclinationMotion), half.Radial)
	tilt := math.Asin(sun.Height / sun.Distance)
	left := whole - half.Share
	x, y, z := kpc(place)
	wantX, wantZ := -sun.Distance*math.Cos(tilt)*left, sun.Height*left
	if !within(x, wantX, half.RoomKpc) || !within(y, nothing, half.RoomKpc) ||
		!within(z, wantZ, half.RoomKpc) {
		test.Errorf("the star is at %g %g %g kpc, want %g 0 %g", x, y, z,
			wantX, wantZ)
	}
	reflected := numbers.AUAYear * sun.Distance
	wantVX := sun.TowardCentre + half.Radial*math.Cos(tilt)
	wantVY := -reflected * sgr.LongitudeMotion * left
	wantVZ := -reflected*sgr.LatitudeMotion*left - half.Radial*math.Sin(tilt)
	vx, vy, vz := kms(moving)
	if !within(vx, wantVX, half.RoomSpeed) || !within(vy, wantVY,
		half.RoomSpeed) || !within(vz, wantVZ, half.RoomSpeed) {
		test.Errorf("the star moves at %g %g %g km/s, want %g %g %g", vx, vy,
			vz, wantVX, wantVY, wantVZ)
	}
}

// the anti-test: a star without a parallax has no distance, and the
// refusal says so; one that would outrun light is refused too, with
// nothing beside the error.
func TestAStarWithoutAParallaxIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	frame := published(test)
	sgr := numbers.SgrA
	seen := equatorial(test, sgr.Ascension, sgr.Declination)
	still := galactocentric.NewProperMotion(numbers.None, numbers.None)
	for _, parallax := range []float64{numbers.Refused.Parallax,
		-numbers.Refused.Parallax - whole} {
		place, moving, err := frame.Of(seen, measure.Arcseconds(parallax),
			still, numbers.None)
		if err == nil || !strings.Contains(err.Error(), "parallax") ||
			place != nil || moving != nil {
			test.Errorf("a parallax of %g arcseconds gave %v, %v, %v",
				parallax, place, moving, err)
		}
	}
	place, moving, err := frame.Of(seen, parallaxOf(numbers.Sun.Distance),
		still, numbers.Refused.Radial)
	if err == nil || place != nil || moving != nil {
		test.Errorf("a radial velocity of %g km/s gave %v, %v, %v",
			numbers.Refused.Radial, place, moving, err)
	}
}

// the anti-test: a frame the physics cannot hold is refused: a centre
// past a pole, a sun at the centre, a sun as high as it is far or
// higher, a sun past light.
func TestAFrameThePhysicsCannotHoldIsRefused(test *testing.T) {
	numbers := pickedNumbers(test)
	sun, sgr, refused := numbers.Sun, numbers.SgrA, numbers.Refused
	distance := measure.Metres(sun.Distance * kiloparsec)
	height := measure.Metres(sun.Height * kiloparsec)
	cases := map[string]struct {
		ascension, declination  float64
		distance, height        measure.Length
		toCentre, along, toPole float64
	}{
		"a centre past a pole": {sgr.Ascension, refused.Declination, distance,
			height, sun.TowardCentre, numbers.None, numbers.None},
		"a sun at the centre": {sgr.Ascension, sgr.Declination,
			measure.Metres(refused.Distance), height, sun.TowardCentre,
			numbers.None, numbers.None},
		"a sun as high as it is far": {sgr.Ascension, sgr.Declination,
			distance, distance, sun.TowardCentre, numbers.None, numbers.None},
		"a sun higher than it is far": {sgr.Ascension, sgr.Declination,
			distance, measure.Metres(refused.Height * kiloparsec),
			sun.TowardCentre, numbers.None, numbers.None},
		"a sun past light": {sgr.Ascension, sgr.Declination, distance, height,
			refused.Speed, numbers.None, numbers.None},
	}
	for name, each := range cases {
		frame, err := galactocentric.NewFrame(measure.Degrees(each.ascension),
			measure.Degrees(each.declination), each.distance, each.height,
			each.toCentre, each.along, each.toPole)
		if err == nil || frame != nil {
			test.Errorf("%s was not refused: %v, %v", name, frame, err)
		}
	}
}

// properties is the frame's properties, under a test.
type properties struct {
	test    *testing.T
	numbers picked
	frame   *galactocentric.Frame
}

// observed is a star as drawn: where it is, its parallax in
// milliarcseconds, its proper motion and its radial velocity.
type observed struct {
	seen     sky.Equatorial
	mas      float64
	motion   galactocentric.ProperMotion
	radial   float64
	across   float64
	distance float64
}

// the property: the frame keeps a star's distance from the sun and its
// speed against the sun, whatever its place on the sky and however it
// moves: turning and moving the axes changes neither.
func TestDistancesAndSpeedsAreKept(test *testing.T) {
	numbers := pickedNumbers(test)
	holds := properties{test, numbers, published(test)}
	rapid.Check(test, holds.keepsDistanceAndSpeed)
}

// keepsDistanceAndSpeed is the property on one drawn star.
func (holds properties) keepsDistanceAndSpeed(draw *rapid.T) {
	star := holds.anyStar(draw)
	place, moving := placed(draw, holds.frame, star.seen,
		measure.Arcseconds(star.mas/masInArcsecond), star.motion, star.radial)
	sun, sunMoving := holds.frame.Sun()
	sunX, sunY, sunZ := kpc(sun)
	x, y, z := kpc(place)
	room := holds.numbers.PropertyRoom
	if got := size(x-sunX, y-sunY, z-sunZ); !within(got, star.distance,
		room*star.distance) {
		draw.Errorf("a star %g kpc from the sun is placed %g kpc from it",
			star.distance, got)
	}
	vx, vy, vz := kms(moving)
	sx, sy, sz := kms(sunMoving)
	if got := size(vx-sx, vy-sy, vz-sz); !within(got, star.across,
		(room+holds.numbers.AURoom)*star.across+room) {
		draw.Errorf("a star moving %g km/s against the sun is given %g",
			star.across, got)
	}
}

// anyStar is a star drawn from the picked ranges, with its distance in
// kiloparsecs and its speed against the sun in kilometres per second
// as the draw makes them. Its proper motion is drawn no faster across
// the sky than the fastest speed at its distance, so no drawn star
// outruns light far out.
func (holds properties) anyStar(draw *rapid.T) observed {
	drawn := holds.numbers.Drawn
	ascension := rapid.Float64Range(nothing, drawn.Turn).Draw(draw,
		"ascension")
	declination := rapid.Float64Range(-drawn.Pole, drawn.Pole).Draw(draw,
		"declination")
	mas := rapid.Float64Range(drawn.NearestMas, drawn.FarthestMas).Draw(
		draw, "parallax")
	fastest := math.Min(drawn.FastestMas,
		drawn.FastestKmS*mas/holds.numbers.AUAYear)
	alongAscension := rapid.Float64Range(-fastest, fastest).Draw(draw,
		"ascension motion")
	alongDeclination := rapid.Float64Range(-fastest, fastest).Draw(draw,
		"declination motion")
	radial := rapid.Float64Range(-drawn.FastestKmS, drawn.FastestKmS).Draw(
		draw, "radial")
	distance := whole / mas
	tangent := holds.numbers.AUAYear * distance
	return observed{
		seen:     equatorial(draw, ascension, declination),
		mas:      mas,
		motion:   galactocentric.NewProperMotion(alongAscension, alongDeclination),
		radial:   radial,
		across:   size(radial, tangent*alongAscension, tangent*alongDeclination),
		distance: distance,
	}
}

// a proper motion gives back the two numbers it was made of.
func TestAProperMotionGivesBackItsNumbers(test *testing.T) {
	sgr := pickedNumbers(test).SgrA
	motion := galactocentric.NewProperMotion(sgr.AscensionMotion,
		sgr.DeclinationMotion)
	if motion.Ascension() != sgr.AscensionMotion ||
		motion.Declination() != sgr.DeclinationMotion {
		test.Errorf("the motion is %g %g, want %g %g", motion.Ascension(),
			motion.Declination(), sgr.AscensionMotion, sgr.DeclinationMotion)
	}
}

// the test and the anti-test: a frame with the sun moving otherwise
// keeps its axes and the sun's place: sgr a* lands at the centre as
// before, and moves along the rotation by what the sun was slowed by;
// a sun past light is refused.
func TestAFrameWithTheSunMovingOtherwiseKeepsItsAxes(test *testing.T) {
	numbers := pickedNumbers(test)
	frame := published(test)
	sgr, slower := numbers.SgrA, numbers.Sun.TowardCentre
	_, sunMoving := frame.Sun()
	vx, vy, vz := kms(sunMoving)
	moved, err := frame.MovingSun(vx, vy-slower, vz)
	if err != nil {
		test.Fatal(err)
	}
	place, moving := placed(test, moved, equatorial(test, sgr.Ascension,
		sgr.Declination), parallaxOf(numbers.Sun.Distance),
		galactocentric.NewProperMotion(sgr.AscensionMotion,
			sgr.DeclinationMotion), numbers.None)
	x, y, z := kpc(place)
	if !within(size(x, y, z), nothing, sgr.RoomKpc) {
		test.Errorf("sgr a* is at %g %g %g kpc, not the centre", x, y, z)
	}
	if _, got, _ := kms(moving); !within(got, -slower, sgr.RoomSpeed) {
		test.Errorf("sgr a* moves %g km/s along the rotation, want %g", got,
			-slower)
	}
	if past, err := frame.MovingSun(vx, numbers.Refused.Speed,
		vz); err == nil || past != nil {
		test.Errorf("a sun past light gave %v, %v", past, err)
	}
}

// seen is how the frame sees a star at a place, moving so, or the test
// stops.
func seen(test stopping, frame *galactocentric.Frame, place measure.Coord,
	moving measure.Velocity) galactocentric.Sighting {
	test.Helper()
	sighting, err := frame.Seen(place, moving)
	if err != nil {
		test.Fatal(err)
	}
	return *sighting
}

// still is no motion at all, or the test stops.
func still(test stopping) measure.Velocity {
	test.Helper()
	moving, err := measure.KilometresPerSecond(nothing, nothing, nothing)
	if err != nil {
		test.Fatal(err)
	}
	return *moving
}

// the test: sgr a* at rest at the centre is seen where reid and
// brunthaler see it, at its own ascension and declination as numbers,
// at the parallax of the sun's distance, with their equatorial motion
// to within what their two fits agree, and coming toward the sun by
// the sun's own motion along the line of sight, worked by hand: the
// line runs along the centre's axis tilted by the sun's height, so it
// takes the motion toward the centre less the tilt's share of the
// motion toward the pole.
func TestSgrAStarAtRestIsSeenWhereReidAndBrunthalerSeeIt(test *testing.T) {
	numbers := pickedNumbers(test)
	sun, sgr := numbers.Sun, numbers.SgrA
	sighting := seen(test, published(test), measure.Origin(), still(test))
	place := sighting.Place()
	room := numbers.PropertyRoom
	if !within(place.Ascension().InDegrees(), sgr.Ascension, room) ||
		!within(place.Declination().InDegrees(), sgr.Declination, room) {
		test.Errorf("sgr a* is seen at %v, want %g %g", place, sgr.Ascension,
			sgr.Declination)
	}
	mas := sighting.Parallax().InArcseconds() * masInArcsecond
	if !within(mas, whole/sun.Distance, room/sun.Distance) {
		test.Errorf("sgr a* is seen at a parallax of %g mas, want %g", mas,
			whole/sun.Distance)
	}
	reflected := numbers.AUAYear * sun.Distance
	toPole := -reflected * sgr.LatitudeMotion
	tilt := math.Asin(sun.Height / sun.Distance)
	radial := -(sun.TowardCentre*math.Cos(tilt) - toPole*math.Sin(tilt))
	if !within(sighting.Radial(), radial, sun.RoomSpeed) {
		test.Errorf("sgr a* comes at %g km/s, want %g", sighting.Radial(),
			radial)
	}
	motion, roomMas := sighting.Motion(), sgr.RoomSpeed/reflected
	if !within(motion.Ascension(), sgr.AscensionMotion, roomMas) ||
		!within(motion.Declination(), sgr.DeclinationMotion, roomMas) {
		test.Errorf("sgr a* moves %g %g mas a year, want %g %g",
			motion.Ascension(), motion.Declination(), sgr.AscensionMotion,
			sgr.DeclinationMotion)
	}
}

// the anti-test: a star at the sun's own place has no direction to be
// seen in, and is refused with nothing beside the error.
func TestAStarAtTheSunsPlaceIsRefused(test *testing.T) {
	frame := published(test)
	place, _ := frame.Sun()
	sighting, err := frame.Seen(place, still(test))
	if err == nil || sighting != nil ||
		!strings.Contains(err.Error(), "no direction") {
		test.Errorf("a star at the sun's place gave %v, %v", sighting, err)
	}
}

// the property: a star the frame places, seen and placed again, is the
// same star, at the same place and moving the same way, wherever it is
// on the sky, the poles too.
func TestAStarSeenIsPlacedWhereItWas(test *testing.T) {
	numbers := pickedNumbers(test)
	holds := properties{test, numbers, published(test)}
	rapid.Check(test, holds.placesWhatItSees)
}

// placesWhatItSees is the property on one drawn star.
func (holds properties) placesWhatItSees(draw *rapid.T) {
	star := holds.anyStar(draw)
	place, moving := placed(draw, holds.frame, star.seen,
		measure.Arcseconds(star.mas/masInArcsecond), star.motion, star.radial)
	sighting := seen(draw, holds.frame, place, moving)
	again, movingAgain := placed(draw, holds.frame, sighting.Place(),
		sighting.Parallax(), sighting.Motion(), sighting.Radial())
	x, y, z := kpc(place)
	ax, ay, az := kpc(again)
	room := holds.numbers.PropertyRoom
	if got := size(ax-x, ay-y, az-z); !within(got, nothing,
		room*star.distance) {
		draw.Errorf("a star %g kpc from the sun is placed %g kpc off",
			star.distance, got)
	}
	vx, vy, vz := kms(moving)
	bx, by, bz := kms(movingAgain)
	if got := size(bx-vx, by-vy, bz-vz); !within(got, nothing,
		room*(star.across+size(vx, vy, vz))+room) {
		draw.Errorf("a star moving %g km/s against the sun is %g km/s off",
			star.across, got)
	}
}

// the property: a star the frame places is seen as it was drawn: in the
// same direction, at an ascension within one turn, at the same parallax,
// coming or going as fast, and as fast across the sky.
func TestAStarPlacedIsSeenAsItWasDrawn(test *testing.T) {
	numbers := pickedNumbers(test)
	holds := properties{test, numbers, published(test)}
	rapid.Check(test, holds.seesWhatItPlaces)
}

// seesWhatItPlaces is the property on one drawn star.
func (holds properties) seesWhatItPlaces(draw *rapid.T) {
	star := holds.anyStar(draw)
	place, moving := placed(draw, holds.frame, star.seen,
		measure.Arcseconds(star.mas/masInArcsecond), star.motion, star.radial)
	sighting := seen(draw, holds.frame, place, moving)
	room := holds.numbers.PropertyRoom
	if got := gap(sighting.Place(), star.seen); !within(got, nothing,
		room) {
		draw.Errorf("a star at %v is seen %g radians off", star.seen, got)
	}
	ascension := sighting.Place().Ascension().InDegrees()
	if !(ascension >= nothing && ascension < holds.numbers.Drawn.Turn) {
		draw.Errorf("a star is seen at ascension %g, outside one turn",
			ascension)
	}
	mas := sighting.Parallax().InArcseconds() * masInArcsecond
	if !within(mas, star.mas, room*star.mas) {
		draw.Errorf("a parallax of %g mas is seen as %g", star.mas, mas)
	}
	vx, vy, vz := kms(moving)
	speed := size(vx, vy, vz)
	if !within(sighting.Radial(), star.radial, room*(star.across+speed)+
		room) {
		draw.Errorf("a radial velocity of %g km/s is seen as %g",
			star.radial, sighting.Radial())
	}
	tangent := holds.numbers.AUAYear * star.distance
	drawn := tangent * size(star.motion.Ascension(),
		star.motion.Declination(), nothing)
	motion := sighting.Motion()
	got := tangent * size(motion.Ascension(), motion.Declination(), nothing)
	if !within(got, drawn, room*(star.across+speed)+room) {
		draw.Errorf("%g km/s across the sky is seen as %g", drawn, got)
	}
}
