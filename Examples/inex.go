// inex is a command line program that uses regular expression matches to include or exclude text from the input file.

package main

import (
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

	filters, defaultFilter, err := buildFilters()
	if err != nil {
		log.Fatal(err)
	}
	err = filefilter.ProcessText(infile, outfile, filters, defaultFilter, filefilter.GetBufferSize())
	if err != nil {
		log.Fatal(err)
	}
}

// Build the filters for processing the input file. Use the regular expressions from the command line.
func buildFilters() (filters []*filefilter.Filter, defaultFilter *filefilter.Filter, err error) {

	// Set the line separator
	filefilter.SetLineSeparator(filefilter.GetEol())
	// The include filters process the matched data with the echo match handler.
	filters, err = loadFilters(filefilter.GetIncludes(), filefilter.EchoMh)
	if err != nil {
		return nil, nil, err
	}

	// The exclude filters don't process the matched data - the data is skipped. That's why they have a nil handler.
	excludeFilters, err := loadFilters(filefilter.GetExcludes(), nil)
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
func loadFilters(regexes []string, mh filefilter.MatchHandler) ([]*filefilter.Filter, error) {
	filters := make([]*filefilter.Filter, len(regexes))
	var err error
	for i, regex := range regexes {
		if filters[i], err = filefilter.NewFilter(regex, mh); err != nil {
			return nil, err
		}
	}
	return filters, nil
}
