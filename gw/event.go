package gw

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/janearc/celestial/sky"
)

// Event is a merger as a the centre catalogue lists it: its name and version,
// the catalogue it is in, its gps time, and every estimate the centre gives
// for it, by the centre's name for the parameter.
type Event struct {
	name, catalog string
	version       int
	at            sky.Instant
	estimates     map[string]*Estimate
}

// The errors reading the centre gives.
var (
	errEvents = errors.New("gw: not the centre's events")
	errAbsent = errors.New("gw: no such estimate")
)

// ErrEvents is a file that is not the centre's event json, or an event in it
// without its name or its gps time.
func ErrEvents() error { return errEvents }

// ErrAbsent is a parameter an event has no estimate for: the centre left it
// blank, or never gave it.
func ErrAbsent() error { return errAbsent }

// String is the event's name and version, as the centre keys it: GW150914-v3.
func (event *Event) String() string {
	return fmt.Sprintf("%s-v%d", event.name, event.version)
}

// Catalog is the short name of the catalogue the event is in.
func (event *Event) Catalog() string { return event.catalog }

// At is the event's gps time, as the centre gives it.
func (event *Event) At() sky.Instant { return event.at }

// Parameters is the name of every parameter the event has an estimate
// for, in order.
func (event *Event) Parameters() []string {
	return slices.Sorted(maps.Keys(event.estimates))
}

// Estimator gives an event's estimates, by the centre's name for each
// parameter: an Event is one. A caller that works from an event's
// estimates takes an Estimator, so its tests stand a mock in
// (celestial/mocks) and expect its calls.
type Estimator interface {
	Estimate(parameter string) (*Estimate, error)
}

// An Event is an Estimator; the compiler holds it to that here.
var _ Estimator = &Event{}

// Estimate is the event's estimate of a parameter, by the centre's name
// for it. It refuses a parameter the event has no estimate for, and then
// gives none.
func (event *Event) Estimate(parameter string) (*Estimate, error) {
	if estimate, held := event.estimates[parameter]; held {
		return estimate, nil
	}
	return nil, fmt.Errorf("%w: %q for %s", errAbsent, parameter, event)
}

// ReadEvents is every event in the centre's event json, as its event api
// gives it: an object of events, each keyed by its name and version. It
// refuses a file that is not that, an event without its name or gps
// time, and a unit gw has no conversion for, and then gives none.
func ReadEvents(raw []byte) ([]*Event, error) {
	var file struct {
		Events map[string]map[string]json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("%w: %w", errEvents, err)
	}
	if len(file.Events) == nothingRead {
		return nil, fmt.Errorf("%w: no events", errEvents)
	}
	var events []*Event
	for _, key := range slices.Sorted(maps.Keys(file.Events)) {
		event, err := readEvent(key, file.Events[key])
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// nothingRead is a count of nothing. Defined by counting.
const nothingRead = 0
