package fitparser_test

import (
	"bytes"
	"math"
	"testing"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/proto"
)

func TestDecodeSmokeRunningActivity(t *testing.T) {
	start := time.Date(2024, 5, 4, 7, 30, 0, 0, time.UTC)
	data := fitgen.New().
		AddStrydDescriptions(0).
		AddRun(fitgen.Run{
			Start:     start,
			Seconds:   120,
			UTCOffset: 2 * time.Hour,
			RecordDev: func(i int) []proto.DeveloperField {
				return fitgen.StrydValues(0, fitgen.StrydSample{
					Power: 260, GroundTime: 240, VerticalOscillation: 8.5,
					FormPower: 70, LegSpringStiffness: 10.25, AirPower: 3,
				})
			},
		}).
		MustBytes()

	f, err := fitparser.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if f.FileID == nil || f.FileID.Type != "activity" || f.FileID.Manufacturer != "garmin" || f.FileID.ProductName != "fr965" {
		t.Errorf("FileID = %+v", f.FileID)
	}
	if len(f.Sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(f.Sessions))
	}
	s := f.Sessions[0]
	if s.Sport != "running" || s.SubSport != "generic" {
		t.Errorf("sport = %q/%q", s.Sport, s.SubSport)
	}
	if !s.StartTime.Equal(start) || s.StartTime.Location() != time.UTC {
		t.Errorf("start = %v", s.StartTime)
	}
	if s.ElapsedTime() != 120*time.Second {
		t.Errorf("elapsed = %v", s.ElapsedTime())
	}
	if got := s.DistanceM(); got != 360 {
		t.Errorf("distance = %v m, want 360", got)
	}
	if got := s.AvgSpeedMps(); got != 3 {
		t.Errorf("avg speed = %v, want 3", got)
	}

	if len(f.Records) != 120 {
		t.Fatalf("records = %d, want 120", len(f.Records))
	}
	r := f.Records[10]
	if r.HeartRate != 150 || r.SpeedMps() != 3 || r.DistanceM() != 30 {
		t.Errorf("record 10: hr=%d speed=%v dist=%v", r.HeartRate, r.SpeedMps(), r.DistanceM())
	}
	if alt, ok := r.AltitudeM(); !ok || alt != 100 {
		t.Errorf("altitude = %v, %v", alt, ok)
	}
	if lat, lon, ok := r.Position(); !ok || math.Abs(lat-10) > 1e-4 || math.Abs(lon-20) > 1e-6 {
		t.Errorf("position = %v, %v, %v", lat, lon, ok)
	}
	if c, ok := r.TemperatureC(); !ok || c != 21 {
		t.Errorf("temperature = %v, %v", c, ok)
	}
	if r.StanceTimeMs() != 250 || r.RespirationRateBrpm() != 30 {
		t.Errorf("stance=%v resp=%v", r.StanceTimeMs(), r.RespirationRateBrpm())
	}

	if !f.HasStryd || r.Stryd == nil {
		t.Fatalf("HasStryd=%v Stryd=%v", f.HasStryd, r.Stryd)
	}
	want := fitparser.Stryd{Power: 260, GroundTime: 240, VerticalOscillation: 8.5, FormPower: 70, LegSpringStiffness: 10.25, AirPower: 3}
	if *r.Stryd != want {
		t.Errorf("stryd = %+v, want %+v", *r.Stryd, want)
	}
	if len(r.DeveloperFields) != 6 || r.DeveloperFields[0].Name != "Power" || r.DeveloperFields[0].Value != uint16(260) {
		t.Errorf("developer fields = %+v", r.DeveloperFields)
	}

	if !f.HasUTCOffset || f.UTCOffset != 2*time.Hour {
		t.Errorf("utc offset = %v, %v", f.UTCOffset, f.HasUTCOffset)
	}
	if got := r.Timestamp.In(f.Location()).Hour(); got != 9 {
		t.Errorf("local hour = %d, want 9", got)
	}
	if len(f.Events) != 2 || f.Events[1].EventType != "stop_all" {
		t.Errorf("events = %+v", f.Events)
	}
	if len(f.Laps) != 1 || f.Laps[0].LapTrigger != "session_end" {
		t.Errorf("laps = %+v", f.Laps)
	}
	if len(f.DeviceInfos) != 1 || f.DeviceInfos[0].DeviceIndex != 0 || f.DeviceInfos[0].SoftwareVersion != 12.34 {
		t.Errorf("device infos = %+v", f.DeviceInfos)
	}
}
