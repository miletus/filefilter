// inex is a command line program that uses regular expression matches to include or exclude text from the input file.

package main

import (
	"bytes"
	"filefilter"
	"fmt"
	"log"
	"os"
)

func init() {
}

func main() {
	var err error
	filefilter.ProcessFlags()
	infile, outfile, err := filefilter.GetFilesFromCommandLine()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer infile.Close()
	defer outfile.Close()

	// Find the newline in the filter regex strins
	newLine := findFilterNewline()

	filters, defaultFilter, err := buildFilters(newLine)
	if err != nil {
		log.Fatal(err)
	}
	// Do not pass the newLine found in the regex strings. That newLine could be different from the one in the input text.
	// Instead pass the newLine from the command line.
	err = filefilter.ProcessText(infile, outfile, filters, defaultFilter, filefilter.GetBufferSize(), filefilter.GetNewLine())
	if err != nil {
		log.Fatal(err)
	}
}

// findFilterNewline will find the newLine character used by the include and exclude regular expression strings.
func findFilterNewline() (newLine string) {
	// If a newLine character(s) has been specified on the command line, use it.
	if newLine = filefilter.GetNewLine(); newLine != "" {
		return newLine
	}
	// No newLine character specified. Look at the regex strings to find a newLine.
	// Put the include and exclude strings together in a byte buffer.
	var all bytes.Buffer
	for _, s := range filefilter.GetIncludes() {
		all.WriteString(s)
	}
	for _, s := range filefilter.GetExcludes() {
		all.WriteString(s)
	}
	// Now that we have all the regexes in one buffer, find the newline
	newLine, _ = filefilter.FindNewLineChar(all.Bytes())
	return newLine
}

// buildFilters builds the filters for processing the input file. Use the regular expressions from the command line.
func buildFilters(newLine string) (filters []*filefilter.Filter, defaultFilter *filefilter.Filter, err error) {
	// The include filters process the matched data with the echo match handler.
	filters, err = loadFilters(filefilter.GetIncludes(), filefilter.EchoMh, newLine)
	if err != nil {
		return nil, nil, err
	}

	// The exclude filters don't process the matched data - the data is skipped. That's why they have a nil handler.
	excludeFilters, err := loadFilters(filefilter.GetExcludes(), nil, newLine)
	if err != nil {
		return nil, nil, err
	}
	filters = append(filters, excludeFilters...)

	// Include filters have no defaultFilter because they only write what is matched in the include filters.
	// So start with a nil defaultFilter.
	defaultFilter = nil
	if len(filefilter.GetExcludes()) > 0 {
		// Exclude filters use a pass-all filter as the default.
		defaultFilter, err = filefilter.NewPassAllFilter()
		if err != nil {
			return nil, nil, err
		}
	}
	return filters, defaultFilter, nil
}

// loadFilter will combine a slice of regular expressions with a match handler to make a slice of Filters.
func loadFilters(regexes []string, mh filefilter.MatchHandler, newLine string) ([]*filefilter.Filter, error) {
	filters := make([]*filefilter.Filter, len(regexes))
	var err error
	for i, regex := range regexes {
		if filters[i], err = filefilter.NewFilter(regex, mh, newLine); err != nil {
			return nil, err
		}
	}
	return filters, nil
}
