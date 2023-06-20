package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var globalOffset = flag.Int64("offset", 0, "global offset of subtitle, in case that srt file does not exactly match video.")

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "%s [flags] <file> [mm:ss]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
	}
}

type Entry struct {
	Idx   uint64
	Start int64
	End   int64
	Text  string
}

// srt subtitle use timestamp format 'hh:mm:ss,milli'
func parseTimestamp(s string) (int64, error) {
	var millsecs int64
	var hours int64
	var minutes int64
	var seconds int64
	var err error

	parts := strings.Split(s, ",")
	if len(parts) == 2 {
		millsecs, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, err
		}
	}

	parts = strings.Split(parts[0], ":")
	if len(parts) != 3 {
		return 0, errors.New("invalid timestamp: " + s)
	} else {
		hours, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, err
		}
		minutes, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, err
		}
		seconds, err = strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return 0, err
		}
	}

	return hours*60*60*1000 + minutes*60*1000 + seconds*1000 + millsecs, nil
}

var validIdx = regexp.MustCompile(`^[0-9]+$`)

func readEntry(s *bufio.Scanner) (*Entry, error) {
	var idx uint64
	var err error
	// Idx
	if s.Scan() {
		text := s.Text()
		if !validIdx.MatchString(text) {
			return nil, errors.New("invalid idx: " + text)
		}

		idx, err = strconv.ParseUint(text, 10, 0)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	// timestamp lines are in format 'start --> end'
	var start int64
	var end int64
	if s.Scan() {
		texts := strings.Split(s.Text(), " --> ")
		if len(texts) != 2 {
			return nil, errors.New("invalid timestamp")
		}

		start, err = parseTimestamp(texts[0])
		if err != nil {
			return nil, err
		}
		end, err = parseTimestamp(texts[1])
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("invalid input: no timestamp after idx")
	}

	// text
	lines := make([]string, 0)
	for s.Scan() {
		text := s.Text()
		if text == "" {
			break
		} else {
			lines = append(lines, text)
		}
	}

	// TODO: handle scan error
	if len(lines) == 0 {
		return nil, errors.New("invalid input: no text")
	} else {
		return &Entry{
			Idx:   idx,
			Start: start,
			End:   end,
			Text:  strings.Join(lines, "\n"),
		}, nil
	}
}

const OFFSET_HEADER string = "OFFSET:"

func tryParseGlobalOffset(s *bufio.Scanner) (*int64, error) {
	if s.Scan() {
		text := s.Text()
		if strings.HasPrefix(text, OFFSET_HEADER) {
			offset, err := strconv.ParseInt(strings.TrimPrefix(text, OFFSET_HEADER), 10, 64)
			if err != nil {
				return nil, err
			}

			return &offset, nil
		}
	}

	return nil, nil
}

func parse(r io.ReadSeeker) ([]*Entry, error) {
	entries := make([]*Entry, 0)
	scanner := bufio.NewScanner(r)

	offset, err := tryParseGlobalOffset(scanner)
	if err != nil {
		return entries, nil
	}
	if offset != nil {
		*globalOffset = *offset
	} else {
		// reset reader and scanner if the first line is not global offset
		r.Seek(0, io.SeekStart)
		scanner = bufio.NewScanner(r)
	}

	for {
		entry, err := readEntry(scanner)
		if err != nil {
			return entries, err
		}

		if entry == nil {
			break
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

// mpc outputs current time of song in 'hh:mm' format, this function
// parses it into milliseconds
func parseQuery(s string) (int64, error) {
	parts := strings.Split(s, ":")
	var minutes int64
	var seconds int64
	var err error
	minutes, err = strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return 0, err
	}
	seconds, err = strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return 0, err
	}

	return minutes*60*1000 + seconds*1000, nil
}

var readQueryEOF = errors.New("EOF")

func readQuery() (int64, error) {
	s := bufio.NewScanner(os.Stdin)

	if !s.Scan() {
		return 0, errors.New("can not read query")
	}

	text := s.Text()
	if s.Text() == "" {
		return 0, readQueryEOF
	}

	return parseQuery(text)
}

func queryAndPrint(entries []*Entry, query int64, offset int64) {
	for _, e := range entries {
		// +/- 1 second to make the search more tolerant
		if e.End+offset+1000 >= query && e.Start+offset-1000 <= query {
			fmt.Println(e.Text)
		}
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: srt <file> [mm:ss]")
		os.Exit(1)
	}

	f, err := os.Open(args[0])
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	entries, err := parse(f)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("No entries found, now existing")
		os.Exit(0)
	}

	stat, _ := os.Stdin.Stat()
	if len(args) >= 2 {
		query, err := parseQuery(args[1])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		queryAndPrint(entries, query, *globalOffset)
	} else if stat.Mode()&os.ModeCharDevice == 0 {
		query, err := readQuery()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		queryAndPrint(entries, query, *globalOffset)
	} else {
		// interactive mode
		for {
			fmt.Print("Input timestamp: ")
			query, err := readQuery()
			if err != nil {
				if errors.Is(err, readQueryEOF) {
					os.Exit(0)
				} else {
					fmt.Println(err)
					os.Exit(1)
				}
			}

			queryAndPrint(entries, query, *globalOffset)
		}
	}
}
