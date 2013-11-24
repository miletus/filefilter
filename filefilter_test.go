// Tests for the filefilter package

package filefilter

import (
	"bytes"
	//	"fmt"
	"testing"
)

// The number of times the default match handler has been called since the count was zeroed.
var mhCount int

// The indicies discoverd by the default match handler.
var mhi []int

// The text buffer
var buffer []byte

// The default match handler function.
var dmh = func(indicies []int, text []byte) (outText []byte, err error) {
	mhi = append(mhi, indicies...)
	mhCount++
	return text[indicies[0]:indicies[1]], nil
}
var panicHandler = func() {
	if e := recover(); e != nil {
	}
}

func initVars() {
	mhCount = 0
	mhi = make([]int, 0, 100)
	buffer = nil
}

// Test_NewLineSource_0bytes tests whether NewLineSource panics when asked to create a LineSource with a 0 byte buffer.
func Test_Filter_Next(t *testing.T) {
	defer panicHandler()
	var testString string
	var match *FilterMatch

	// Regular expression will match a single line.
	f, err := NewFilter("(?m)^.*\n", dmh, "\n")

	// nil buffer
	initVars()
	match, err = f.Next(buffer)
	if match != nil && len(match.span) == 0 && mhCount == 0 {
		t.Error("Filter.Next with an empty buffer found something")
	}
	if err != nil {
		t.Errorf("Filter.Next with an empty buffer returned an error:%s", err)
	}

	// Empty buffer
	initVars()
	buffer = make([]byte, 0)
	match, err = f.Next(buffer)
	if match != nil && len(match.span) == 0 && mhCount == 0 {
		t.Error("Filter.Next with an empty buffer found something")
	}
	if err != nil {
		t.Errorf("Filter.Next with an empty buffer returned an error:%s", err)
	}

	// Only line ending
	initVars()
	testString = "\n"
	buffer = []byte(testString)
	match, err = f.Next(buffer)
	if bytes.Compare(match.textOut, []byte(testString)) != 0 {
		t.Error("Match handler did not match input!")
	}
	if mhCount != 1 || match.span[0] != 0 || match.span[1] != len(testString) {
		t.Error("Incorrect indicies returned")
	}

	// Buffer without line ending
	initVars()
	testString = "test string"
	buffer = []byte(testString)
	// Filter.Next call will add an eol to the very end of the buffer.
	testString += "\n"
	match, err = f.Next(buffer)
	if match != nil && bytes.Compare(match.textOut, []byte(testString)) != 0 {
		t.Error("Match handler did not match input!")
	}
	if match != nil && mhCount == 1 && match.span[0] == 0 && match.span[1] == len(testString) {
		t.Error("Incorrect indicies returned")
	}

	// Buffer with line ending
	initVars()
	testString = "test string\n"
	buffer = []byte(testString)
	match, err = f.Next(buffer)
	if bytes.Compare(match.textOut, []byte(testString)) != 0 {
		t.Error("Match handler did not match input!")
	}
	if mhCount != 1 || match.span[0] != 0 || match.span[1] != len(testString) {
		t.Error("Incorrect indicies returned")
	}

	// Buffer with more text past line ending
	initVars()
	testStringMatch := "test string\n"
	testStringFull := testStringMatch + "More text\n\n"
	buffer = []byte(testStringFull)
	match, err = f.Next(buffer)
	if bytes.Compare(match.textOut, []byte(testStringMatch)) != 0 {
		t.Error("Match handler did not match input!")
	}
	if mhCount != 1 || match.span[0] != 0 || match.span[1] != len(testStringMatch) {
		t.Error("Incorrect indicies returned")
	}
}

// // Various strings used in the test to initialize an io.Reader object.
// var crlf []byte = []byte("\r\n")
// var backslashn []byte = []byte("\n")
// var empty []byte = []byte("")
// var one []byte = []byte("1")
// var onecrlf []byte = []byte("1\r\n")
// var three []byte = []byte("012")
// var threecrlf []byte = append(three, crlf...)

// // Two batches of 5
// var five2 []byte = []byte("01234\r\n56789")
// var longString []byte = []byte("aldkjfeiojf\r\nIdivoasdIU*)#HGleSDJgbn\r\n!J0gjsvD)(FJG84LDGFJ ASE9R")

// // 29 lines
// var base29 []byte = []byte("1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14\n15\n16\n17\n18\n19\n20\n21\n22\n23\n24\n25\n26\n27\n28\n29\n")
// var windowsSep29 []byte = []byte("1\r\n2\r\n3\r\n4\r\n5\r\n6\r\n7\r\n8\r\n9\r\n10\r\n11\r\n12\r\n13\r\n14\r\n15\r\n16\r\n17\r\n18\r\n19\r\n21\r\n21\r\n22\r\n23\r\n24\r\n25\r\n26\r\n27\r\n28\r\n29\r\n")

// // Short line followed by long line.
// var shortLong []byte = []byte("123\r\n12345678901234567890")

// // Test_NewLineSource_0bytes tests whether NewLineSource panics when asked to create a LineSource with a 0 byte buffer.
// func Test_NewLineSource_0bytes(t *testing.T) {
// 	defer func() {
// 		if e := recover(); e != nil {
// 		}
// 	}()
// 	r := bytes.NewReader(five2)
// 	// Following call should cause panic.
// 	ls := NewLineSource(r, 0, crlf)
// 	ls.loadBuffer()
// 	t.Error("Did not panic with 0 byte buffer!")
// }

// // Test_NewLineSource_nilReader tests whether NewLineSource panics when passed a nil reader.
// func Test_NewLineSource_nilReader(t *testing.T) {
// 	defer func() {
// 		if e := recover(); e != nil {
// 		}
// 	}()
// 	// Following call should cause panic.
// 	ls := NewLineSource(nil, 1, crlf)
// 	ls.loadBuffer()
// 	t.Error("Did not panic with nil io.Reader!")
// }

// // Test_NewLineSource_noSeparator tests whether NewLineSource panics when passed an empty byte array for the separator.
// func Test_NewLineSource_noSeparator(t *testing.T) {
// 	defer func() {
// 		if e := recover(); e != nil {
// 		}
// 	}()
// 	r := bytes.NewReader(five2)
// 	// Following call should cause panic.
// 	ls := NewLineSource(r, 1, empty)
// 	ls.loadBuffer()
// 	t.Error("Did not panic with no separator!")
// }

// // Test_loadBuffer_emptyReader checks whether the loadBuffer behaves properly if the input io.Reader has 0 bytes.
// func Test_loadBuffer_emptyReader(t *testing.T) {
// 	r := bytes.NewReader(empty)
// 	ls := NewLineSource(r, 5, crlf)
// 	err := ls.loadBuffer()
// 	if err != EOF || len(ls.buffer) > 0 {
// 		t.Error("Reading from an empty reader did not return an EOF or buffer was not empty.")
// 	}
// 	// Should continue to get EOFs
// 	err = ls.loadBuffer()
// 	if err != EOF {
// 		t.Error("Reading from an empty reader 2nd time did not return an EOF or buffer was not empty.")
// 	}
// }

// // Test_loadBuffer_emptyReader checks whether the loadBuffer behaves properly if the input io.Reader has 1 bytes.
// func Test_loadBuffer_one(t *testing.T) {
// 	output := make([]byte, 0, 10)
// 	r := bytes.NewReader(one)
// 	ls := NewLineSource(r, 5, crlf)
// 	var err error
// 	if err = ls.loadBuffer(); err == EOF {
// 		t.Error("Reading from an one-byte reader returned premature EOF.")
// 	}
// 	// Load the buffer again. Have to trigger EOF before ls.sep will be added.
// 	ls.loadBuffer()
// 	// Read the buffer.
// 	output = append(output, ls.buffer...)
// 	ls.current = len(ls.buffer)
// 	// Buffer should have the one byte array with a crlf appended to it.
// 	onecrlf = append(one, crlf...)
// 	if bytes.Equal(onecrlf, output) == false {
// 		t.Error("Error reading one-byte reader.")
// 	}
// 	// Load the buffer again to make sure we got an EOF -> adding ls.sep.
// 	if err = ls.loadBuffer(); err != EOF {
// 		t.Error("Second read should have returned EOF.")
// 	}
// 	// Should continue to get EOFs
// 	err = ls.loadBuffer()
// 	if err != EOF {
// 		t.Error("Reading from an empty reader 2nd time did not return an EOF or buffer was not empty.")
// 	}
// }

// // Test_loadBuffer_splitSep calls loadBuffer with different size buffers so that it takes
// // one and then two reads to get all the data.
// func Test_loadBuffer_splitSep(t *testing.T) {
// 	input := three
// 	output := make([]byte, 0, len(input)+len(crlf))
// 	for bufferSize := 2; bufferSize <= 5; bufferSize++ {
// 		r := bytes.NewReader(input)
// 		ls := NewLineSource(r, bufferSize, crlf)
// 		// Read all the bytes
// 		for ls.loadBuffer() != EOF {
// 			output = append(output, ls.buffer...)
// 			ls.current += len(ls.buffer)
// 		}
// 		// Did we get what we should?
// 		if bytes.Equal(output, threecrlf) == false {
// 			t.Errorf("No match for buffer size = %d", bufferSize)
// 		}
// 		// Clear out output and do it again.
// 		output = output[0:0]
// 	}
// }

// // Test_loadBuffer_differentChunks calls loadBuffer with different chunks of bytes read.
// func Test_loadBuffer_differentChunks(t *testing.T) {
// 	input := longString
// 	output := make([]byte, 0, len(input)+len(crlf))
// 	r := bytes.NewReader(input)
// 	ls := NewLineSource(r, 3, crlf)
// 	// i is the chunk size.
// 	var i int = 0
// 	// Each time through the loop extract i bytes from ls.buffer
// 	// and save them into output. Then call loadBuffer.
// 	// Vary the number of bytes extracted from 0 to len(ls.buffer)
// 	for err := ls.loadBuffer(); err != EOF; err = ls.loadBuffer() {
// 		// Check i against len(ls.buffer) both because i will be incremented
// 		// each time through the loop and because last read may have shortened ls.buffer.
// 		//		fmt.Printf("len(ls.buffer)=%d\n", len(ls.buffer))
// 		if i > len(ls.buffer) {
// 			i = len(ls.buffer)
// 		}
// 		output = append(output, ls.buffer[:i]...)
// 		ls.current += i
// 		i++
// 	}
// 	// Output should have a separator at the end
// 	if bytes.HasSuffix(output, ls.sep) == false {
// 		t.Error("Output did not have separator at the end!")
// 	}
// 	// The output should match the input except perhaps for the line
// 	// separator at the end of the output
// 	if bytes.HasSuffix(input, ls.sep) == false {
// 		// Remove the separator from the output so can compare
// 		output = output[:len(output)-len(ls.sep)]
// 	}
// 	if bytes.Equal(input, output) == false {
// 		t.Error("differentChunks - input and output were not equal!")
// 	}
// }

// // Test_loadBuffer_differentChunksn calls loadBuffer with different chunks of bytes read. The separator
// // is \n.
// func Test_loadBuffer_differentChunksn(t *testing.T) {
// 	input := longString
// 	output := make([]byte, 0, len(input)+len(backslashn))
// 	r := bytes.NewReader(input)
// 	ls := NewLineSource(r, 3, backslashn)
// 	// i is the chunk size.
// 	var i int = 0
// 	// Each time through the loop extract i bytes from ls.buffer
// 	// and save them into output. Then call loadBuffer.
// 	// Vary the number of bytes extracted from 0 to len(ls.buffer)
// 	for err := ls.loadBuffer(); err != EOF; err = ls.loadBuffer() {
// 		// Check i against len(ls.buffer) both because i will be incremented
// 		// each time through the loop and because last read may have shortened ls.buffer.
// 		//		fmt.Printf("len(ls.buffer)=%d\n", len(ls.buffer))
// 		if i > len(ls.buffer) {
// 			i = len(ls.buffer)
// 		}
// 		output = append(output, ls.buffer[:i]...)
// 		ls.current += i
// 		i++
// 	}
// 	// Output should have a separator at the end
// 	if bytes.HasSuffix(output, ls.sep) == false {
// 		t.Error("Output did not have separator at the end!")
// 	}
// 	// The output should match the input except perhaps for the line
// 	// separator at the end of the output
// 	if bytes.HasSuffix(input, ls.sep) == false {
// 		// Remove the separator from the output so can compare
// 		output = output[:len(output)-len(ls.sep)]
// 	}
// 	if bytes.Equal(input, output) == false {
// 		t.Error("differentChunks - input and output were not equal!")
// 	}
// }

// // Test_bracketLines_matchSplit should give the same results as a call to bytes.Split.
// func Test_bracketLines_matchSplit(t *testing.T) {
// 	input := windowsSep29
// 	r := bytes.NewReader(input)
// 	// Make sure there's room for all of input in the LineSource's buffer.
// 	ls := NewLineSource(r, len(input), crlf)
// 	// Ask for enough lines.
// 	lineLocs, err := ls.bracketLines(len(input))
// 	if err != nil {
// 		t.Errorf("bracketLines returned a non-nil error equal to %s", err)
// 	}
// 	// use bytes.Split but remove the final separator. Otherwise Splits will
// 	// add a blank line after it.
// 	splits := bytes.Split(input[:len(input)-len(crlf)], crlf)
// 	if len(splits) != len(lineLocs)-1 {
// 		t.Errorf("len(splits) = %d, len(lineLocs) = %d", len(splits), len(lineLocs))
// 	}
// 	// Compare the characters from Split with those from LineSource.
// 	for i, line := range splits {
// 		if i+1 > len(lineLocs) {
// 			break
// 		}
// 		if bytes.Equal(line, input[lineLocs[i]:lineLocs[i+1]-len(ls.sep)]) == false {
// 			t.Errorf("%s from Split does not match %s from ls.", line, input[lineLocs[i]:lineLocs[i+1]-len(ls.sep)])
// 		}
// 	}
// }

// // Test_bracketLines_emptyReader should give no lines for an empty reader
// func Test_bracketLines_emptyReader(t *testing.T) {
// 	input := empty
// 	r := bytes.NewReader(input)
// 	ls := NewLineSource(r, 5, backslashn)
// 	// How many lines do we get?
// 	lineLocs, err := ls.bracketLines(1)
// 	if err != nil && err != EOF {
// 		t.Errorf("bracketLines returned a non-nil error equal to %s", err)
// 	}
// 	if len(lineLocs) > 1 {
// 		// Note - lineLocs is actually nil in this case.
// 		t.Error("bracketLines should have less than 2 entries for an empty reader!")
// 	}
// }

// // Test_bracketLines_tooSmallBuffer makes sure an error is returned if there are too many
// // intra-line characters to fit in a buffer.
// func Test_bracketLines_tooSmallBuffer(t *testing.T) {
// 	input := shortLong
// 	r := bytes.NewReader(input)
// 	// Make sure there's not room for all of input in the LineSource's buffer.
// 	ls := NewLineSource(r, 5, crlf)
// 	// How many lines do we get?
// 	lineLocs, err := ls.bracketLines(3)
// 	// First call should return a single line with no errors and no EOF
// 	if err != nil && err != EOF {
// 		t.Errorf("bracketLines returned a non-nil error equal to %s", err)
// 	}
// 	if len(lineLocs) != 2 {
// 		// Note - lineLocs is actually nil in this case.
// 		t.Error("bracketLines should have 2 entries for an empty reader!")
// 	}
// 	// Step ls.current
// 	ls.current = lineLocs[1]
// 	// Try again. This time we should get an error
// 	lineLocs, err = ls.bracketLines(3)
// 	if len(lineLocs) > 1 || err != ErrShortBuffer {
// 		t.Errorf("len(lineLocs) = %d, err = %s.", len(lineLocs), err)
// 	}
// }

// // Test_bracketLines_multipleLoads will test when mutiple loadBuffers are needed to
// // find all the lines.
// func Test_bracketLines_multipleLoads(t *testing.T) {
// 	input := make([]byte, 0, 100)
// 	// Set up an input with each line having 7 characters
// 	lineSize := 7
// 	input = []byte("01231\r\n01232\r\n01233\r\n01234\r\n0")
// 	sep := crlf
// 	// Choose a buffer size so that the last chunk fits with a little extra space. Then
// 	// start growing the end so it will take another buffer to fit.
// 	bufferSize := lineSize*2 + 2 + len(sep)
// 	for i := 0; i < 2; i++ {
// 		ls := NewLineSource(bytes.NewReader(input), bufferSize, sep)
// 		lineLocs, err := ls.bracketLines(5)
// 		if err != nil {
// 			t.Error("Non-nil error on first read!")
// 		}
// 		if lineLocs[1] != 7 && lineLocs[2] != 14 {
// 			t.Error("First bracket is off!")
// 		}
// 		ls.current += lineLocs[2]
// 		lineLocs, err = ls.bracketLines(5)
// 		if err != nil {
// 			t.Error("Non-nil error on second read!")
// 		}
// 		if lineLocs[1] != 7 && lineLocs[2] != 14 {
// 			t.Error("Second bracket is off!")
// 		}
// 		ls.current += lineLocs[2]
// 		lineLocs, err = ls.bracketLines(5)
// 		if err != nil {
// 			t.Error("Non-nil error on third read!")
// 		}
// 		if lineLocs[1] != i+3 {
// 			t.Error("Third bracket is off!")
// 		}
// 		input = append(input, []byte("a")...)
// 	}
// }

// // Test_GetLines will make sure the correct number of lines are read.
// func Test_GetLines(t *testing.T) {
// 	var nReturned int
// 	var err error
// 	var nLines int
// 	ls := NewLineSource(bytes.NewReader(base29), 10, backslashn)
// 	for ; err != EOF; _, nReturned, _, err = ls.GetLines(30) {
// 		nLines += nReturned
// 	}
// 	if nLines != 29 {
// 		t.Errorf("Total number of lines =%d. It should be 29.", nLines)
// 	}
// }

// // Test_GetLines_offset will check the returned offset.
// func Test_GetLines_offset(t *testing.T) {
// 	input := base29
// 	var nReturned int
// 	var err error
// 	var nLines int
// 	var offset int
// 	ls := NewLineSource(bytes.NewReader(input), 10, backslashn)
// 	// First GetLines should return an offset of 0.
// 	_, nReturned, offset, err = ls.GetLines(30)
// 	if offset != 0 {
// 		t.Errorf("First offset should have been 0. Instead it was %d", offset)
// 	}
// 	// Next call to GetLines should return an offset equal to the buffer size.
// 	_, nReturned, offset, err = ls.GetLines(30)
// 	if offset != len(ls.buffer) {
// 		t.Errorf("Second offset should have been %d. Instead it was %d", len(ls.buffer), offset)
// 	}
// 	// Read the rest of the data.
// 	for ; err != EOF; _, nReturned, offset, err = ls.GetLines(30) {
// 		nLines += nReturned
// 	}
// 	if (offset != len(input)) || (offset != ls.nReadTotal) {
// 		t.Errorf("After reading everything offset should be the number of bytes read:%d. Instead offset is %d, %d", len(input), offset, ls.nReadTotal)
// 	}
// }

// // Test_SkipBytes tries to skip the entire input []byte.
// func Test_SkipBytes(t *testing.T) {
// 	input := base29
// 	ls := NewLineSource(bytes.NewReader(input), 5, backslashn)
// 	// Skip all the bytes.
// 	nSkipped, err := ls.SkipBytes(len(input) + 1)
// 	// Did we skip everything?
// 	if nSkipped != len(input) {
// 		t.Errorf("Should have skipped %d bytes. Instead skipped %d", len(input), nSkipped)
// 	}
// 	// Since asked to skip an extra byte beyond what could be skipped, err should be EOF
// 	if err != EOF {
// 		t.Errorf("err should be EOF, but instead its %s", err)
// 	}
// }
