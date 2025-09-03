package parser

import (
	"archive/zip"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileReader struct {
	Name string
	io.Reader
}

// ZipOrDirReaders yields csv readers for known GTFS files from a zip or directory path.
func ZipOrDirReaders(path string) (map[string]*csv.Reader, func() error, error) {
	cleanup := func() error { return nil }
	out := make(map[string]*csv.Reader)
	info, err := os.Stat(path)
	if err != nil {
		return nil, cleanup, err
	}
	if info.IsDir() {
		err = filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return nil
			}
			base := strings.ToLower(filepath.Base(p))
			if isKnownGTFS(base) {
				f, err := os.Open(p)
				if err != nil {
					return err
				}
				r := csv.NewReader(f)
				r.FieldsPerRecord = -1
				out[base] = r
			}
			return nil
		})
		return out, cleanup, err
	}
	// assume zip
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, cleanup, err
	}
	cleanup = z.Close
	for _, f := range z.File {
		base := strings.ToLower(filepath.Base(f.Name))
		if isKnownGTFS(base) {
			rc, err := f.Open()
			if err != nil {
				_ = z.Close()
				return nil, func() error { return nil }, err
			}
			r := csv.NewReader(rc)
			r.FieldsPerRecord = -1
			out[base] = r
		}
	}
	return out, cleanup, nil
}

func isKnownGTFS(name string) bool {
	switch name {
	case "agency.txt", "routes.txt", "trips.txt", "stops.txt", "stop_times.txt", "shapes.txt", "frequencies.txt":
		return true
	default:
		return false
	}
}
