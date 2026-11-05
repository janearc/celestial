// Package composite shows layers together, as the Sgr A* page did. Each
// layer is stretched from a low value to a high one and weighed by a
// strength. Each is on or off.
//
// Between each two layers there is one relation. They lie beside each
// other, or one occludes the other, or one masks it.
//
// What comes out at a cell is each lit layer's level after the relations
// acting on it. Levels run from none to one. Colour is given when it is
// drawn; the composite holds none.
package composite

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/grissom/layer"
	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/celestial/research"
)

// The levels a layer is shown at.
const (
	dark = 0 // defined: no light
	full = 1 // defined: all of it
	none = 0 // defined: no layers, and the first place in a stack
)

// errRefused is what Refused gives.
var errRefused = errors.New("refused")

// Refused is wrapped by each refusal here. Layers that are missing, named
// twice or laid at another depth are refused. So are stretches with no
// width and strengths past none or one. So are relations a layer has to
// itself.
func Refused() error { return errRefused }

// Relation is how one layer acts on another.
type Relation int

// The relations one layer may have to another.
const (
	// Beside is no relation: both show as they are. It is every pair's
	// to begin with.
	Beside Relation = iota
	// Occludes is the one in front dimming the other by its own level.
	// Dust dims the light behind it so.
	Occludes
	// Masks is the other showing only where the one holds light.
	Masks
	// relations is how many there are. It's defined.
	relations
)

// shown is one layer as the stack shows it.
type shown struct {
	laid      *layer.Layer
	low, high float64
	strength  float64
	on        bool
	curve     Curve
}

// Curve is how a layer's stretch runs from dark to full.
type Curve int

// The curves a stretch may follow.
const (
	// Linear is a straight stretch: halfway up the values is half. It is
	// every layer's to begin with.
	Linear Curve = iota
	// Logarithmic lifts the faint values and presses the bright ones, as
	// a map of the whole sky needs: its plane is far brighter than the
	// rest.
	Logarithmic
	// curves is how many there are. It's defined.
	curves
)

// logBase is how steep the logarithmic curve is: a stretched value x
// shows as log(1 + logBase x) / log(1 + logBase). It is picked, and
// celestial's research data holds it and says who picked it and why.
func logBase() float64 { return research.Value(logStretchName) }

// logStretchName is logBase's name in the research data.
const logStretchName = "log-stretch"

// Shape sets the curve a layer's stretch follows. It refuses a name the
// stack does not hold and a curve not named here.
func (stack *Stack) Shape(name string, curve Curve) error {
	at, err := stack.place(name)
	if err != nil {
		return err
	}
	if curve < Linear || curve >= curves {
		return fmt.Errorf("%w: layer %s by curve %d", errRefused, name,
			curve)
	}
	stack.shown[at].curve = curve
	return nil
}

// pair is two layers by their places in the stack: the one acting, then
// the one acted on.
type pair struct{ acting, actedOn int }

// Stack is layers shown together, in the order they were added.
type Stack struct {
	shown     []*shown
	places    map[string]int
	relations map[pair]Relation
}

// NewStack is a stack with no layers.
func NewStack() *Stack {
	return &Stack{places: make(map[string]int),
		relations: make(map[pair]Relation)}
}

// Add puts a layer at the back of the stack, on and at full strength.
// Values from low to high are stretched from dark to full.
//
// It refuses no layer and a name the stack holds. It refuses a layer at
// another depth than the first. Stretches that are not finite or have no
// width are refused too.
func (stack *Stack) Add(laid *layer.Layer, low, high float64) error {
	if laid == nil {
		return fmt.Errorf("%w: no layer", errRefused)
	}
	if _, held := stack.places[laid.Name()]; held {
		return fmt.Errorf("%w: a second layer named %s", errRefused,
			laid.Name())
	}
	if !stack.takes(laid) {
		return fmt.Errorf("%w: layer %s at depth %d", errRefused,
			laid.Name(), laid.Depth())
	}
	if !(high > low) || math.IsInf(high-low, dark) {
		return fmt.Errorf("%w: layer %s stretched from %v to %v",
			errRefused, laid.Name(), low, high)
	}
	stack.places[laid.Name()] = len(stack.shown)
	stack.shown = append(stack.shown, &shown{laid: laid, low: low,
		high: high, strength: full, on: true})
	return nil
}

// place is where a named layer is in the stack.
func (stack *Stack) place(name string) (int, error) {
	at, held := stack.places[name]
	if !held {
		return none, fmt.Errorf("%w: no layer named %s", errRefused,
			name)
	}
	return at, nil
}

// Turn turns a layer on or off. Layers that are off show nothing and act
// on nothing.
func (stack *Stack) Turn(name string, on bool) error {
	at, err := stack.place(name)
	if err != nil {
		return err
	}
	stack.shown[at].on = on
	return nil
}

// Strength weighs a layer's level, from none to one. It is the slider the
// Sgr A* page gave each layer.
func (stack *Stack) Strength(name string, strength float64) error {
	at, err := stack.place(name)
	if err != nil {
		return err
	}
	if !(strength >= dark && strength <= full) {
		return fmt.Errorf("%w: layer %s at strength %v", errRefused,
			name,
			strength)
	}
	stack.shown[at].strength = strength
	return nil
}

// Relate sets how one layer acts on another. Each pair holds one relation
// each way round. It refuses layers acting on themselves and relations not
// named here.
func (stack *Stack) Relate(acting, actedOn string, relation Relation) error {
	one, err := stack.place(acting)
	if err != nil {
		return err
	}
	other, err := stack.place(actedOn)
	if err != nil {
		return err
	}
	if one == other || relation < Beside || relation >= relations {
		return fmt.Errorf("%w: %s acting on %s by relation %d",
			errRefused,
			acting, actedOn, relation)
	}
	stack.relations[pair{acting: one, actedOn: other}] = relation
	return nil
}

// Light is one layer's level at a cell.
type Light struct {
	name  string
	level float64
}

// Name is the layer's name.
func (lit Light) Name() string { return lit.name }

// Level is how much of the layer shows, from none to one.
func (lit Light) Level() float64 { return lit.level }

// At is the light at a cell. Each layer that is on and holds the cell is
// there in the stack's order. The relations on it have acted.
func (stack *Stack) At(cell mesh.Cell) []Light {
	levels := make([]float64, len(stack.shown))
	held := make([]bool, len(stack.shown))
	for i, each := range stack.shown {
		levels[i], held[i] = each.level(cell)
	}
	var lights []Light
	for i, each := range stack.shown {
		if !held[i] {
			continue
		}
		level := levels[i]
		for k, other := range stack.shown {
			if !other.on {
				continue
			}
			acting := stack.relations[pair{acting: k, actedOn: i}]
			level = acting.on(level, levels[k], held[k])
		}
		lights = append(lights, Light{name: each.laid.Name(),
			level: level})
	}
	return lights
}

// level is a layer's stretched and weighed level at a cell. It says
// whether the layer shows there: on and holding the cell.
func (each *shown) level(cell mesh.Cell) (float64, bool) {
	value, held := each.laid.At(cell)
	if !each.on || !held {
		return dark, false
	}
	stretched := (value.Mean() - each.low) / (each.high - each.low)
	clamped := math.Max(dark, math.Min(full, stretched))
	if each.curve == Logarithmic {
		steep := logBase()
		clamped = math.Log1p(steep*clamped) / math.Log1p(steep)
	}
	return clamped * each.strength, true
}

// on is a level after a relation acts on it. The acting layer is at its
// own level and may not show at the cell.
func (relation Relation) on(level, acting float64, shows bool) float64 {
	switch {
	case relation == Occludes && shows:
		return level * (full - acting)
	case relation == Masks && !(shows && acting > dark):
		return dark
	}
	return level
}

// takes is whether a layer is at the stack's depth. That is the first
// layer's depth, and any when the stack is empty.
func (stack *Stack) takes(laid *layer.Layer) bool {
	return len(stack.shown) == none ||
		laid.Depth() == stack.shown[none].laid.Depth()
}
