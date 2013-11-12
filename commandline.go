// filefilter uses regular expressions to transform input files to output files.
// This file has command line processing for both flags and input/output files.
// ProcessFlags should be called at the beginning of main() in order to process the command line.
package filefilter

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
)

// regexs is a slice of regular expression strings.
type regexes []string

// includeFlag is a slice of regular expressions as strings. Text in the input that matches the regular expressions will be
// copied to the output.
var includeFlag regexes

// excludeFlag is a slice of regular expressions as strings. All text that does NOT match the regular expressions will
// be copied to the output.
var excludeFlag regexes

// helpFlag=true means the user would like to see information on the program
var helpFlag bool

// verbose=true will cause debugging information to be written to stdout.
var verboseFlag bool

// sizeFlag is size of the in-memory buffer used for reading the file in kB.
var sizeFlag int64

// eolFlag is the end-of-line string specified on the command line.
var eolFlag string

// areFlagsProcessed keeps track of whether ProcessFlags has been called.
var areFlagsProcessed bool = false

func init() {
	// Tie the command-line flag to the flag variables.
	flag.Var(&includeFlag, "include", "Write text matching the specified regular expression to the output.")
	flag.Var(&includeFlag, "i", "Write text matching the specified regular expression to the output (short version).")
	flag.Var(&excludeFlag, "exclude", "Don't write text matching the specified regular expression to the output.")
	flag.Var(&excludeFlag, "e", "Don't write text matching the specified regular expression to the output (short version).")
	flag.BoolVar(&helpFlag, "help", false, "List the default usage and flags.")
	flag.BoolVar(&helpFlag, "h", false, "List the default usage and flags.")
	flag.Int64Var(&sizeFlag, "size", 100, "The size in KB of the memory buffer used for reading the file")
	flag.StringVar(&eolFlag, "eol", "", "The end-of-line string. If the eol flag is not used then the eol will be detected in the input text.")
	flag.BoolVar(&verboseFlag, "verbose", false, "Write information useful for debugging to stdout")
	name := path.Base(os.Args[0])
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage of %s:\n"+
				"%s [flags] [input file] [output file]\n"+
				"If input and/or output files are not given, stdin and stout are used.\n"+
				"-exclude and -include flags are followed by a quoted string with a single regular expression. \n"+
				"Use single quotes for the string on the command line.\n"+
				"Multiple -include and -exclude flags are allowed.\n"+
				"Set the actual line break that's in the input text with the -eol flag. If there is no eol flag the program will detect "+
				"the newline used in the text.\n"+
				"You can embed regular expression processing flags in your expression. '(?i)USA' will match with case insensitivity.\n"+
				"\n"+
				"Example\n"+
				"%s -exclude '(?m)^.*aaaa.*\\n' -exclude '(?m)^.*bbbb.*\\n' input.txt output.txt\n"+
				"will copy all the complete lines of text from input.txt to output.txt except for lines with aaaa or bbbb in them.\n\n",
			name, name, name)
		fmt.Fprintf(os.Stderr, "Flags\n")
		flag.PrintDefaults()
	}
}

// Set will be called by the flag package for flags of type regexes. It's part of the flag.Value interface.
// Set's argument is a string to be parsed to set the flag.
func (rs *regexes) Set(value string) error {
	// Value is a string with a single regular expression. Escape characters such as \n in the string will have been quoted by the
	// shell before the string is passed to the problem. Therefore the string needs to be unquoted before we can use it.
	valueq, err := strconv.Unquote("\"" + value + "\"")
	if err != nil {
		// Give the user more information if we have it about why Unquote failed.
		err = unquoteErrorExplanation(value, err)
		return err
	}
	*rs = append(*rs, valueq)
	return nil
}

// String is the method to format the regexes flag's value. It's part of the flag.Value interface.
func (rs *regexes) String() string {
	return fmt.Sprint(*rs)
}

// Get the input and output files from the command line.
func GetFilesFromCommandLine() (infile *os.File, outfile *os.File, err error) {
	var inFilename, outFilename string
	if flag.NArg() > 0 {
		inFilename = flag.Arg(0)
		if flag.NArg() > 1 {
			outFilename = flag.Arg(1)
		}
	}

	// Make sure inFilename is different from outFilename.
	if inFilename != "" && inFilename == outFilename {
		return nil, nil, errors.New("won't overwrite the infile")
	}

	// Start with stdin and stdout and replace them if filenames were on the command line.
	infile, outfile = os.Stdin, os.Stdout
	if inFilename != "" {
		if infile, err = os.Open(inFilename); err != nil {
			return nil, nil, fmt.Errorf("Couldn't open infile = %s", inFilename)
		}
	}
	if outFilename != "" {
		if outfile, err = os.Create(outFilename); err != nil {
			// Close infile if it was opened successfully.
			if infile != os.Stdin {
				infile.Close()
			}
			return nil, nil, fmt.Errorf("Couldn't open outfile = %s", outFilename)
		}
	}

	return infile, outfile, nil
}

// Parse the command line for flags and process the flags.
// If the command line has the help flag, help will be printed followed by os.Exit().
func ProcessFlags() {
	areFlagsProcessed = true
	flag.Parse()
	// If the command line has the help flag, just print help and exit.
	if helpFlag {
		flag.Usage()
		os.Exit(0)
	}
}

// GetExcludes returns the regular expressions strings specified by the exclude flag.
func GetExcludes() []string {
	if !areFlagsProcessed {
		ProcessFlags()
	}
	return excludeFlag
}

// GetIncludes returns the regular expressions strings specified by the include flag.
func GetIncludes() []string {
	if !areFlagsProcessed {
		ProcessFlags()
	}
	return includeFlag
}

// GetBufferSize returns the buffer size specified on the command line, or its default value. The buffer
// size is in number of bytes, so have to convert from the command line argument which is in KB.
func GetBufferSize() int64 {
	return sizeFlag * 1000
}

// Get the end-of-line character.
func GetEol() string {
	eol, err := strconv.Unquote("\"" + eolFlag + "\"")
	if err != nil {
		// Give the user more information if we have it about why Unquote failed.
		err = unquoteErrorExplanation(eolFlag, err)
		log.Fatal(err)
	}
	return eol
}

func IsVerbose() bool {
	return verboseFlag
}

// unquoteErrorExplanation examines the string for common errors that
// will cause strconv.Unquote to return an error. If it finds a problem
// it returns an err with more information.
func unquoteErrorExplanation(s string, err error) error {
	// Look for an unquoted \
	if match, _ := regexp.MatchString("[^\\]\\[^\\abfnrtvxuU0-7'\"]", s); match == true {
		return errors.New("Regular expression contains an unquoted backslash. Try \\\\ in the input string instead of \\.")
	}
	return fmt.Errorf("strconv.Unquote error:%s. Offending string is %s", err, s)
}
