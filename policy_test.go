package fitparser_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/profile/untyped/fieldnum"
	"github.com/muktihari/fit/proto"
)

func TestRecordRespiration(t *testing.T) {
	cases := []struct {
		name     string
		enhanced uint16 // 0 = not written
		legacy   uint8  // 0 = not written
		want     uint16
	}{
		{"enhanced", 3125, 0, 3125},
		{"enhanced at cap", 10000, 0, 10000},
		{"enhanced above cap", 10001, 0, 0},
		{"enhanced near sentinel", 65534, 0, 0},
		{"legacy x100", 0, 20, 2000},
		{"legacy at cap", 0, 100, 10000},
		{"legacy above cap", 0, 101, 0},
		{"legacy 254", 0, 254, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := decode(t, fitgen.New().AddRecord(func(m *mesgdef.Record) {
				m.Timestamp = t0
				if c.enhanced != 0 {
					m.EnhancedRespirationRate = c.enhanced
				}
				if c.legacy != 0 {
					m.RespirationRate = c.legacy
				}
			}))
			r := one(t, f.Records)
			if r.EnhancedRespirationRate != c.want {
				t.Errorf("EnhancedRespirationRate = %d, want %d", r.EnhancedRespirationRate, c.want)
			}
			if got, want := r.RespirationRateBrpm(), float64(c.want)/100; got != want {
				t.Errorf("RespirationRateBrpm = %v, want %v", got, want)
			}
		})
	}
}

func TestRecordRespirationExplicitSentinel(t *testing.T) {
	rec := mesgdef.NewRecord(nil)
	rec.Timestamp = t0
	m := rec.ToMesg(nil)
	fitgen.SetField(&m, fieldnum.RecordEnhancedRespirationRate, proto.Uint16(basetype.Uint16Invalid))
	fitgen.SetField(&m, fieldnum.RecordRespirationRate, proto.Uint8(basetype.Uint8Invalid))
	if r := one(t, decode(t, fitgen.New().Add(m)).Records); r.EnhancedRespirationRate != 0 {
		t.Errorf("EnhancedRespirationRate = %d, want 0", r.EnhancedRespirationRate)
	}
}

func TestSessionRespirationCap(t *testing.T) {
	f := decode(t, fitgen.New().AddSession(func(m *mesgdef.Session) {
		m.EnhancedAvgRespirationRate = 10000
		m.EnhancedMaxRespirationRate = 10001
		m.EnhancedMinRespirationRate = 65534
	}))
	s := one(t, f.Sessions)
	if s.EnhancedAvgRespirationRate != 10000 || s.EnhancedMaxRespirationRate != 0 || s.EnhancedMinRespirationRate != 0 {
		t.Errorf("respiration avg/max/min = %d/%d/%d, want 10000/0/0",
			s.EnhancedAvgRespirationRate, s.EnhancedMaxRespirationRate, s.EnhancedMinRespirationRate)
	}
}

func TestRecordPowerNoPlausibilityCap(t *testing.T) {
	cases := []struct {
		raw, want uint16
	}{
		{2000, 2000}, {2500, 2500}, {65534, 65534}, {basetype.Uint16Invalid, 0},
	}
	for _, c := range cases {
		rec := mesgdef.NewRecord(nil)
		rec.Timestamp = t0
		m := rec.ToMesg(nil)
		fitgen.SetField(&m, fieldnum.RecordPower, proto.Uint16(c.raw))
		if got := one(t, decode(t, fitgen.New().Add(m)).Records).Power; got != c.want {
			t.Errorf("Power(%d) = %d, want %d", c.raw, got, c.want)
		}
	}
}

func TestNoOtherPlausibilityCaps(t *testing.T) {
	f := decode(t, fitgen.New().
		AddRecord(func(m *mesgdef.Record) {
			m.Timestamp = t0
			m.HeartRate = 254
			m.Cadence = 254
			m.EnhancedSpeed = 100_000
		}).
		AddSession(func(m *mesgdef.Session) { m.AvgSpo2 = 150; m.MaxHeartRate = 254; m.MaxPower = 3000 }))
	r := one(t, f.Records)
	if r.HeartRate != 254 || r.Cadence != 254 || r.EnhancedSpeed != 100_000 {
		t.Errorf("record = %+v", r)
	}
	s := one(t, f.Sessions)
	if s.AvgSpo2 != 150 || s.MaxHeartRate != 254 || s.MaxPower != 3000 {
		t.Errorf("session AvgSpo2=%d MaxHeartRate=%d MaxPower=%d", s.AvgSpo2, s.MaxHeartRate, s.MaxPower)
	}
}

func TestLegacySpeedExpandedToEnhanced(t *testing.T) {
	// The upstream decoder expands the legacy uint16 speed into
	// enhanced_speed; SpeedMps reads it either way.
	f := decode(t, fitgen.New().AddRecord(func(m *mesgdef.Record) { m.Timestamp = t0; m.Speed = 2750 }))
	r := one(t, f.Records)
	if r.SpeedMps() != 2.75 {
		t.Errorf("SpeedMps = %v, want 2.75 (EnhancedSpeed=%d Speed=%d)", r.SpeedMps(), r.EnhancedSpeed, r.Speed)
	}
}

func TestRRIntervals(t *testing.T) {
	f := decode(t, fitgen.New().
		AddHRV(800, 0, basetype.Uint16Invalid, 650).
		AddRecord(func(m *mesgdef.Record) { m.Timestamp = t0 }).
		AddHRV(700).
		AddHRV(basetype.Uint16Invalid).
		AddHRV(1, 65534))
	want := []uint16{800, 650, 700, 1, 65534}
	if !reflect.DeepEqual(f.RRIntervals, want) {
		t.Errorf("RRIntervals = %v, want %v", f.RRIntervals, want)
	}
}

func TestRRIntervalsNoneValid(t *testing.T) {
	f := decode(t, fitgen.New().AddHRV(0, basetype.Uint16Invalid))
	if f.RRIntervals != nil {
		t.Errorf("RRIntervals = %v, want nil", f.RRIntervals)
	}
}

func TestChainedFilesMerge(t *testing.T) {
	start2 := t0.Add(3 * time.Hour)
	f := decode(t, fitgen.New().
		AddRun(fitgen.Run{Start: t0, Seconds: 10, UTCOffset: time.Hour}).
		AddZonesTarget(func(m *mesgdef.ZonesTarget) { m.FunctionalThresholdPower = 250 }).
		AddHRV(800).
		NewSequence().
		AddFileID(func(m *mesgdef.FileId) {
			m.Type = typedef.FileActivity
			m.Manufacturer = typedef.ManufacturerWahooFitness
			m.TimeCreated = start2
		}).
		AddRecord(func(m *mesgdef.Record) { m.Timestamp = start2; m.HeartRate = 99 }).
		AddSession(func(m *mesgdef.Session) { m.StartTime = start2; m.Sport = typedef.SportCycling }).
		AddZonesTarget(func(m *mesgdef.ZonesTarget) { m.FunctionalThresholdPower = 300 }).
		AddHRV(750).
		AddActivity(activity(start2, start2.Add(-4*time.Hour))))

	if f.FileID == nil || f.FileID.Manufacturer != "garmin" {
		t.Errorf("FileID = %+v, want the first sequence's (garmin)", f.FileID)
	}
	if len(f.Sessions) != 2 || f.Sessions[0].Sport != "running" || f.Sessions[1].Sport != "cycling" {
		t.Errorf("sessions = %+v", f.Sessions)
	}
	if len(f.Records) != 11 || f.Records[10].HeartRate != 99 {
		t.Errorf("records = %d, last HR = %d", len(f.Records), f.Records[len(f.Records)-1].HeartRate)
	}
	if len(f.Events) != 2 || len(f.Laps) != 1 || len(f.DeviceInfos) != 1 {
		t.Errorf("events=%d laps=%d devices=%d", len(f.Events), len(f.Laps), len(f.DeviceInfos))
	}
	if f.ZonesTarget == nil || f.ZonesTarget.FunctionalThresholdPower != 300 {
		t.Errorf("ZonesTarget = %+v, want the last one (FTP 300)", f.ZonesTarget)
	}
	if !reflect.DeepEqual(f.RRIntervals, []uint16{800, 750}) {
		t.Errorf("RRIntervals = %v", f.RRIntervals)
	}
	if !f.HasUTCOffset || f.UTCOffset != -4*time.Hour {
		t.Errorf("UTCOffset = %v, %v; want -4h (last activity)", f.UTCOffset, f.HasUTCOffset)
	}
}

func TestChainedThreeSequences(t *testing.T) {
	b := fitgen.New()
	for i := 0; i < 3; i++ {
		if i > 0 {
			b.NewSequence()
		}
		b.AddFileID(func(m *mesgdef.FileId) { m.Type = typedef.FileActivity; m.SerialNumber = uint32(i + 1) })
		b.AddSession(func(m *mesgdef.Session) { m.NumLaps = uint16(i + 1) })
	}
	f := decode(t, b)
	if f.FileID.SerialNumber != 1 {
		t.Errorf("FileID serial = %d, want 1", f.FileID.SerialNumber)
	}
	if len(f.Sessions) != 3 {
		t.Fatalf("sessions = %d, want 3", len(f.Sessions))
	}
	for i, s := range f.Sessions {
		if s.NumLaps != uint16(i+1) {
			t.Errorf("session %d NumLaps = %d", i, s.NumLaps)
		}
	}
}

func TestWorkoutOnlyFile(t *testing.T) {
	f := decode(t, fitgen.New().
		AddFileID(func(m *mesgdef.FileId) {
			m.Type = typedef.FileWorkout
			m.Manufacturer = typedef.ManufacturerDevelopment
			m.TimeCreated = t0
		}).
		AddWorkout(func(m *mesgdef.Workout) {
			m.WktName = "Easy 5k"
			m.Sport = typedef.SportRunning
			m.NumValidSteps = 2
		}).
		AddWorkoutStep(func(m *mesgdef.WorkoutStep) {
			m.MessageIndex = 0
			m.DurationType = typedef.WktStepDurationDistance
			m.DurationValue = 400_000
			m.Intensity = typedef.IntensityActive
		}).
		AddWorkoutStep(func(m *mesgdef.WorkoutStep) {
			m.MessageIndex = 1
			m.DurationType = typedef.WktStepDurationTime
			m.DurationValue = 300_000
			m.Intensity = typedef.IntensityCooldown
		}))
	if len(f.Sessions) != 0 || f.Sessions != nil {
		t.Errorf("Sessions = %+v, want nil", f.Sessions)
	}
	if f.FileID.Type != "workout" {
		t.Errorf("FileID.Type = %q", f.FileID.Type)
	}
	if len(f.Workouts) != 1 || f.Workouts[0].WktName != "Easy 5k" || f.Workouts[0].NumValidSteps != 2 {
		t.Errorf("Workouts = %+v", f.Workouts)
	}
	if len(f.WorkoutSteps) != 2 {
		t.Fatalf("WorkoutSteps = %+v", f.WorkoutSteps)
	}
	if m, ok := f.WorkoutSteps[0].DurationDistanceM(); !ok || m != 4000 {
		t.Errorf("step 0 distance = %v, %v", m, ok)
	}
	if d, ok := f.WorkoutSteps[1].DurationTime(); !ok || d != 5*time.Minute {
		t.Errorf("step 1 time = %v, %v", d, ok)
	}
}
