package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/proto"
)

var sampleStart = time.Date(2024, 5, 4, 7, 30, 0, 0, time.UTC)

// sampleRun is the synthetic 45-minute run with Stryd fields whose summary
// is shown in the README.
func sampleRun() *fitgen.Builder {
	return fitgen.New().
		AddStrydDescriptions(0).
		AddRun(fitgen.Run{
			Start:     sampleStart,
			Seconds:   2700,
			UTCOffset: 2 * time.Hour,
			RecordDev: func(i int) []proto.DeveloperField {
				return fitgen.StrydValues(0, fitgen.StrydSample{
					Power: 260, GroundTime: 240, VerticalOscillation: 8.5,
					FormPower: 70, LegSpringStiffness: 10.25, AirPower: 3,
				})
			},
		})
}

// writeFile writes data into a temporary directory and returns its path.
func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// runCLI runs fitparse with args and returns its exit status and output.
func runCLI(t *testing.T, args ...string) (status int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	status = run(args, &out, &errOut)
	return status, out.String(), errOut.String()
}

func TestSummary(t *testing.T) {
	path := writeFile(t, "run.fit", sampleRun().MustBytes())
	status, out, errOut := runCLI(t, path)
	if status != 0 || errOut != "" {
		t.Fatalf("status = %d, stderr = %q", status, errOut)
	}
	want := path + `
  Type        activity
  Sport       running (generic)
  Start       2024-05-04 09:30:00 +02:00
  Duration    0:45:00
  Distance    8.10 km
  Avg pace    5:33 /km (10.8 km/h)
  Heart rate  avg 150, max 159 bpm
  Power       avg 250, max 250 W
  Ascent      0 m
  Laps        1
  Records     2700
  Device      garmin fr965 (software 12.34)
  Stryd       yes
`
	if out != want {
		t.Errorf("summary mismatch\ngot:\n%s\nwant:\n%s", out, want)
	}
}

func TestSummaryMiles(t *testing.T) {
	path := writeFile(t, "run.fit", sampleRun().MustBytes())
	status, out, errOut := runCLI(t, "-units", "mi", path)
	if status != 0 || errOut != "" {
		t.Fatalf("status = %d, stderr = %q", status, errOut)
	}
	for _, want := range []string{
		"  Distance    5.03 mi\n",
		"  Avg pace    8:56 /mi (6.7 mph)\n",
		"  Ascent      0 ft\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestSummaryWithoutOffsetOrSpeed(t *testing.T) {
	// No activity message (so no local offset) and no avg speed: the start
	// time is shown in UTC and the speed is derived from distance and time.
	data := fitgen.New().
		AddFileID(func(m *mesgdef.FileId) {
			m.Type = typedef.FileActivity
			m.Manufacturer = typedef.ManufacturerDevelopment
			m.Product = 7
		}).
		AddSession(func(m *mesgdef.Session) {
			m.Timestamp = sampleStart.Add(time.Hour)
			m.StartTime = sampleStart
			m.Sport = typedef.SportCycling
			m.TotalElapsedTime = 3_700_000
			m.TotalTimerTime = 3_600_000
			m.TotalDistance = 3_000_000
			m.TotalAscent = 100
		}).
		MustBytes()
	path := writeFile(t, "ride.fit", data)
	status, out, errOut := runCLI(t, path)
	if status != 0 || errOut != "" {
		t.Fatalf("status = %d, stderr = %q", status, errOut)
	}
	for _, want := range []string{
		"  Sport       cycling\n",
		"  Start       2024-05-04 07:30:00 UTC\n",
		"  Duration    1:00:00 (elapsed 1:01:40)\n",
		"  Distance    30.00 km\n",
		"  Avg pace    2:00 /km (30.0 km/h)\n",
		"  Heart rate  -\n",
		"  Ascent      100 m\n",
		"  Device      development (product 7)\n",
		"  Stryd       no\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Power") {
		t.Errorf("power line printed without power data:\n%s", out)
	}
}

func TestSummaryNoSession(t *testing.T) {
	data := fitgen.New().
		AddFileID(func(m *mesgdef.FileId) {
			m.Type = typedef.FileWorkout
			m.Manufacturer = typedef.ManufacturerGarmin
			m.Product = uint16(typedef.GarminProductFr965)
		}).
		AddWorkout(func(m *mesgdef.Workout) {
			m.WktName = "Intervals"
			m.Sport = typedef.SportRunning
			m.NumValidSteps = 0
		}).
		MustBytes()
	path := writeFile(t, "workout.fit", data)
	status, out, errOut := runCLI(t, path)
	if status != 0 || errOut != "" {
		t.Fatalf("status = %d, stderr = %q", status, errOut)
	}
	want := path + `
  Type        workout
  Sessions    none
  Laps        0
  Records     0
  Device      garmin fr965
  Stryd       no
`
	if out != want {
		t.Errorf("summary mismatch\ngot:\n%s\nwant:\n%s", out, want)
	}
}

func TestSummaryMultipleSessions(t *testing.T) {
	b := fitgen.New().AddRun(fitgen.Run{Start: sampleStart, Seconds: 60})
	b.AddSession(func(m *mesgdef.Session) {
		m.StartTime = sampleStart.Add(2 * time.Minute)
		m.Sport = typedef.SportCycling
		m.SubSport = typedef.SubSportRoad
		m.TotalTimerTime = 600_000
		m.TotalDistance = 500_000
	})
	path := writeFile(t, "multi.fit", b.MustBytes())
	status, out, errOut := runCLI(t, path)
	if status != 0 || errOut != "" {
		t.Fatalf("status = %d, stderr = %q", status, errOut)
	}
	for _, want := range []string{
		"  Session 1 of 2\n    Sport       running (generic)\n",
		"  Session 2 of 2\n    Sport       cycling (road)\n",
		"    Distance    5.00 km\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestJSON(t *testing.T) {
	path := writeFile(t, "run.fit", sampleRun().MustBytes())
	status, out, errOut := runCLI(t, "-json", path)
	if status != 0 || errOut != "" {
		t.Fatalf("status = %d, stderr = %q", status, errOut)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("-json output is not a single line")
	}
	if strings.Contains(out, `"records"`) {
		t.Errorf("-json output contains records without -records")
	}
	var f fitparser.File
	if err := json.Unmarshal([]byte(out), &f); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(f.Sessions) != 1 || f.Sessions[0].Sport != "running" || f.Sessions[0].TotalDistance != 810000 {
		t.Errorf("sessions = %+v", f.Sessions)
	}
	if !f.HasStryd || !f.HasUTCOffset || f.UTCOffset != 2*time.Hour {
		t.Errorf("HasStryd = %v, HasUTCOffset = %v, UTCOffset = %v", f.HasStryd, f.HasUTCOffset, f.UTCOffset)
	}
	if len(f.Records) != 0 {
		t.Errorf("records = %d, want 0", len(f.Records))
	}
}

func TestJSONRecords(t *testing.T) {
	path := writeFile(t, "run.fit", sampleRun().MustBytes())
	status, out, errOut := runCLI(t, "-json", "-records", path)
	if status != 0 || errOut != "" {
		t.Fatalf("status = %d, stderr = %q", status, errOut)
	}
	var f fitparser.File
	if err := json.Unmarshal([]byte(out), &f); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(f.Records) != 2700 {
		t.Fatalf("records = %d, want 2700", len(f.Records))
	}
	r := f.Records[10]
	if !r.Timestamp.Equal(sampleStart.Add(10*time.Second)) || r.HeartRate != fitgen.RunBaseHR+10 {
		t.Errorf("record 10 = %+v", r)
	}
	if r.Stryd == nil || r.Stryd.Power != 260 {
		t.Errorf("record 10 Stryd = %+v", r.Stryd)
	}
}

func TestPrettyImpliesJSON(t *testing.T) {
	path := writeFile(t, "run.fit", sampleRun().MustBytes())
	status, out, errOut := runCLI(t, "-pretty", path)
	if status != 0 || errOut != "" {
		t.Fatalf("status = %d, stderr = %q", status, errOut)
	}
	if !strings.HasPrefix(out, "{\n  \"file_id\": {\n") {
		t.Errorf("-pretty output is not indented JSON:\n%.200s", out)
	}
	if !json.Valid([]byte(out)) {
		t.Errorf("-pretty output is not valid JSON")
	}
}

func TestDecodeErrors(t *testing.T) {
	bad := writeFile(t, "bad.fit", []byte("this is not a FIT file at all"))
	missing := filepath.Join(t.TempDir(), "missing.fit")
	for _, path := range []string{bad, missing} {
		for _, flags := range [][]string{nil, {"-json"}} {
			status, out, errOut := runCLI(t, append(flags, path)...)
			if status != 1 {
				t.Errorf("%v %s: status = %d, want 1", flags, path, status)
			}
			if out != "" {
				t.Errorf("%v %s: stdout = %q, want empty", flags, path, out)
			}
			if !strings.HasPrefix(errOut, path+": fitparser: ") || strings.Count(errOut, "\n") != 1 {
				t.Errorf("%v %s: stderr = %q", flags, path, errOut)
			}
		}
	}
}

func TestErrorDoesNotStopOtherFiles(t *testing.T) {
	good := writeFile(t, "run.fit", sampleRun().MustBytes())
	bad := writeFile(t, "bad.fit", []byte{0, 1, 2, 3})
	status, out, errOut := runCLI(t, bad, good)
	if status != 1 {
		t.Errorf("status = %d, want 1", status)
	}
	if !strings.HasPrefix(out, good+"\n  Type        activity\n") {
		t.Errorf("good file not summarized:\n%s", out)
	}
	if !strings.HasPrefix(errOut, bad+": ") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestMultipleFilesSeparatedByBlankLine(t *testing.T) {
	a := writeFile(t, "a.fit", sampleRun().MustBytes())
	b := writeFile(t, "b.fit", sampleRun().MustBytes())
	status, out, _ := runCLI(t, a, b)
	if status != 0 {
		t.Fatalf("status = %d", status)
	}
	if !strings.Contains(out, "Stryd       yes\n\n"+b+"\n") {
		t.Errorf("files not separated by a blank line:\n%s", out)
	}
}

func TestUsage(t *testing.T) {
	path := writeFile(t, "run.fit", sampleRun().MustBytes())
	tests := []struct {
		name   string
		args   []string
		status int
		stderr string
	}{
		{"no files", nil, 2, "Usage: fitparse"},
		{"bad units", []string{"-units", "furlong", path}, 2, `invalid -units "furlong"`},
		{"unknown flag", []string{"-bogus", path}, 2, "flag provided but not defined"},
		{"help", []string{"-h"}, 0, "Usage: fitparse"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, out, errOut := runCLI(t, tt.args...)
			if status != tt.status {
				t.Errorf("status = %d, want %d", status, tt.status)
			}
			if out != "" {
				t.Errorf("stdout = %q, want empty", out)
			}
			if !strings.Contains(errOut, tt.stderr) {
				t.Errorf("stderr = %q, want it to contain %q", errOut, tt.stderr)
			}
		})
	}
}
