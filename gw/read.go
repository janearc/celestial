package gw

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/janearc/celestial/sky"
)

// The keys the centre's event format gives an event, and the suffixes it
// writes beside a value. Each is the format's own.
const (
	nameKey       = "commonName"
	versionKey    = "version"
	catalogKey    = "catalog.shortName"
	gpsKey        = "GPS"
	parametersKey = "parameters"
	pipelineKey   = "pipeline_type"
	preferredKey  = "is_preferred"
	estimation    = "pe"
	lowerSuffix   = "_lower"
	upperSuffix   = "_upper"
	unitSuffix    = "_unit"
)

// fields retains JSON members in their original encoded form.
type fields map[string]json.RawMessage

// read decodes a member into its destination, leaving the destination unchanged
// when the member is absent or null.
func (object fields) read(key string, into any) error {
	written, held := object[key]
	if !held {
		return nil
	}
	if err := json.Unmarshal(written, into); err != nil {
		return fmt.Errorf("gw: %s: %w", key, err)
	}
	return nil
}

// readEvent prefers estimates on the event itself, then fills missing
// parameters from the preferred parameter-estimation record.
func readEvent(key string, object fields) (*Event, error) {
	event := Event{}
	var seconds *float64
	for name, into := range map[string]any{nameKey: &event.name,
		versionKey: &event.version, catalogKey: &event.catalog,
		gpsKey: &seconds} {
		if err := object.read(name, into); err != nil {
			return nil, err
		}
	}
	if event.name == "" || seconds == nil {
		return nil, fmt.Errorf("gw: %s has no name or gps time", key)
	}
	at, err := sky.FromGPS(*seconds)
	if err != nil {
		return nil, fmt.Errorf("gw: %s: %w", key, err)
	}
	event.at = *at
	if event.estimates, err = readEstimates(object); err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	if err := event.estimatedFrom(object); err != nil {
		return nil, err
	}
	return &event, nil
}

// estimatedFrom fills parameters absent from the event’s own estimates using
// its preferred parameter-estimation record.
func (event *Event) estimatedFrom(object fields) error {
	var pipelines map[string]fields
	if err := object.read(parametersKey, &pipelines); err != nil {
		return err
	}
	for _, pipeline := range pipelines {
		var kind string
		var preferred bool
		if err := pipeline.read(pipelineKey, &kind); err != nil {
			return err
		}
		if err := pipeline.read(preferredKey, &preferred); err != nil {
			return err
		}
		if kind != estimation || !preferred {
			continue
		}
		more, err := readEstimates(pipeline)
		if err != nil {
			return fmt.Errorf("%s: %w", event, err)
		}
		for name, estimate := range more {
			if _, held := event.estimates[name]; !held {
				event.estimates[name] = estimate
			}
		}
	}
	return nil
}

// readEstimates reads values with accompanying unit fields and any reported
// credible-interval offsets.
func readEstimates(object fields) (map[string]*Estimate, error) {
	estimates := map[string]*Estimate{}
	for key := range object {
		parameter, valued := strings.CutSuffix(key, unitSuffix)
		if !valued {
			continue
		}
		estimate, err := readEstimate(object, parameter)
		if err != nil {
			return nil, err
		}
		if estimate != nil {
			estimates[parameter] = estimate
		}
	}
	return estimates, nil
}

// nothing is no reach at all. Defined by the scale.
const nothing = 0.0

// readEstimate converts a value, bounds, and unit to an SI estimate. Blank
// values remain absent. The centre reports bounds as signed offsets, with the
// lower offset nonpositive.
//
// A missing unit, unknown conversion, or bound on the wrong side returns an
// error.
func readEstimate(object fields, parameter string) (*Estimate, error) {
	var value, lower, upper *float64
	var unit *string
	for key, into := range map[string]any{parameter: &value,
		parameter + lowerSuffix: &lower,
		parameter + upperSuffix: &upper,
		parameter + unitSuffix:  &unit} {
		if err := object.read(key, into); err != nil {
			return nil, err
		}
	}
	if value == nil {
		return nil, nil
	}
	if unit == nil {
		return nil, fmt.Errorf("gw: %s has a value and no unit",
			parameter)
	}
	conversion, err := toSI(*unit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", parameter, err)
	}
	estimate := Estimate{unit: conversion.unit,
		value: *value * conversion.factor}
	if lower == nil || upper == nil {
		return &estimate, nil
	}
	// The centre writes each bound as an offset from the value, the lower
	// one below nothing; a reach is how far the interval runs each way
	reachBelow, reachAbove := -*lower, *upper
	if reachBelow < nothing || reachAbove < nothing {
		return nil, fmt.Errorf("gw: %s reaches %v below and %v above, "+
			"and neither may be below %v", parameter, reachBelow,
			reachAbove, nothing)
	}
	estimate.below = reachBelow * conversion.factor
	estimate.above = reachAbove * conversion.factor
	estimate.bounded = true
	return &estimate, nil
}
