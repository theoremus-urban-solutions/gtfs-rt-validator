package validator

import (
	"encoding/csv"
	"errors"
	"io"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func readAllStd(in string) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(in))
	r.FieldsPerRecord = -1
	return r.ReadAll()
}

func readAllScan(in string) ([][]string, error) {
	s := newCSVScanner(strings.NewReader(in))
	var out [][]string
	for {
		rec, err := s.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		row := make([]string, len(rec))
		for i, f := range rec {
			row[i] = string(f)
		}
		out = append(out, row)
	}
}

// TestCSVScannerMatchesEncodingCSV checks records and quote errors against encoding/csv.
func TestCSVScannerMatchesEncodingCSV(t *testing.T) {
	inputs := []string{
		"", "a", "a,b\n", "a,b\r\nc,d\r\n", "a,b\nc,d", "a,,b,\n", ",\n", ",,\n",
		"\n\na\n\n\nb\n", "\r\n\r\na\r\n", "x\r\n\r\n",
		`"a,b",c` + "\n", `"a""b",c` + "\n", `"",""` + "\n", `"a` + "\n" + `b",c` + "\n", `"a` + "\r\n" + `b",c` + "\n",
		`"a"` + "\n", `a,"b"` + "\n", `a,"b",` + "\n", `"a",` + "\n",
		`a"b,c` + "\n", `"a"b,c` + "\n", `"a` + "\n", `"a" ,b` + "\n", `a,"b` + "\n" + "\n" + "c\"\n",
		" a , b \n", "a\rb,c\n", "\ufeffstop_id,stop_name\nS1,x\n",
		strings.Repeat("x", 70000) + ",y\n" + strings.Repeat("z,", 100) + "\n",
		`"` + strings.Repeat("q", 70000) + "\n" + strings.Repeat("w", 70000) + `"` + ",e\n",
	}
	for _, in := range inputs {
		want, wantErr := readAllStd(in)
		got, gotErr := readAllScan(in)
		short := in
		if len(short) > 40 {
			short = short[:40] + "..."
		}
		if (wantErr == nil) != (gotErr == nil) {
			t.Errorf("%q: encoding/csv error %v, scanner error %v", short, wantErr, gotErr)
			continue
		}
		if wantErr != nil {
			var we, ge *csv.ParseError
			if errors.As(wantErr, &we) && errors.As(gotErr, &ge) && !errors.Is(we.Err, ge.Err) {
				t.Errorf("%q: encoding/csv error %v, scanner error %v", short, wantErr, gotErr)
			}
			continue
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("%q: encoding/csv %q, scanner %q", short, want, got)
		}
	}
}

// TestStartFormatsMatchRegexps checks the hand-written E020 and E021 tests against the regexps they replace.
func TestStartFormatsMatchRegexps(t *testing.T) {
	reTime := regexp.MustCompile(`^([0-1]?\d|2\d|29):[0-5]\d:[0-5]\d$`)
	reDate := regexp.MustCompile(`^\d{8}$`)
	alphabet := "01235689:x"
	var walk func(prefix []byte, left int)
	n := 0
	walk = func(prefix []byte, left int) {
		s := string(prefix)
		if validStartTime(s) != reTime.MatchString(s) {
			t.Fatalf("start_time %q: got %v, regexp %v", s, validStartTime(s), reTime.MatchString(s))
		}
		if validStartDate(s) != reDate.MatchString(s) {
			t.Fatalf("start_date %q: got %v, regexp %v", s, validStartDate(s), reDate.MatchString(s))
		}
		n++
		if left == 0 {
			return
		}
		for i := 0; i < len(alphabet); i++ {
			walk(append(prefix, alphabet[i]), left-1)
		}
	}
	walk(nil, 7)
	// Length 8 with the digits on each boundary: hour tens 2 and 3, minute tens 5 and 6.
	if !raceEnabled {
		alphabet = "02356:x"
		walk(nil, 8)
	}
	for _, s := range []string{"12:00:00", "29:59:59", "30:00:00", "02:00:00", "2:00:00", "1:00:00", "20260101", "2026010", "202601011", "12:60:00", "12:00:60"} {
		if validStartTime(s) != reTime.MatchString(s) || validStartDate(s) != reDate.MatchString(s) {
			t.Errorf("%q differs from the regexp", s)
		}
	}
	t.Logf("%d strings agree", n)
}
