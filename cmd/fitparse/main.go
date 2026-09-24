// Command fitparse prints a summary of FIT activity files, or their decoded
// content as JSON.
//
// Usage:
//
//	fitparse [flags] FILE...
//
// By default fitparse prints a short human-readable summary of each file:
// sport, local start time, duration, distance, pace and speed, heart rate,
// power, ascent, lap and record counts, the recording device and whether
// Stryd developer fields are present.
//
// Flags:
//
//	-units km|mi  distance units for the summary (default km)
//	-json         print the decoded file as JSON instead of a summary
//	-records      include per-second records in the JSON output (implies -json)
//	-pretty       indent the JSON output (implies -json)
//
// With -json, one JSON document is written for each file that decodes, in
// argument order. Files that fail produce no document; their errors go to
// standard error and the exit status is 1.
//
// Errors are written to standard error, prefixed with the file name. The
// exit status is 1 if any file fails to decode and 2 on invalid usage.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
)

const (
	metersPerMile = 1609.344
	feetPerMeter  = 3.28084
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// options holds the parsed command-line flags.
type options struct {
	json    bool
	records bool
	pretty  bool
	miles   bool
}

// run executes fitparse with args (without the program name) and returns
// the process exit status.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fitparse", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts options
	units := fs.String("units", "km", "summary distance `unit`: km or mi")
	fs.BoolVar(&opts.json, "json", false, "print the decoded file as JSON instead of a summary")
	fs.BoolVar(&opts.records, "records", false, "include records in the JSON output (implies -json)")
	fs.BoolVar(&opts.pretty, "pretty", false, "indent the JSON output (implies -json)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: fitparse [flags] FILE...")
		fmt.Fprintln(fs.Output(), "\nPrints a summary of each FIT file, or its decoded content as JSON.")
		fmt.Fprintln(fs.Output(), "\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	switch *units {
	case "km":
	case "mi":
		opts.miles = true
	default:
		fmt.Fprintf(stderr, "fitparse: invalid -units %q: want km or mi\n", *units)
		return 2
	}
	if opts.records || opts.pretty {
		opts.json = true
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}

	status := 0
	printed := false
	for _, path := range fs.Args() {
		f, err := fitparser.DecodeFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", path, err)
			status = 1
			continue
		}
		if opts.json {
			if err := writeJSON(stdout, f, opts); err != nil {
				fmt.Fprintf(stderr, "%s: %v\n", path, err)
				status = 1
			}
			continue
		}
		if printed {
			fmt.Fprintln(stdout)
		}
		writeSummary(stdout, path, f, opts)
		printed = true
	}
	return status
}

// writeJSON writes f as one JSON document. Records are left out unless
// opts.records is set.
func writeJSON(w io.Writer, f *fitparser.File, opts options) error {
	if !opts.records {
		f.Records = nil
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if opts.pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(f)
}

// writeSummary writes a human-readable summary of f.
func writeSummary(w io.Writer, path string, f *fitparser.File, opts options) {
	fmt.Fprintln(w, path)
	line := func(indent, label, value string) {
		fmt.Fprintf(w, "%s%-11s %s\n", indent, label, value)
	}

	fileType := "-"
	if f.FileID != nil && f.FileID.Type != "" {
		fileType = f.FileID.Type
	}
	line("  ", "Type", fileType)

	switch len(f.Sessions) {
	case 0:
		line("  ", "Sessions", "none")
	case 1:
		writeSession(line, "  ", &f.Sessions[0], f.Location(), f.HasUTCOffset, opts)
	default:
		for i := range f.Sessions {
			fmt.Fprintf(w, "  Session %d of %d\n", i+1, len(f.Sessions))
			writeSession(line, "    ", &f.Sessions[i], f.Location(), f.HasUTCOffset, opts)
		}
	}

	line("  ", "Laps", fmt.Sprint(len(f.Laps)))
	line("  ", "Records", fmt.Sprint(len(f.Records)))
	line("  ", "Device", device(f))
	line("  ", "Stryd", yesNo(f.HasStryd))
}

// writeSession writes the summary lines of one session.
func writeSession(line func(indent, label, value string), indent string, s *fitparser.Session, loc *time.Location, hasOffset bool, opts options) {
	sport := orDash(s.Sport)
	if s.SubSport != "" {
		sport += " (" + s.SubSport + ")"
	}
	line(indent, "Sport", sport)
	line(indent, "Start", startTime(s.StartTime, loc, hasOffset))

	duration := "-"
	if s.TotalTimerTime > 0 {
		duration = clock(s.TimerTime())
		if s.TotalElapsedTime > s.TotalTimerTime {
			duration += " (elapsed " + clock(s.ElapsedTime()) + ")"
		}
	} else if s.TotalElapsedTime > 0 {
		duration = clock(s.ElapsedTime()) + " (elapsed)"
	}
	line(indent, "Duration", duration)

	distUnit, distScale := "km", 1000.0
	if opts.miles {
		distUnit, distScale = "mi", metersPerMile
	}
	distance := "-"
	if s.TotalDistance > 0 {
		distance = fmt.Sprintf("%.2f %s", s.DistanceM()/distScale, distUnit)
	}
	line(indent, "Distance", distance)

	speed := s.AvgSpeedMps()
	if speed == 0 && s.TotalTimerTime > 0 {
		speed = s.DistanceM() / s.TimerTime().Seconds()
	}
	avgPace := "-"
	if speed > 0 {
		perHour := "km/h"
		if opts.miles {
			perHour = "mph"
		}
		avgPace = fmt.Sprintf("%s /%s (%.1f %s)", pace(distScale/speed), distUnit, speed*3600/distScale, perHour)
	}
	line(indent, "Avg pace", avgPace)

	line(indent, "Heart rate", avgMax(uint16(s.AvgHeartRate), uint16(s.MaxHeartRate), "bpm"))
	if s.AvgPower > 0 || s.MaxPower > 0 {
		line(indent, "Power", avgMax(s.AvgPower, s.MaxPower, "W"))
	}

	ascent := fmt.Sprintf("%d m", s.TotalAscent)
	if opts.miles {
		ascent = fmt.Sprintf("%.0f ft", float64(s.TotalAscent)*feetPerMeter)
	}
	line(indent, "Ascent", ascent)
}

// startTime formats t in the file's local time, or in UTC when the file
// has no local offset.
func startTime(t time.Time, loc *time.Location, hasOffset bool) string {
	if t.IsZero() {
		return "-"
	}
	if !hasOffset {
		return t.UTC().Format("2006-01-02 15:04:05") + " UTC"
	}
	return t.In(loc).Format("2006-01-02 15:04:05 -07:00")
}

// clock formats d as h:mm:ss, rounded to the second.
func clock(d time.Duration) string {
	s := int64(math.Round(d.Seconds()))
	return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
}

// pace formats seconds per distance unit as m:ss.
func pace(secs float64) string {
	s := int64(math.Round(secs))
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// avgMax formats an average and a maximum; absent (zero) values print as
// "-".
func avgMax(avg, peak uint16, unit string) string {
	if avg == 0 && peak == 0 {
		return "-"
	}
	num := func(v uint16) string {
		if v == 0 {
			return "-"
		}
		return fmt.Sprint(v)
	}
	return fmt.Sprintf("avg %s, max %s %s", num(avg), num(peak), unit)
}

// device describes the recording device: the file_id manufacturer and
// product, plus the creator device's software version when known.
func device(f *fitparser.File) string {
	var parts []string
	var product uint16
	if f.FileID != nil {
		parts = appendNonEmpty(parts, f.FileID.Manufacturer, f.FileID.ProductName)
		product = f.FileID.Product
	}
	var software float64
	for i := range f.DeviceInfos {
		d := &f.DeviceInfos[i]
		if d.DeviceIndex != 0 {
			continue
		}
		if len(parts) == 0 {
			parts = appendNonEmpty(parts, d.Manufacturer, d.ProductName)
		}
		if d.SoftwareVersion > 0 {
			software = d.SoftwareVersion
		}
		break
	}
	if len(parts) == 0 {
		return "-"
	}
	if len(parts) == 1 && product > 0 {
		parts = append(parts, fmt.Sprintf("(product %d)", product))
	}
	s := strings.Join(parts, " ")
	if software > 0 {
		s += fmt.Sprintf(" (software %.2f)", software)
	}
	return s
}

func appendNonEmpty(dst []string, values ...string) []string {
	for _, v := range values {
		if v != "" {
			dst = append(dst, v)
		}
	}
	return dst
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
