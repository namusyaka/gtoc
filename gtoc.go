package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

const (
	defaultH1Heading = "Table Of Content"
	defaultFilename  = "./README.md"
)

func removeDashes(s string) string {
	return strings.Replace(s, "`", "", -1)
}

var htmlEscaper = strings.NewReplacer(
	`&`, "&amp;",
	`<`, "&lt;",
	`>`, "&gt;",
	`"`, "&#34;", // "&#34;" is shorter than "&quot;".
)

func canonicalHeading(s string) string {
	return removeDashes(htmlEscaper.Replace(s))
}

var (
	group        = regexp.MustCompile(`\(([^\(]+?)\)`)
	specialChars = regexp.MustCompile(`[\[\]\(\)\':\.?><&"]`)
)

func canonicalAnchor(s string) string {
	// ([^(]+?) #=> $1
	s = group.ReplaceAllString(s, "$1")

	// Foo Bar baz #=> foo bar baz
	s = strings.ToLower(s)

	// trims spaces
	s = strings.TrimSpace(s)

	// removes []()':
	s = specialChars.ReplaceAllString(s, "")

	// removes dashes
	s = removeDashes(s)

	// replace middle whitespaces with -
	return strings.Replace(s, " ", "-", -1)
}

func isBlockQuote(s string) bool {
	return strings.HasPrefix(s, "```")
}

type toc struct {
	io.Reader
	buf *bytes.Buffer

	tocHeading   bool
	indentString string
}

func (t *toc) reset() {
	t.buf = bytes.NewBuffer([]byte{})
}

func (t *toc) hasPrefixHeader(s string) (level int, ok bool) {
	b := []byte(s)
	size := len(b)
	if size == 0 || b[0] != '#' {
		return 0, false
	}
	if t.tocHeading {
		level++
	}
	for i := 1; i < size; i++ {
		if b[i] == '#' {
			level++
		} else {
			break
		}
	}
	return level, true
}

func (t *toc) writeHeading(n int, h string) {
	sp := strings.Repeat(t.indentString, n*3)
	s := fmt.Sprintf("%s* [%s](#%s)\n", sp, canonicalHeading(h), canonicalAnchor(h))
	t.buf.WriteString(s)
}

func (t *toc) parse() error {
	t.reset()

	s := bufio.NewScanner(t)
	skip := false
	if t.tocHeading {
		t.writeHeading(0, "Table Of Content")
	}
	// TODO: support underline headings
	for s.Scan() {
		line := s.Text()
		if isBlockQuote(line) {
			skip = !skip
			continue
		}
		if skip {
			continue
		}
		// #...#####
		if n, ok := t.hasPrefixHeader(line); ok {
			p := strings.SplitN(line, " ", 2)
			t.writeHeading(n, p[1])
		}
	}
	if t.buf.Len() == 0 {
		return errors.New("toc: failed to detect heading line from given markdown")
	}
	return nil
}

func (t *toc) String() string {
	return t.buf.String()
}

var (
	tocHeading   bool
	indentString string
)

func main() {
	flag.BoolVar(&tocHeading, "d", false, "Whether to enable the default top level heading or not")
	flag.StringVar(&indentString, "s", "  ", "A string used as indent")
	flag.Parse()

	path := defaultFilename
	if args := flag.Args(); len(args) > 1 {
		path = args[0]
	}
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("toc: failed to open file, error: %v", err)
	}
	defer f.Close()

	t := &toc{
		Reader:       f,
		tocHeading:   tocHeading,
		indentString: indentString,
	}
	if err := t.parse(); err != nil {
		log.Fatalf("toc: failed to parse given markdown file, error: %v", err)
	}

	fmt.Print(t)
}
