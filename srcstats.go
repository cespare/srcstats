package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"unicode"
	"unicode/utf8"

	humanize "github.com/dustin/go-humanize"
)

const binaryDetectionBytes = 8000 // Same as git

var tabWidth = flag.Int("tabwidth", 4, "Width to assign tabs for determining line length")

type Stats struct {
	Lines              int // # of \n, plus the last line of a file if it's not newline-terminated (for shame)
	NonEmptyLines      int // Lines with at least one non-whitespace character
	NonWhitespaceChars int
	LengthSum          int // Sum of the length (position of last non-whitespace char) of non-empty lines
	Bytes              int
	Files              int // Number of files in these stats
}

func (s *Stats) Merge(other *Stats) {
	if other == nil {
		return
	}
	s.Lines += other.Lines
	s.NonEmptyLines += other.NonEmptyLines
	s.NonWhitespaceChars += other.NonWhitespaceChars
	s.LengthSum += other.LengthSum
	s.Bytes += other.Bytes
	s.Files += other.Files
}

func statsFromFile(f *os.File) (*Stats, error) {
	s := &Stats{Files: 1}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		s.Lines++
		nonWhitespace := 0
		rightmostNonWhitespace := 0
		totalWidth := 0
		for i, w := 0, 0; i < len(line); i += w {
			c, width := utf8.DecodeRuneInString(line[i:])
			w = width
			if c == '\t' {
				totalWidth += *tabWidth
			} else {
				totalWidth += width
			}
			if !unicode.IsSpace(c) {
				nonWhitespace++
				rightmostNonWhitespace = totalWidth
			}
		}
		if nonWhitespace > 0 {
			s.NonWhitespaceChars += nonWhitespace
			s.NonEmptyLines++
			s.LengthSum += rightmostNonWhitespace
		}
		s.Bytes += len(line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return s, nil
}

func statsFromFilename(filename string) *Stats {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: skipping file %s: %s\n", filename, err)
		return nil
	}
	defer f.Close()

	if isBinary(f) {
		fmt.Fprintf(os.Stderr, "Warning: skipping binary file %s\n", filename)
		return nil
	}

	stats, err := statsFromFile(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error scanning %s: %s\n", filename, err)
		return nil
	}
	return stats
}

func div(p, q int) float64 {
	return float64(p) / float64(q)
}

func (s *Stats) String() string {
	output := []struct {
		label string
		value interface{}
	}{
		{"files", s.Files},
		{"total size", humanize.IBytes(uint64(s.Bytes))},
		{"mean file size", humanize.IBytes(uint64(div(s.Bytes, s.Files)))},
		{"total lines", s.Lines},
		{"lines / file", div(s.Lines, s.Files)},
		{"non-empty lines", s.NonEmptyLines},
		{"non-empty lines / file", div(s.NonEmptyLines, s.Files)},
		{"chars / non-empty line", div(s.NonWhitespaceChars, s.NonEmptyLines)},
		{"mean non-empty line length", div(s.LengthSum, s.NonEmptyLines)},
	}
	buf := &bytes.Buffer{}
	for _, line := range output {
		fmt.Fprintf(buf, "%-30s", line.label)
		var format string
		switch line.value.(type) {
		case int:
			format = "%10d\n"
		case float64:
			format = "%10.1f\n"
		case string:
			format = "%10s\n"
		}
		fmt.Fprintf(buf, format, line.value)
	}
	return buf.String()
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: srcstats [OPTIONS] FILE1 FILE2 ... (or pass filenames from stdin)
where OPTIONS are:`)
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()

	ncpu := runtime.NumCPU()
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(ncpu)
	}
	nWorkers := 2 * ncpu // This setting worked well for me one time.

	files := make(chan string)
	go func() {
		if flag.NArg() == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				files <- scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				fmt.Println("Error reading stdin:", err)
			}
		} else {
			for _, name := range flag.Args() {
				files <- name
			}
		}
		close(files)
	}()

	subTotals := make(chan *Stats)
	for i := 0; i < nWorkers; i++ {
		go func() {
			subTotal := &Stats{}
			for filename := range files {
				subTotal.Merge(statsFromFilename(filename))
			}
			subTotals <- subTotal
		}()
	}

	totals := &Stats{}
	for i := 0; i < nWorkers; i++ {
		totals.Merge(<-subTotals)
	}

	if totals.Files == 0 {
		fmt.Fprintln(os.Stderr, "Error: no files to analyze")
		os.Exit(1)
	}
	fmt.Print(totals)
}

// isBinary guesses whether a file is binary by reading the first X bytes and seeing if there are any nulls.
// Assumes the file is at the beginning.
func isBinary(file *os.File) bool {
	defer file.Seek(0, 0)
	buf := make([]byte, binaryDetectionBytes)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return false
		}
		if n == 0 {
			break
		}
		for i := 0; i < n; i++ {
			if buf[i] == 0x00 {
				return true
			}
		}
		buf = buf[n:]
	}
	return false
}
