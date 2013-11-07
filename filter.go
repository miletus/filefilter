// filefilter will process a file (or any io.Reader) by applying regular expression based "filters"
// to the file. A filefilter.Filter has both the regular expression and the code to process matches to the
// regex.
// The ProcessText method takes an data source, breaks it into chunks and passes each chunk to applyFilters.
// applyFilters matches each Filter object against the chunk and writes the text from Filter with the
// earliest match.
package filefilter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// lineSeparator in the input data. It can be changed with SetLineSeparator. Make sure
// the lineSeparator is set before creating the Filters.
var lineSeparator string = "\n"

// defaultLineSeparator should be used in the regular expressions. NewFilter
// will convert all defaultLineSeparators to lineSeparators in the regular expression.
var defaultLineSeparator string = "\n"

// Change the lineSeparator used by the homerLog routines. This must be called before you
// create your Filters.
func SetLineSeparator(newLineSeparator string) {
	lineSeparator = newLineSeparator
}

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
	NLines int            "The number of lines this regular expression will match."
	expr   string         "The string for the regular expression."
	re     *regexp.Regexp "The compiled regular expression."
	mh     MatchHandler   "The method that takes a regular expression submatch index list and does something with the data."
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

// NewFilter creates a new Filter object.
// expr is the regular expression string to use for matching. Use defaultLineSeparator as the line ending
// in the regular expression string. They will be replaced by lineSeparator before the regular expression is applied.
// MatchHandler takes the output of a FindStringSubmatchIndex match and produces a []byte as output.
func NewFilter(expr string, mh MatchHandler) (*Filter, error) {
	var f Filter
	var err error
	if len(expr) < 1 {
		panic("Empty regular expression string ")
	}
	f.mh = mh
	// Count the number of lines in expr. The number of lines is equal to the number of defaultLneSeparators
	// plus 1 if there are characters past the last defaultLineSeparator. This works as long as the regular
	// expression doesn't include the s flag.
	f.NLines = strings.Count(expr, string(defaultLineSeparator))
	if strings.HasSuffix(expr, string(defaultLineSeparator)) == false {
		f.NLines += 1
	}
	// If the lineSeparator is not empty and different from the defaultLineSeparator
	// then replace the defaultLineSeparator in expr with lineSeparator.
	if len(lineSeparator) > 0 && lineSeparator != defaultLineSeparator {
		f.expr = strings.Replace(expr, defaultLineSeparator, lineSeparator, -1)
	} else {
		// Use the input expr
		f.expr = expr
	}

	// Compile the regular expression.
	f.re, err = regexp.Compile(f.expr)
	return &f, err
}

// NewPassAllFilter creates a filter that matches a complete line and will return that line from Filter.Next.
// It's main use is as the default filter
func NewPassAllFilter() (*Filter, error) {
	return NewFilter("^.*\n", EchoMh)
}

// ProcessText will apply the filters to the input data buffer by buffer and write the result to the writer.
// reader is the source of the input text
// output The text output from the filters will be written to output.
// filters the Filters to apply to the text.
// defaultFilter will be applied if none of the others filters matches the section of the input that is currently
// being matched. defaultFilter can be nil.
// bufferSize is the size of the buffer in bytes to use for i/o.
func ProcessText(reader io.Reader, output io.Writer, filters []*Filter, defaultFilter *Filter, bufferSize int64) (err error) {
	// Check to make sure the input data is usable.
	if bufferSize <= 0 {
		return errors.New("bufferSize <= 0!")
	}
	if len(filters) == 0 && defaultFilter == nil {
		return errors.New("No filters!")
	}
	for _, filter := range filters {
		if filter == nil {
			return errors.New("Nil *Filter in list of filters!")
		} else {
			if filter.NLines == 0 {
				return errors.New("Filter processes 0 lines!")
			}
		}
	}
	// Save room in the input buffer for a lineSeparator in case have to add one to the last buffer. In the future the
	// program may accept unicode line separators so save 4 bytes.
	buffer := make([]byte, bufferSize, bufferSize+4)
	// nUnprocessed is the number of bytes at the end of the buffer that were not processed by the
	// most recent call to applyFilters.
	nUnprocessed := 0
	for {
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
		if err == io.EOF {
			// There are no more bytes to read. There can still be unprocessed data. If
			// there is, make sure it's terminated with a lineSeparator.
			if nUnprocessed > 0 {
				// Make sure this last buffer is terminated with lineSeparator
				if !bytes.HasSuffix(buffer[:nUnprocessed], []byte(lineSeparator)) {
					// Room was saved at the end of the buffer during its creation.
					buffer = append(buffer, lineSeparator...)
				}
				// One last go-round
				nUnprocessed, err = applyFilters(buffer, true, filters, defaultFilter, output)
			}
			return err
		} else if err != nil {
			// Something went wrong with the read
			return err
		} else {
			nUnprocessed, err = applyFilters(buffer, false, filters, defaultFilter, output)
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
func applyFilters(buffer []byte, isLastBuffer bool, filters []*Filter,
	defaultFilter *Filter, output io.Writer) (int, error) {
	var err error
	// Can't process all the way to the end of the buffer until we're sure all the characters
	// have been read in. That only happens in the last buffer. Before then we'll return with some
	// of the characters unprocessed.
	unprocessedIndex := findProcessingLimit(buffer, isLastBuffer, filters)
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
			// Get the next value from the filter. Use winner.span[1] as the staring index to make sure
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
		if _, err := output.Write(winner.textOut); err != nil {
			return 0, err
		}
	} // End of for {}
}

// findProcessingLimit finds the last index in the buffer that could be matched by all
// the filters. For example if you only have two lines left in the buffer but your
// regular expression matches 3 lines, then that regular expression will never match. But
// there could be another line that hasn't been read in yet. The solution is to not accept
// any matches past the findProcessingLimit.
func findProcessingLimit(buffer []byte, isLastBuffer bool, filters []*Filter) int {
	if isLastBuffer {
		// For the last buffer process all the bytes.
		return len(buffer)
	}

	// Don't want to allow a match past the index that corresponds to the last possible match of the biggest
	// filter. Biggest in this case means it matches the most lines. Find that index in buffer.
	maxLines := 0
	for _, filter := range filters {
		if filter.NLines > maxLines {
			maxLines = filter.NLines
		}
	}
	// Now step back from the end of the buffer until have one complete line less than maxLines.
	limit := len(buffer)
	for i := 0; i < maxLines; i++ {
		limit = bytes.LastIndex(buffer[:limit], []byte(lineSeparator))
		if limit == -1 {
			// Don't have enough lines in the buffer.
			return 0
		}
	}
	// limit points at the beginning of the lineSeparator for the last line that can
	// be processesed. Lines should include their lineSeparator, so increment limit.
	limit += len(lineSeparator)
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
