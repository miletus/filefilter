// Tests for the filefilter package

package filefilter

import (
	"bytes"
	"testing"
)

// mhCount is the number of times the default match handler has been called since the count was zeroed.
var mhCount int

// mhi is the indicies discoverd by the default match handler.
var mhi []int

// buffer is the text buffer
var buffer []byte

// dmh is the default MatchHandler function
var dmh = func(indicies []int, text []byte) (outText []byte, err error) {
	mhi = append(mhi, indicies...)
	mhCount++
	return text[indicies[0]:indicies[1]], nil
}

// panicHandler is used when a test could throw a panic.
var panicHandler = func() {
	if e := recover(); e != nil {
	}
}

// initVars makes sure the standard variables are ready for the next test.
func initVars() {
	mhCount = 0
	mhi = make([]int, 0, 100)
	buffer = nil
}

// Test_NewLineSource_0bytes tests whether NewLineSource panics when asked to create a LineSource with a 0 byte buffer.
func Test_Filter_Next(t *testing.T) {
	var testString string
	var match *FilterMatch

	// Regular expression will match a single line.
	f, err := NewFilter("(?m)^.*\n", dmh, "\n")

	// nil buffer
	initVars()
	match, err = f.Next(buffer)
	if match != nil || mhCount != 0 || err != nil {
		t.Error("Filter.Next with a nil buffer found something or returned an error.")
	}

	// Empty buffer
	initVars()
	buffer = make([]byte, 0)
	match, err = f.Next(buffer)
	if match != nil || mhCount != 0 || err != nil {
		t.Error("Filter.Next with an empty buffer found something or had an error.")
	}

	// Only line ending
	initVars()
	testString = "\n"
	buffer = []byte(testString)
	match, err = f.Next(buffer)
	if match == nil || bytes.Compare(match.textOut, []byte(testString)) != 0 || err != nil {
		t.Error("Match handler did not match input or returned an error!")
	} else if mhCount != 1 || match.span[0] != 0 || match.span[1] != len(testString) {
		t.Error("Incorrect indicies returned")
	}

	// Buffer with line ending
	initVars()
	testString = "test string\n"
	buffer = []byte(testString)
	match, err = f.Next(buffer)
	if match == nil || bytes.Compare(match.textOut, []byte(testString)) != 0 || err != nil {
		t.Error("Match handler did not match input or returned an error!")
	} else if mhCount != 1 || match.span[0] != 0 || match.span[1] != len(testString) {
		t.Error("Incorrect indicies returned")
	}

	// Buffer with more text past line ending
	initVars()
	testStringMatch := "test string\n"
	testStringFull := testStringMatch + "More text\n\n"
	buffer = []byte(testStringFull)
	match, err = f.Next(buffer)
	if match == nil || bytes.Compare(match.textOut, []byte(testStringMatch)) != 0 || err != nil {
		t.Error("Match handler did not match input or returned an error!")
	} else if mhCount != 1 || match.span[0] != 0 || match.span[1] != len(testStringMatch) {
		t.Error("Incorrect indicies returned")
	}
}
