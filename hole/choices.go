package hole

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

// choicesText is choices.json, built in: the choices the march, a fall
// and the classifier are made of, each with what it says and who picked
// it. It is hole's own, so that hole keeps to the standard library.
//
//go:embed choices.json
var choicesText []byte

// marching is the choices, by what they are for.
type marching struct {
	// pace and time are a step's share of the radius it is taken at, in
	// the affine parameter and in time.
	pace, time float64
	// hover is how many times its energy an arriving ray's time velocity
	// is when the march gives it up as come from the horizon's edge.
	hover float64
	// turnedBack is how far above its least radius a ray has risen, as a
	// share of it, when it is taken to be on its way out.
	turnedBack float64
	// margin is how near the singularity a falling body is advanced, in
	// M, and advance the share of the pace it steps by.
	margin, advance float64
	// farFactor and farFloor are how far out a classified ray has got
	// when it escapes: so many times its starting radius, or so many M.
	farFactor, farFloor float64
	// steps is how many steps a ray is marched to class it.
	steps int
}

// choice is one of choices.json's entries.
type choice struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// The names of the choices in choices.json.
const (
	paceName       = "pace share"
	timeName       = "time share"
	hoverName      = "hover ratio"
	turnedBackName = "turned back share"
	marginName     = "singularity margin"
	advanceName    = "advance share"
	farFactorName  = "far factor"
	farFloorName   = "far floor"
	stepsName      = "classify steps"
)

// picks is choices.json's values by their names.
type picks map[string]float64

// of is the value of a choice by its name. A name choices.json does not
// hold is a mistake in hole's code, which its tests find, so asking for
// one stops the program and says which.
func (held picks) of(name string) float64 {
	value, known := held[name]
	if !known {
		panic(fmt.Errorf("hole: choices.json has no %q", name))
	}
	return value
}

// readMarching reads choices.json. The file is hole's own, built in, and
// one that does not read is a mistake in hole, which its tests find, so
// it stops the program and says why.
func readMarching() marching {
	var file struct {
		Choices []choice `json:"choices"`
	}
	if err := json.Unmarshal(choicesText, &file); err != nil {
		panic(fmt.Errorf("hole: choices.json: %w", err))
	}
	held := picks{}
	for _, each := range file.Choices {
		held[each.Name] = each.Value
	}
	return marching{pace: held.of(paceName), time: held.of(timeName),
		hover: held.of(hoverName), turnedBack: held.of(turnedBackName),
		margin: held.of(marginName), advance: held.of(advanceName),
		farFactor: held.of(farFactorName),
		farFloor:  held.of(farFloorName),
		steps:     int(held.of(stepsName))}
}

// choices is the march's choices, read once.
var choices = sync.OnceValue(readMarching)
