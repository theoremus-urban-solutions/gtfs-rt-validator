package validator

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
)

// csvScanner reads records as encoding/csv does with FieldsPerRecord -1 and no LazyQuotes, but reuses one buffer for every record.
type csvScanner struct {
	r      *bufio.Reader
	long   []byte
	buf    []byte
	ends   []int
	fields [][]byte
	line   int
}

func newCSVScanner(r io.Reader) *csvScanner {
	return &csvScanner{r: bufio.NewReaderSize(r, 64<<10)}
}

// readLine returns the next line without its line ending; it is valid until the next call.
func (s *csvScanner) readLine() ([]byte, error) {
	line, err := s.r.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		s.long = append(s.long[:0], line...)
		for err == bufio.ErrBufferFull {
			line, err = s.r.ReadSlice('\n')
			s.long = append(s.long, line...)
		}
		line = s.long
	}
	if len(line) > 0 && err == io.EOF {
		err = nil
	}
	s.line++
	if n := len(line); n > 0 && line[n-1] == '\n' {
		line = line[:n-1]
		if n := len(line); n > 0 && line[n-1] == '\r' {
			line = line[:n-1]
		}
	}
	return line, err
}

// Read returns the next record, skipping empty lines; the fields are valid until the next call.
func (s *csvScanner) Read() ([][]byte, error) {
	var line []byte
	for {
		var err error
		if line, err = s.readLine(); err != nil {
			return nil, err
		}
		if len(line) > 0 {
			break
		}
	}
	start := s.line
	s.buf, s.ends = s.buf[:0], s.ends[:0]
	pos := 0
	for {
		if pos < len(line) && line[pos] == '"' {
			pos++
			for {
				i := bytes.IndexByte(line[pos:], '"')
				if i < 0 {
					s.buf = append(s.buf, line[pos:]...)
					s.buf = append(s.buf, '\n')
					next, err := s.readLine()
					if err != nil {
						return nil, &csv.ParseError{StartLine: start, Line: s.line, Err: csv.ErrQuote}
					}
					line, pos = next, 0
					continue
				}
				s.buf = append(s.buf, line[pos:pos+i]...)
				pos += i + 1
				if pos < len(line) && line[pos] == '"' {
					s.buf = append(s.buf, '"')
					pos++
					continue
				}
				break
			}
			s.ends = append(s.ends, len(s.buf))
			if pos == len(line) {
				break
			}
			if line[pos] != ',' {
				return nil, &csv.ParseError{StartLine: start, Line: s.line, Err: csv.ErrQuote}
			}
			pos++
			if pos == len(line) {
				s.ends = append(s.ends, len(s.buf))
				break
			}
			continue
		}
		i := bytes.IndexByte(line[pos:], ',')
		f := line[pos:]
		if i >= 0 {
			f = line[pos : pos+i]
		}
		if bytes.IndexByte(f, '"') >= 0 {
			return nil, &csv.ParseError{StartLine: start, Line: s.line, Err: csv.ErrBareQuote}
		}
		s.buf = append(s.buf, f...)
		s.ends = append(s.ends, len(s.buf))
		if i < 0 {
			break
		}
		pos += i + 1
		if pos == len(line) {
			s.ends = append(s.ends, len(s.buf))
			break
		}
	}
	s.fields = s.fields[:0]
	prev := 0
	for _, e := range s.ends {
		s.fields = append(s.fields, s.buf[prev:e:e])
		prev = e
	}
	return s.fields, nil
}
