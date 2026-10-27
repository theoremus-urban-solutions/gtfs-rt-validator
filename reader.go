package validator

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
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
		var files []*os.File
		cleanup = func() error {
			var first error
			for _, f := range files {
				if err := f.Close(); err != nil && first == nil {
					first = err
				}
			}
			return first
		}
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
				files = append(files, f)
				r := csv.NewReader(f)
				r.FieldsPerRecord = -1
				out[base] = r
			}
			return nil
		})
		if err != nil {
			_ = cleanup()
			return nil, func() error { return nil }, err
		}
		return out, cleanup, nil
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

// ZipBytesReaders yields csv readers for known GTFS files from raw zip bytes.
// This allows reading GTFS data from memory without touching the filesystem.
//
// Parameters:
//   - zipData: Complete GTFS zip file as byte slice
//
// Returns:
//   - map[string]*csv.Reader: Map of filename to CSV reader for each GTFS file
//   - func() error: Cleanup function to close all opened readers
//   - error: Any error during zip extraction
func ZipBytesReaders(zipData []byte) (map[string]*csv.Reader, func() error, error) {
	out := make(map[string]*csv.Reader)
	var closers []io.Closer

	cleanup := func() error {
		var firstErr error
		for _, c := range closers {
			if err := c.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}

	r := bytes.NewReader(zipData)
	zipReader, err := zip.NewReader(r, int64(len(zipData)))
	if err != nil {
		return nil, func() error { return nil }, fmt.Errorf("failed to open GTFS zip from bytes: %w", err)
	}

	for _, f := range zipReader.File {
		base := strings.ToLower(filepath.Base(f.Name))
		if !isKnownGTFS(base) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			_ = cleanup()
			return nil, func() error { return nil }, fmt.Errorf("failed to open %s: %w", f.Name, err)
		}

		closers = append(closers, rc)
		csvReader := csv.NewReader(rc)
		csvReader.FieldsPerRecord = -1
		out[base] = csvReader
	}

	return out, cleanup, nil
}
