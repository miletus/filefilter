// filefilter will process a file (or any io.Reader) by applying regular expression based "filters"
// to the file.
// filefilter has several advantages over simply applying a regular expression replace method to a buffer.
// 1) Multiple regular expressions can be applied. The outputs will be in order of the location in the input
//    text that was matched.
// 2) The function for processing the input text can be arbitrary.
// 3) All of the input text does not have to fit into memory.
// 4) filefilter can determine the new line characters from the input text.
//A filefilter.Filter has both the regular expression and the code to process matches to the
// regex.
// The ProcessText method takes a text source, breaks it into chunks and passes each chunk to applyFilters.
// applyFilters matches each Filter object against the chunk and writes the text from Filter with the
// earliest match.
package filefilter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// NewLineChars are the new line (or end-of-line) characters that can be recognized by the
// newLine-finding routine, FindNewLineChar
// From Wikipedia:
// The Unicode standard defines a large number of characters that conforming applications should recognize as line terminators:[4]
// LF:    Line Feed, U+000A
// VT:    Vertical Tab, U+000B
// FF:    Form Feed, U+000C
// CR:    Carriage Return, U+000D
// CR+LF: CR (U+000D) followed by LF (U+000A)
// NEL:   Next Line, U+0085
// LS:    Line Separator, U+2028
// PS:    Paragraph Separator, U+2029
// FindNewLineChar will also recognize CR-LF as a new line.
const NewLineChars = "\u000a\u000b\u000c\u000d\u0085\u2028\u2029"

// defaultNewLine is used if no newLine is specified or detected in the input text.
var defaultNewLine string = "\n"

// ErrShortBuffer means that the internal buffer is too small to hold a complete line from the reader.
var ErrShortBuffer error = errors.New("Short Buffer")

// MatchHandler is a function used in a Filter object. The MatchHandler takes the match to the
// regular expression and produces output text. The indicies passed to the MatchHandler come from
// a Regexp.FindSubmatchIndex call in the Filter.
type MatchHandler func(indicies []int, text []byte) (outText []byte, err error)

// EchoMh is a MatchHandler whose outText is equal to the text that was matched with the
// regular expression.
var EchoMh = func(indicies []int, text []byte) (outText []byte, err error) {
	// Check the input.
	if len(indicies) < 2 || indicies[0] < 0 || indicies[0] > indicies[1] || indicies[1] > len(text) {
		return nil, errors.New("Invalid indicies argument!")
	}
	return text[indicies[0]:indicies[1]], nil
}

// Filter has a regular expression and the MatchHandler function to execute on the submatches or groups
// matched by that regular expression. See its Next method.
type Filter struct {
	nLines int            "The number of lines this regular expression will match."
	expr   string         "The string for the regular expression."
	re     *regexp.Regexp "The compiled regular expression."
	mh     MatchHandler   "The method that takes a regular expression submatch index list and does something with the data."
	newl   string         "The new line string used in the regular expression. For example \\n."
}

// FilterMatch is a structure used for returning a match from Filter.Next.
type FilterMatch struct {
	span    [2]int "span[0] is the beginning of the match and span[1] is one past the last byte matched."
	textOut []byte "The output text from Filter.mh"
}

// Next finds the first bytes that match with filter.regex.FindSubmatchIndex and then passes the match
// to filter.mh for processing. Next returns a FilterMatch structure with the beginning and end of the
// bytes matched by the regex and the text produced by the Filter's MatchHandler.
// If no match is found then the return will be nil.
func (filter *Filter) Next(text []byte) (*FilterMatch, error) {
	var err error
	// FindSubmatchIndex returns nil if there is no regex match.
	indicies := filter.re.FindSubmatchIndex(text)
	if len(indicies) > 1 {
		var filterMatch FilterMatch
		// The first and second indicies returned by FindSubmatchIndex are the beginning and end index of the match.
		copy(filterMatch.span[:], indicies[0:2])
		if filter.mh != nil {
			filterMatch.textOut, err = filter.mh(indicies, text)
		}
		return &filterMatch, err
	} else {
		return nil, nil
	}
}

// Replaces the Filter's newLine in the Filter's regular expression with newNewLine.
func (f *Filter) changeNewLine(newNewLine string) (err error) {
	// If f.newl is empty, that indicates there was no newLine in the regular expression.
	// Remember the newNewLine, but don't have to do any replacement.
	if newNewLine != f.newl && len(f.newl) > 0 {
		f.expr = strings.Replace(f.expr, f.newl, newNewLine, -1)
		// Compile the regular expression.
		f.re, err = regexp.Compile(f.expr)
	}
	f.newl = newNewLine
	return err
}

// NewFilter creates a new Filter object.
// expr is the regular expression string to use for matching.
// MatchHandler takes the output of a FindStringSubmatchIndex match and produces a []byte as output.
// newLine is the string that represents a new line in the regular expression. For example "\n" or "\r\n".
// The newLine should be the same as for the text that the Filter will be used on. newLine can be changed
// by calling changeNewLine. newLine can be an empty string if there is no newLine in the regular expression.
func NewFilter(expr string, mh MatchHandler, newLine string) (*Filter, error) {
	var f Filter
	var err error
	if len(expr) < 1 {
		return nil, errors.New("Empty regular expression string!")
	}
	f.expr = expr
	f.mh = mh
	f.newl = newLine
	// Count the number of lines in expr. The number of lines is equal to the number of newLines
	// plus 1 if there are characters past the last newLine. This works as long as the regular
	// expression doesn't include the s flag.
	if len(f.newl) > 0 {
		f.nLines = strings.Count(expr, f.newl)
		if strings.HasSuffix(expr, f.newl) == false {
			f.nLines += 1
		}
	} else {
		f.nLines = 1
	}
	// Compile the regular expression.
	f.re, err = regexp.Compile(f.expr)
	if IsVerbose() {
		// Warn the user if the regular expression contains the s flag.
		match, err1 := regexp.MatchString("\\(\\?[imU]*s[imU]*\\)", f.expr)
		if match && err1 == nil {
			fmt.Printf("Warning: The following regular expression contains the s flag: %s\n"+
				"It can match a variable number of lines. Use a buffer large enough to hold all the input text at once.\n",
				strconv.Quote(f.expr))
		}
	}
	return &f, err
}

// NewPassAllFilter creates a filter that matches a complete line and will return that line from Filter.Next.
// It's main use is as the default filter
func NewPassAllFilter() (*Filter, error) {
	return NewFilter("^.*\n", EchoMh, "\n")
}

// ProcessText will apply the filters to the input data buffer by buffer and write the result to the writer.
// reader is the source of the input text
// output The text output from the filters will be written to output.
// filters the Filters to apply to the text.
// defaultFilter will be applied if none of the others filters matches the section of the input that is currently
// being matched. defaultFilter can be nil.
// bufferSize is the size of the buffer in bytes to use for i/o.
// newLine is the newLine character(s) to user for marking the end of an old line or the beginning of a new one. If newLine
// is an empty string then ProcessText will determine the newLine character from the input text.
func ProcessText(reader io.Reader, output io.Writer, filters []*Filter, defaultFilter *Filter, bufferSize int64, newLine string) (err error) {
	// ------------- Verify Inputs ----------------
	// Check to make sure the input data is usable.
	if reader == nil || output == nil {
		return errors.New("nil input or output!")
	}
	if bufferSize <= 0 {
		return errors.New("bufferSize <= 0!")
	}
	if len(filters) == 0 && defaultFilter == nil {
		return errors.New("No filters!")
	}
	for _, filter := range filters {
		if filter == nil {
			return errors.New("Nil *Filter in list of filters!")
		}
	}

	// Save room in the input buffer for a newLine in case have to add one to the last buffer. The maximum size
	// for a utf8 character is 4 bytes. That's also enough for \r\n.
	buffer := make([]byte, bufferSize, bufferSize+4)
	// nUnprocessed is the number of bytes at the end of the buffer that were not processed by the
	// most recent call to applyFilters.
	nUnprocessed := 0
	firsttime := true
	for {
		// ------------ Read Data --------------
		// Move the unprocessed data to the beginning of the buffer
		copy(buffer[:], buffer[len(buffer)-nUnprocessed:])
		// Try to expand the buffer in case it was contracted earlier.
		buffer = buffer[:bufferSize]
		// read in the new data
		var nRead int
		nRead, err = reader.Read(buffer[nUnprocessed:])
		nUnprocessed += nRead
		// Shrink the buffer to fit the new data
		buffer = buffer[:nUnprocessed]

		// ------------ First buffer --------------
		if firsttime {
			// Check to see whether we've detected the line separator yet. If not then look through the buffer to find it.
			if newLine == "" {
				newLine, _ = FindNewLineChar(buffer)
			}
			// Make sure that all the filters have the correct newLine.
			for _, f := range filters {
				f.changeNewLine(newLine)
			}
			firsttime = false
		}

		// -------------- Last buffer ---------------------
		if err == io.EOF {
			// There are no more bytes to read. There can still be unprocessed data. If
			// there is, make sure it's terminated with a newLine.
			if nUnprocessed > 0 {
				// Make sure this last buffer is terminated with newLine
				if !bytes.HasSuffix(buffer[:nUnprocessed], []byte(newLine)) {
					// Room was saved at the end of the buffer during its creation.
					buffer = append(buffer, newLine...)
				}
				// One last go-round
				nUnprocessed, err = applyFilters(buffer, true, filters, defaultFilter, output, newLine)
			}
			return err

			// --------------- Error handling
		} else if err != nil {
			// Something went wrong with the read
			return err

			// -------------- Process buffer
		} else {
			nUnprocessed, err = applyFilters(buffer, false, filters, defaultFilter, output, newLine)
			if err != nil {
				return err
			}
		}
	}
}

// applyFilters steps the filters through the supplied buffer.
// The output bytes provided by the filters are written to output. The output is written in order of where
// the match happens on the input buffer, so the output of a filter whose match starts at buffer[100] will be written before
// a filter whose match starts at buffer[111].
// Filter matches can overlap. However a Filter match will not overlap itself. So if a Filter matches at line 1 and 2, the
// next opportunity to match starts at line 3.
// The defaultFilter's output is only taken if no other filter matches at a lower or the same buffer index. That way text not
// matched by other filters is written to the output. The expectation is that the defaultFilter matches a single line of text
// and reproduces that text as output. The defaultFilter can be nil.
// newLine is the newLine character(s) that will be added to the end of each match output.
func applyFilters(buffer []byte, isLastBuffer bool, filters []*Filter,
	defaultFilter *Filter, output io.Writer, newLine string) (int, error) {
	var err error
	// Can't process all the way to the end of the buffer until we're sure all the characters
	// have been read in. That only happens in the last buffer. Before then we'll return with some
	// of the characters unprocessed.
	unprocessedIndex := findProcessingLimit(buffer, isLastBuffer, filters, newLine)
	if unprocessedIndex <= 0 {
		// Don't have room in the buffer for the number of lines needed by a filter. Client should try again with a
		// larger buffer.
		return 0, ErrShortBuffer
	}
	// Keep track of where we are in the input buffer.
	searchOffset := 0
	// matches keeps track of the most recent match for each filter.
	matches := make([]*FilterMatch, len(filters))
	// Initialize matches with the initial result for each filter.
	for index, filter := range filters {
		if matches[index], err = nextMatch(buffer, filter, searchOffset); err != nil {
			return 0, err
		}
	}
	// Do the same for the defaultFilter
	var defaultMatch *FilterMatch
	if defaultMatch, err = nextMatch(buffer, defaultFilter, searchOffset); err != nil {
		return 0, err
	}
	// Keep running until the all filters have processed the buffer.
	for {
		// The next data to send to output comes from whichever filter matches the earliest in
		// the remaining input buffer. Initialize a variable to keep track of this lowest index.
		lowestMatchIndex := len(buffer)
		// Keep track of the index of the match with that lowest byte.
		matchesIndex := -1
		for i := range matches {
			if matches[i] != nil && matches[i].span[0] < lowestMatchIndex {
				lowestMatchIndex = matches[i].span[0]
				matchesIndex = i
			}
		}
		var winner FilterMatch
		if defaultMatch != nil && defaultMatch.span[0] < lowestMatchIndex && defaultMatch.span[0] < unprocessedIndex {
			// The default match is earlier. Use it.
			winner = *defaultMatch
		} else if lowestMatchIndex < unprocessedIndex {
			// Found a match in the buffer
			winner = *matches[matchesIndex]
			// If the verbose flag is set, write the match data to stdout
			if IsVerbose() {
				reportMatch(matches[matchesIndex], filters[matchesIndex], buffer)
			}
			// Get the next value from the filter. Use winner.span[1] as the starting index to make sure
			// the next match doesn't overlap the current one.
			if matches[matchesIndex], err = nextMatch(buffer, filters[matchesIndex], winner.span[1]); err != nil {
				return 0, err
			}
		} else {
			// No match by any filter. All done.
			return len(buffer) - unprocessedIndex, nil
		}
		// Update the search offset only if the end of the current match comes after the end of any
		// previous match. searchOffset is primarily used to tell where the defaultFilter should be applied.
		// We don't want to apply the defaultFilter to any text that has been matched, so the largest value
		// of the last byte matched should be used for searchOffset.
		if searchOffset < winner.span[1] {
			searchOffset = winner.span[1]
		}
		// Update the defaultMatch no matter whether we're using it or not.
		if defaultMatch, err = nextMatch(buffer, defaultFilter, searchOffset); err != nil {
			return 0, err
		}

		// Write the result
		if _, err := output.Write(winner.textOut); err == nil {
			// Make sure the output text from each match ends in a newLine. While this seems like a good idea now,
			// I may find out that there are cases where you don't want to do this.
			// If that happens then add a command line flag to control it.
			if len(winner.textOut) > 0 && bytes.HasSuffix(winner.textOut, []byte(newLine)) == false {
				output.Write([]byte(newLine))
			}

		} else {
			return 0, err
		}
	} // End of for {}
}

// findProcessingLimit finds the last index in the buffer that could be matched by all
// the filters. For example if you only have two lines left in the buffer but your
// regular expression matches 3 lines, then that regular expression will never match. But
// the next line read could cause the regular expression to match. The solution is to not accept
// any matches past the findProcessingLimit.
func findProcessingLimit(buffer []byte, isLastBuffer bool, filters []*Filter, newLine string) int {
	if isLastBuffer {
		// For the last buffer process all the bytes.
		return len(buffer)
	}

	// Don't want to allow a match past the index that corresponds to the last possible match of the biggest
	// filter. Biggest in this case means it matches the most lines. Find that index in buffer.
	// Note: This won't work if a regular expression uses the s flag.
	maxLines := 0
	for _, filter := range filters {
		if filter.nLines > maxLines {
			maxLines = filter.nLines
		}
	}
	// Now step back from the end of the buffer until have one complete line less than maxLines.
	limit := len(buffer)
	for i := 0; i < maxLines; i++ {
		limit = bytes.LastIndex(buffer[:limit], []byte(newLine))
		if limit == -1 {
			// Don't have enough lines in the buffer.
			return 0
		}
	}
	// limit points at the beginning of the newLine for the last line that can
	// be processesed. Lines should include their newLine, so increment limit.
	limit += len(newLine)
	return limit
}

// nextMatch takes a buffer and a filter and returns the results of filter.Next on buffer[offset:]
// If a match is found nextMatch will also add in offset to each item in match.span so the span
// values are indicies into the underlying buffer instead of indices into buffer[offset:].
func nextMatch(buffer []byte, filter *Filter, offset int) (match *FilterMatch, err error) {
	if filter == nil {
		// It's not an error to pass in a nil *Filter. It just means the match will be nil.
		return nil, nil
	}
	match, err = filter.Next(buffer[offset:])
	if err != nil {
		return nil, err
	}
	// Add the offset to the span values.
	if match != nil {
		for i := range match.span {
			match.span[i] += offset
		}
	}
	return match, err
}

// reportMatch will write information about a match to stdout to help in debugging regular expression
// matches. It will be called for each match found when the verbose flag is set.
func reportMatch(match *FilterMatch, filter *Filter, buffer []byte) {
	fmt.Printf("%q matches from index %d to %d\n", filter.expr, match.span[0], match.span[1])
	fmt.Printf("Input text  = %q\n", buffer[match.span[0]:match.span[1]])
	fmt.Printf("Output text = %q\n\n", match.textOut)
}

// Look through the buffer to find the newLine character. The discovered newLine is returned.
// If no standard newLines are found in buffer then filefilter.defaultNewLine will be returned.
func FindNewLineChar(buffer []byte) (sep string, nFound int) {
	// A map to keep track of the # of newLine characters.
	newLineCount := make(map[string]int, utf8.RuneCountInString(NewLineChars))
	// offset is the starting index in the buffer for the next newLine search
	// in the loop
	var offset int = 0
	// newLinesForDone is the number of newLines of any type to find before stopping the search.
	// Should find one newLine each time through the loop.
	const newLinesForDone = 10
	for nNewLines := 0; nNewLines < newLinesForDone; nNewLines++ {
		newLineOffset := bytes.IndexAny(buffer[offset:], NewLineChars)
		if newLineOffset == -1 {
			// Out of characters in buffer.
			break
		} else {
			offset += newLineOffset
		}
		// Check whether we have \r\n
		if offset > 0 && buffer[offset-1] == '\r' && buffer[offset] == '\n' {
			newLineCount["\r\n"] += 1
			offset += 2
		} else {
			r, size := utf8.DecodeRune(buffer[offset:])
			if r != utf8.RuneError {
				newLineCount[string(r)] += 1
				offset += size
			} else {
				// Something went wrong. Advance the offset to step past whatever it was.
				offset++
			}
		}
	}
	// Find the key with the biggest count
	nFound = 0
	for s, count := range newLineCount {
		if count > nFound {
			nFound = count
			sep = s
		}
	}
	if nFound > 0 {
		if IsVerbose() {
			fmt.Printf("%s chosen as line separator with %d/%d instances.\n", strconv.Quote(sep), nFound, newLinesForDone)
		}
	} else {
		// nFound == 0
		sep = defaultNewLine
		if IsVerbose() {
			fmt.Printf("No line NewLineChars found in buffer. Using default\n")
		}
	}
	return sep, nFound
}
