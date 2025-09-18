package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	rt "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"

	validator "your.org/gtfsrt/validator"
)

var (
	version = "dev"
)

func usage() {
	fmt.Fprintf(os.Stderr, "gtfsrt-validator %s\n", version)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  gtfsrt-validator validate -gtfs <file.zip> -rt <file.pb> [-rt2 <file.pb>] [-format console|json|javajson]")
	fmt.Fprintln(os.Stderr, "  gtfsrt-validator validate -gtfs <file.zip> [-rt-tu <file.pb>] [-rt-vp <file.pb>] [-rt-alerts <file.pb>] [-format console|json|javajson]")
	fmt.Fprintln(os.Stderr, "  gtfsrt-validator validate -gtfs <file.zip> -rt-path <dir> [-sort name|date] [-format console|json|summary|javajson]")
}

func main() {
	if len(os.Args) == 1 {
		usage()
		return
	}
	switch os.Args[1] {
	case "--help", "-h":
		usage()
		return
	case "--version", "version":
		fmt.Println(version)
		return
	case "validate":
		runValidate()
	default:
		usage()
	}
}

func runValidate() {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	gtfsPath := fs.String("gtfs", "", "path to GTFS zip")
	rtPath := fs.String("rt", "", "path to GTFS-rt .pb file")
	rtPath2 := fs.String("rt2", "", "optional second GTFS-rt .pb file for combined checks")
	rtTU := fs.String("rt-tu", "", "TripUpdates .pb for combined checks")
	rtVP := fs.String("rt-vp", "", "VehiclePositions .pb for combined checks")
	rtAL := fs.String("rt-alerts", "", "Alerts .pb for combined checks")
	rtDir := fs.String("rt-path", "", "path to directory of .pb files")
	sortBy := fs.String("sort", "date", "sort order for -rt-path: name|date")
	format := fs.String("format", "console", "output format: console|json|summary|javajson")
	_ = fs.Parse(os.Args[2:])
	if *gtfsPath == "" || (*rtPath == "" && *rtDir == "" && *rtTU == "" && *rtVP == "" && *rtAL == "") {
		fmt.Fprintln(os.Stderr, "-gtfs and (-rt or -rt-tu/-rt-vp/-rt-alerts or -rt-path) are required")
		os.Exit(2)
	}
	v, err := validator.NewValidator(*gtfsPath, validator.Options{EnableShapes: true})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.Background()
	// Multi-source single iteration
	if *rtTU != "" || *rtVP != "" || *rtAL != "" || *rtPath != "" {
		var base *rt.FeedMessage
		var sources []*rt.FeedMessage
		if *rtTU != "" {
			if m, err := readPB(*rtTU); err == nil {
				sources = append(sources, m)
			}
		}
		if *rtVP != "" {
			if m, err := readPB(*rtVP); err == nil {
				sources = append(sources, m)
			}
		}
		if *rtAL != "" {
			if m, err := readPB(*rtAL); err == nil {
				sources = append(sources, m)
			}
		}
		if *rtPath != "" {
			m, err := readPB(*rtPath)
			if err == nil {
				sources = append(sources, m)
			}
		}
		if len(sources) > 0 {
			base = sources[0]
		}
		var combined *rt.FeedMessage
		if len(sources) > 1 {
			combined = validator.BuildCombined(sources...)
		}
		_, res := v.ValidateIteration(ctx, base, combined)
		switch *format {
		case "json":
			b, err := validator.MarshalJSON(res)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			os.Stdout.Write(b)
		case "javajson":
			b, _ := json.MarshalIndent(validator.ToCompat(res), "", "  ")
			os.Stdout.Write(b)
		default:
			fmt.Println(validator.RenderConsole(res))
		}
		return
	}
	if *rtPath != "" {
		msg, err := readPB(*rtPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var combined *rt.FeedMessage
		if *rtPath2 != "" {
			msg2, err := readPB(*rtPath2)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			combined = validator.BuildCombined(msg, msg2)
		}
		_, res := v.ValidateIteration(ctx, msg, combined)
		switch *format {
		case "json":
			b, err := validator.MarshalJSON(res)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			os.Stdout.Write(b)
		case "javajson":
			b, _ := json.MarshalIndent(validator.ToCompat(res), "", "  ")
			os.Stdout.Write(b)
		default:
			fmt.Println(validator.RenderConsole(res))
		}
		return
	}
	// batch directory logic unchanged below...
	entries, err := os.ReadDir(*rtDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	type item struct {
		name  string
		path  string
		mtime time.Time
	}
	var items []item
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(de.Name()), ".pb") {
			continue
		}
		p := filepath.Join(*rtDir, de.Name())
		st, _ := os.Stat(p)
		items = append(items, item{name: de.Name(), path: p, mtime: st.ModTime()})
	}
	switch *sortBy {
	case "name":
		sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	default:
		sort.Slice(items, func(i, j int) bool { return items[i].mtime.Before(items[j].mtime) })
	}
	var prevHash string
	for _, it := range items {
		msg, err := readPB(it.path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		h, _ := fileMD5(it.path)
		if h == prevHash {
			continue
		}
		prevHash = h
		_, res := v.ValidateIteration(ctx, msg, nil)
		if *format == "json" {
			out := it.path + ".results.json"
			b, err := validator.MarshalJSON(res)
			if err == nil {
				_ = os.WriteFile(out, b, 0644)
			}
		} else if *format == "javajson" {
			b, _ := json.MarshalIndent(validator.ToCompat(res), "", "  ")
			out := it.path + ".results.json"
			_ = os.WriteFile(out, b, 0644)
		} else if *format == "summary" {
			b, _ := json.MarshalIndent(struct{ Items interface{} }{Items: validator.Summarize(res)}, "", "  ")
			os.Stdout.Write(b)
			os.Stdout.Write([]byte("\n"))
		} else {
			fmt.Println(it.name)
			fmt.Println(validator.RenderConsole(res))
		}
	}
}

func readPB(p string) (*rt.FeedMessage, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	msg := &rt.FeedMessage{}
	if err := validator.UnmarshalProtoTextOrBinary(data, msg); err != nil {
		return nil, fmt.Errorf("failed to parse .pb: %w", err)
	}
	return msg, nil
}

func fileMD5(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
