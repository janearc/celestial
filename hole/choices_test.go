//ninja:vow dojo a choices file that does not read, or that lacks a choice, stops the program saying why, through readMarching and choicesText, which hole does not export
package hole

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// readingOf is what readMarching stops with when it reads text in place of
// choices.json, or nil if it does not stop. choices.json is put back after.
func readingOf(test *testing.T, text string) (stopped any) {
	test.Helper()
	defer putBack(choicesText, &stopped)
	choicesText = []byte(text)
	readMarching()
	return nil
}

// putBack puts the built-in choices.json back, and keeps what
// readMarching stopped with, if it stopped. It is deferred, so recover
// here catches the stop.
func putBack(builtIn []byte, stopped *any) {
	choicesText = builtIn
	*stopped = recover()
}

// A choices file that is not json stops the program with json's own
// error, wrapped.
func TestChoicesThatAreNotJSONStopTheProgram(t *testing.T) {
	stopped := readingOf(t, innerNumbers(t).Choices.Broken)
	err, isError := stopped.(error)
	var syntax *json.SyntaxError
	if !isError || !errors.As(err, &syntax) {
		t.Errorf("choices that are not json stop with %v", stopped)
	}
}

// The anti-test: a choices file that is json but lacks a choice stops the
// program naming the choice, and not with a json error.
func TestChoicesThatLackAChoiceStopTheProgramNamingIt(t *testing.T) {
	stopped := readingOf(t, innerNumbers(t).Choices.Empty)
	err, isError := stopped.(error)
	var syntax *json.SyntaxError
	if !isError || errors.As(err, &syntax) ||
		!strings.Contains(err.Error(), paceName) {
		t.Errorf("choices with none in them stop with %v", stopped)
	}
}
