package composite_test

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/celestial/fits"
	"github.com/janearc/celestial/grissom/composite"
	"github.com/janearc/celestial/grissom/internal/picture"
	"github.com/janearc/celestial/grissom/layer"
	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/celestial/research"
	"pgregory.net/rapid"
)

// The places, values and stretches the tests build with.
const (
	east     = 10  // defined: the right ascension the layers lie at
	north    = 0   // defined: the declination they lie at
	halfTurn = 180 // defined: half a turn, to a place on another face
	scale    = 1   // defined: a degree a pixel
	single   = 1   // defined: a one-pixel image's width and height
	none     = 0   // defined: no depth, no level, the stretch's bottom
	full     = 1   // defined: all of a level, and full strength
	half     = 0.5 // defined: half a level
	top      = 8   // defined: the stretch's top
	middle   = 4   // defined: a value halfway up the stretch
	between  = 6   // defined: halfway from the middle to the top
	past     = 1   // defined: a step past what is allowed
	spread   = 1e6 // defined: the most a drawn value is, either way
)

// stopper is what the helpers stop a test or a draw with.
type stopper interface{ Fatal(args ...any) }

// laid is a one-pixel layer of a value at a right ascension on the
// equator, at depth none.
func laid(test stopper, name string, ra, value float64) *layer.Layer {
	return layAt(test, name, ra, value, none)
}

// layAt is a one-pixel layer of a value at a right ascension on the
// equator, at a depth.
func layAt(test stopper, name string, ra, value float64,
	depth int) *layer.Layer {
	placed := picture.Tangent(ra, north, scale)
	file := picture.File(single, single, placed, []float64{value})
	image, err := fits.Open(bytes.NewReader(file))
	if err != nil {
		test.Fatal(err)
	}
	made, err := layer.Lay(name, image, depth, unknown)
	if err != nil {
		test.Fatal(err)
	}
	return made
}

// here is the cell the layers lie in.
func here(test stopper) mesh.Cell {
	place, err := mesh.NewPoint(east, north)
	if err != nil {
		test.Fatal(err)
	}
	cell, err := mesh.Locate(place, none)
	if err != nil {
		test.Fatal(err)
	}
	return *cell
}

// stacked is a stack of layers at the cell, each stretched from none to
// the top, in the order given.
func stacked(test stopper, values map[string]float64,
	order ...string) *composite.Stack {
	stack := composite.NewStack()
	for _, name := range order {
		err := stack.Add(laid(test, name, east, values[name]), none, top)
		if err != nil {
			test.Fatal(err)
		}
	}
	return stack
}

// levels is the stack's lights at the cell, by name.
func levels(stack *composite.Stack, cell mesh.Cell) map[string]float64 {
	by := make(map[string]float64)
	for _, lit := range stack.At(cell) {
		by[lit.Name()] = lit.Level()
	}
	return by
}

// must stops a test on an error.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// A layer alone shows its value stretched and weighed: halfway up its
// stretch at full strength is half, and at half strength a quarter. Past
// the stretch's ends it is dark or full.
func TestALayerAloneIsItsStretch(t *testing.T) {
	cell := here(t)
	values := map[string]float64{"middle": middle, "below": -top,
		"above": top + top}
	stack := stacked(t, values, "middle", "below", "above")
	shown := levels(stack, cell)
	if shown["middle"] != half || shown["below"] != none ||
		shown["above"] != full {
		t.Errorf("%v", shown)
	}
	must(t, stack.Strength("middle", half))
	if shown := levels(stack, cell); shown["middle"] != half*half {
		t.Errorf("at half strength: %v", shown)
	}
}

// A stretch that starts above none is measured from where it starts: a
// value halfway from its bottom to its top is half.
func TestAStretchIsFromItsBottom(t *testing.T) {
	stack := composite.NewStack()
	raised := laid(t, "raised", east, between)
	must(t, stack.Add(raised, middle, top))
	lights := stack.At(here(t))
	if len(lights) != single || lights[none].Level() != half {
		t.Errorf("%v", lights)
	}
}

// staysLevel is the property of TestEveryLevelIsWithinOne, for one draw.
func staysLevel(t *rapid.T) {
	value := rapid.Float64Range(-spread, spread).Draw(t, "value")
	low := rapid.Float64Range(-spread, spread).Draw(t, "low")
	width := rapid.Float64Range(full, spread).Draw(t, "width")
	strength := rapid.Float64Range(none, full).Draw(t, "strength")
	stack := composite.NewStack()
	if err := stack.Add(laid(t, "drawn", east, value), low,
		low+width); err != nil {
		t.Fatal(err)
	}
	if err := stack.Strength("drawn", strength); err != nil {
		t.Fatal(err)
	}
	lights := stack.At(here(t))
	if len(lights) != single || lights[none].Level() < none ||
		lights[none].Level() > strength {
		t.Fatalf("%v", lights)
	}
}

// Property: every level lies from none to the layer's strength.
func TestEveryLevelIsWithinOne(t *testing.T) {
	rapid.Check(t, staysLevel)
}

// Only layers that are on and hold the cell are lit, in the stack's
// order.
func TestOnlyLayersOnAndHereAreLit(t *testing.T) {
	cell := here(t)
	stack := composite.NewStack()
	for _, name := range []string{"first", "second", "third"} {
		put(t, stack, name, east, middle)
	}
	put(t, stack, "away", east+halfTurn, middle)
	must(t, stack.Turn("second", false))
	lights := stack.At(cell)
	if len(lights) != len("ab") || lights[none].Name() != "first" ||
		lights[full].Name() != "third" {
		t.Errorf("%v", lights)
	}
	must(t, stack.Turn("second", true))
	if lights := stack.At(cell); len(lights) != len("abc") {
		t.Errorf("turned back on: %v", lights)
	}
}

// One layer occluding another dims it by its own level. It does nothing
// when it is off, at no strength, or not at the cell.
func TestOcclusionDimsByTheLevelInFront(t *testing.T) {
	cell := here(t)
	values := map[string]float64{"dust": middle, "light": top}
	stack := stacked(t, values, "dust", "light")
	must(t, stack.Relate("dust", "light", composite.Occludes))
	if shown := levels(stack, cell); shown["light"] != half ||
		shown["dust"] != half {
		t.Errorf("occluded: %v", shown)
	}
	must(t, stack.Strength("dust", none))
	if shown := levels(stack, cell); shown["light"] != full {
		t.Errorf("at no strength: %v", shown)
	}
	must(t, stack.Strength("dust", full))
	must(t, stack.Turn("dust", false))
	if shown := levels(stack, cell); shown["light"] != full {
		t.Errorf("off: %v", shown)
	}
	away := composite.NewStack()
	put(t, away, "dust", east+halfTurn, top)
	put(t, away, "light", east, top)
	must(t, away.Relate("dust", "light", composite.Occludes))
	if shown := levels(away, cell); shown["light"] != full {
		t.Errorf("elsewhere: %v", shown)
	}
}

// One layer masking another shows it only where the mask holds light. A
// mask that is off hides nothing.
func TestAMaskShowsOnlyWhereItIsLit(t *testing.T) {
	cell := here(t)
	for value, want := range map[float64]float64{middle: full, none: none} {
		values := map[string]float64{"mask": value, "light": top}
		stack := stacked(t, values, "mask", "light")
		must(t, stack.Relate("mask", "light", composite.Masks))
		if shown := levels(stack, cell); shown["light"] != want {
			t.Errorf("a mask of %v: %v", value, shown)
		}
		must(t, stack.Turn("mask", false))
		if shown := levels(stack, cell); shown["light"] != full {
			t.Errorf("a mask of %v, off: %v", value, shown)
		}
	}
	away := composite.NewStack()
	put(t, away, "mask", east+halfTurn, top)
	put(t, away, "light", east, top)
	must(t, away.Relate("mask", "light", composite.Masks))
	if shown := levels(away, cell); shown["light"] != none {
		t.Errorf("a mask elsewhere: %v", shown)
	}
}

// A relation acts one way round, and setting the pair beside again
// undoes it.
func TestRelationsActOneWayAndUndo(t *testing.T) {
	cell := here(t)
	values := map[string]float64{"dust": middle, "light": top}
	stack := stacked(t, values, "dust", "light")
	must(t, stack.Relate("light", "dust", composite.Masks))
	must(t, stack.Relate("dust", "light", composite.Occludes))
	must(t, stack.Relate("dust", "light", composite.Beside))
	if shown := levels(stack, cell); shown["light"] != full ||
		shown["dust"] != half {
		t.Errorf("%v", shown)
	}
}

// refusal is one call the stack must refuse, and why.
type refusal struct {
	err  error
	says string
}

// Anti-tests: a stack refuses no layer, a second of a name, a layer at
// another depth, and a stretch with no width or not finite. It refuses
// names it does not hold, strengths past none or one, a layer acting on
// itself and relations not named.
func TestWhatIsNotAStackIsRefused(t *testing.T) {
	stack := stacked(t, map[string]float64{"one": top, "two": top}, "one",
		"two")
	deeper := composite.NewStack()
	put(t, deeper, "shallow", east, top)
	deep := layAt(t, "deep", east, top, full)
	for _, each := range []refusal{
		{stack.Add(nil, none, top), "no layer"},
		{stack.Add(laid(t, "one", east, top), none, top), "a second"},
		{deeper.Add(deep, none, top), "at depth"},
		{stack.Add(laid(t, "flat", east, top), top, top), "stretched"},
		{stack.Add(laid(t, "nan", east, top), none, math.NaN()),
			"stretched"},
		{stack.Add(laid(t, "wide", east, top), -math.MaxFloat64,
			math.MaxFloat64), "stretched"},
		{stack.Add(laid(t, "inf", east, top), none, math.Inf(full)),
			"stretched"},
		{stack.Turn("three", true), "no layer named three"},
		{stack.Strength("three", half), "no layer named three"},
		{stack.Strength("one", -half), "at strength"},
		{stack.Strength("one", full+half), "at strength"},
		{stack.Strength("one", math.NaN()), "at strength"},
		{stack.Relate("three", "one", composite.Masks), "named three"},
		{stack.Relate("one", "three", composite.Masks), "named three"},
		{stack.Relate("one", "one", composite.Masks), "acting on one"},
	} {
		if !errors.Is(each.err, composite.Refused()) ||
			!strings.Contains(each.err.Error(), each.says) {
			t.Errorf("%q: %v", each.says, each.err)
		}
	}
	for _, unnamed := range []composite.Relation{composite.Beside - past,
		composite.Masks + past} {
		err := stack.Relate("one", "two", unnamed)
		if !errors.Is(err, composite.Refused()) ||
			!strings.Contains(err.Error(), "by relation") {
			t.Errorf("relation %d: %v", unnamed, err)
		}
	}
}

// unknown is the source of the tests' layers: none.
var unknown = layer.NewSource("", "", time.Unix(none, none), none, "")

// put adds a one-pixel layer to a stack, stretched from none to the top.
func put(t *testing.T, stack *composite.Stack, name string, ra,
	value float64) {
	t.Helper()
	made := laid(t, name, ra, value)
	must(t, stack.Add(made, none, top))
}

// A logarithmic stretch keeps its ends: dark below and full above. It
// lifts what lies halfway up to where the curve's formula puts it, worked
// here from the formula; nothing holds the curve to DS9's.
func TestALogarithmicStretchLiftsTheFaint(t *testing.T) {
	cell := here(t)
	values := map[string]float64{"bottom": none, "middle": middle,
		"top": top}
	stack := stacked(t, values, "bottom", "middle", "top")
	for _, name := range []string{"bottom", "middle", "top"} {
		must(t, stack.Shape(name, composite.Logarithmic))
	}
	shown := levels(stack, cell)
	steep := research.Value("log-stretch")
	lifted := math.Log1p(steep*half) / math.Log1p(steep)
	if shown["bottom"] != none || shown["top"] != full ||
		shown["middle"] != lifted {
		t.Errorf("%v", shown)
	}
	must(t, stack.Shape("middle", composite.Linear))
	if shown := levels(stack, cell); shown["middle"] != half {
		t.Errorf("linear again: %v", shown)
	}
}

// keepsOrder is the property of TestALogarithmicStretchKeepsOrder, for
// one draw.
func keepsOrder(t *rapid.T) {
	low := rapid.Float64Range(-spread, spread).Draw(t, "low")
	width := rapid.Float64Range(full, spread).Draw(t, "width")
	one := rapid.Float64Range(low, low+width).Draw(t, "one")
	other := rapid.Float64Range(one, low+width).Draw(t, "other")
	stack := composite.NewStack()
	for name, value := range map[string]float64{"one": one,
		"other": other} {
		made := laid(t, name, east, value)
		if err := stack.Add(made, low, low+width); err != nil {
			t.Fatal(err)
		}
		if err := stack.Shape(name, composite.Logarithmic); err != nil {
			t.Fatal(err)
		}
	}
	shown := levels(stack, here(t))
	if shown["one"] > shown["other"] || shown["other"] > full {
		t.Fatalf("%v of %v and %v", shown, one, other)
	}
}

// Property: a logarithmic stretch keeps brighter values brighter, within
// one.
func TestALogarithmicStretchKeepsOrder(t *testing.T) {
	rapid.Check(t, keepsOrder)
}

// Anti-tests: a curve for a name the stack does not hold is refused, and
// so are curves not named.
func TestUnnamedCurvesAreRefused(t *testing.T) {
	stack := stacked(t, map[string]float64{"one": top}, "one")
	if err := stack.Shape("two", composite.Linear); !errors.Is(err,
		composite.Refused()) || !strings.Contains(err.Error(), "two") {
		t.Errorf("no layer: %v", err)
	}
	for _, unnamed := range []composite.Curve{composite.Linear - past,
		composite.Logarithmic + past} {
		err := stack.Shape("one", unnamed)
		if !errors.Is(err, composite.Refused()) ||
			!strings.Contains(err.Error(), "by curve") {
			t.Errorf("curve %d: %v", unnamed, err)
		}
	}
}
