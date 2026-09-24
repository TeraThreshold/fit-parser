package fitparser_test

import (
	"reflect"
	"testing"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
)

// These tests set every extracted field of a message to a distinct valid
// value and check that Decode returns exactly those values.

func TestRoundTripFileID(t *testing.T) {
	f := decode(t, fitgen.New().AddFileID(func(m *mesgdef.FileId) {
		m.Type = typedef.FileActivity
		m.Manufacturer = typedef.ManufacturerGarmin
		m.Product = uint16(typedef.GarminProductFr965)
		m.SerialNumber = 1234567890
		m.TimeCreated = t0
	}))
	want := &fitparser.FileID{
		Type:         "activity",
		Manufacturer: "garmin",
		Product:      uint16(typedef.GarminProductFr965),
		ProductName:  "fr965",
		SerialNumber: 1234567890,
		TimeCreated:  t0,
	}
	if !reflect.DeepEqual(f.FileID, want) {
		t.Errorf("FileID =\n%+v\nwant\n%+v", f.FileID, want)
	}
	if f.FileID.TimeCreated.Location() != time.UTC {
		t.Errorf("TimeCreated location = %v, want UTC", f.FileID.TimeCreated.Location())
	}
}

func TestRoundTripSession(t *testing.T) {
	f := decode(t, fitgen.New().AddSession(func(m *mesgdef.Session) {
		m.Timestamp = t0.Add(time.Hour)
		m.Sport = typedef.SportRunning
		m.SubSport = typedef.SubSportTrail
		m.StartTime = t0
		m.TotalElapsedTime = 3_700_000
		m.TotalTimerTime = 3_600_000
		m.TotalMovingTime = 3_500_000
		m.TotalDistance = 1_234_567
		m.EnhancedAvgSpeed = 3429
		m.EnhancedMaxSpeed = 5123
		m.TotalAscent = 321
		m.TotalDescent = 318
		m.TotalCalories = 845
		m.TotalFatCalories = 120
		m.TotalTrainingEffect = 34
		m.TotalAnaerobicTrainingEffect = 12
		m.AvgHeartRate = 152
		m.MaxHeartRate = 181
		m.MinHeartRate = 95
		m.AvgCadence = 86
		m.MaxCadence = 101
		m.MaxFractionalCadence = 64
		m.AvgPower = 265
		m.MaxPower = 612
		m.NormalizedPower = 278
		m.TrainingStressScore = 875
		m.IntensityFactor = 912
		m.TotalWork = 954_000
		m.AvgStanceTime = 2451
		m.AvgVerticalOscillation = 873
		m.AvgStepLength = 11234
		m.AvgVerticalRatio = 812
		m.AvgStanceTimePercent = 3456
		m.AvgStanceTimeBalance = 4987
		m.RmssdHrv = 42
		m.SdrrHrv = 55
		m.EnhancedAvgRespirationRate = 3125
		m.EnhancedMaxRespirationRate = 4550
		m.EnhancedMinRespirationRate = 1875
		m.AvgSpo2 = 97
		m.AvgStress = 31
		m.WorkoutRpe = 70
		m.WorkoutFeel = 75
		m.AvgCoreTemperature = 3812
		m.MaxCoreTemperature = 3905
		m.AvgPosGrade = 312
		m.AvgNegGrade = -298
		m.MaxPosGrade = 1450
		m.MaxNegGrade = -1320
		m.MinAltitude = 2750 // 50 m
		m.MaxAltitude = 3400 // 180 m
		m.AvgVam = 512
		m.NumLaps = 13
		m.TotalCycles = 5432
		m.TimeInHrZone = []uint32{60000, 120000, 900000, 1800000, 720000}
		m.TimeInPowerZone = []uint32{30000, basetype.Uint32Invalid, 45000}
		m.TrainingLoadPeak = 8_061_952
		m.PoolLength = 2500
		m.NumActiveLengths = 40
		m.AvgStrokeCount = 185
		m.AvgStrokeDistance = 215
		m.SwimStroke = typedef.SwimStrokeFreestyle
	}))
	want := fitparser.Session{
		Sport:                        "running",
		SubSport:                     "trail",
		StartTime:                    t0,
		TotalElapsedTime:             3_700_000,
		TotalTimerTime:               3_600_000,
		TotalMovingTime:              3_500_000,
		TotalDistance:                1_234_567,
		EnhancedAvgSpeed:             3429,
		EnhancedMaxSpeed:             5123,
		TotalAscent:                  321,
		TotalDescent:                 318,
		TotalCalories:                845,
		TotalFatCalories:             120,
		TotalTrainingEffect:          34,
		TotalAnaerobicTrainingEffect: 12,
		AvgHeartRate:                 152,
		MaxHeartRate:                 181,
		MinHeartRate:                 95,
		AvgCadence:                   86,
		MaxCadence:                   101,
		MaxFractionalCadence:         64,
		AvgPower:                     265,
		MaxPower:                     612,
		NormalizedPower:              278,
		TrainingStressScore:          875,
		IntensityFactor:              912,
		TotalWork:                    954_000,
		AvgStanceTime:                2451,
		AvgVerticalOscillation:       873,
		AvgStepLength:                11234,
		AvgVerticalRatio:             812,
		AvgStanceTimePercent:         3456,
		AvgStanceTimeBalance:         4987,
		RmssdHrv:                     42,
		SdrrHrv:                      55,
		EnhancedAvgRespirationRate:   3125,
		EnhancedMaxRespirationRate:   4550,
		EnhancedMinRespirationRate:   1875,
		AvgSpo2:                      97,
		AvgStress:                    31,
		WorkoutRpe:                   70,
		WorkoutFeel:                  75,
		AvgCoreTemperature:           3812,
		MaxCoreTemperature:           3905,
		AvgPosGrade:                  312,
		AvgNegGrade:                  -298,
		MaxPosGrade:                  1450,
		MaxNegGrade:                  -1320,
		MinAltitude:                  2750,
		MaxAltitude:                  3400,
		AvgVam:                       512,
		NumLaps:                      13,
		TotalCycles:                  5432,
		TimeInHrZone:                 []uint32{60000, 120000, 900000, 1800000, 720000},
		TimeInPowerZone:              []uint32{30000, 0, 45000}, // invalid element zeroed in place
		TrainingLoadPeak:             8_061_952,
		PoolLength:                   2500,
		NumActiveLengths:             40,
		AvgStrokeCount:               185,
		AvgStrokeDistance:            215,
		SwimStroke:                   "freestyle",
	}
	if got := one(t, f.Sessions); !reflect.DeepEqual(got, want) {
		t.Errorf("Session =\n%+v\nwant\n%+v", got, want)
	}
}

func TestRoundTripSessionMaxFractionalCadenceNeedsMaxCadence(t *testing.T) {
	f := decode(t, fitgen.New().AddSession(func(m *mesgdef.Session) {
		m.MaxFractionalCadence = 64 // no MaxCadence
	}))
	if got := one(t, f.Sessions).MaxFractionalCadence; got != 0 {
		t.Errorf("MaxFractionalCadence = %d, want 0 without MaxCadence", got)
	}
}

// TestRoundTripSessionAltitudeSources: the altitude range is read from the
// enhanced fields, which the decoder also fills from the legacy ones, so
// files that write either kind are covered.
func TestRoundTripSessionAltitudeSources(t *testing.T) {
	for _, tc := range []struct {
		name     string
		fn       func(*mesgdef.Session)
		min, max uint16
	}{
		{"legacy only", func(m *mesgdef.Session) { m.MinAltitude = 3000; m.MaxAltitude = 3750 }, 3000, 3750},
		{"enhanced only", func(m *mesgdef.Session) { m.EnhancedMinAltitude = 3000; m.EnhancedMaxAltitude = 3750 }, 3000, 3750},
		{"enhanced above 16 bits", func(m *mesgdef.Session) { m.EnhancedMinAltitude = 3000; m.EnhancedMaxAltitude = 70000 }, 3000, 0},
		{"neither", func(m *mesgdef.Session) { m.TotalDistance = 1 }, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := one(t, decode(t, fitgen.New().AddSession(tc.fn)).Sessions)
			if s.MinAltitude != tc.min || s.MaxAltitude != tc.max {
				t.Errorf("MinAltitude, MaxAltitude = %d, %d, want %d, %d", s.MinAltitude, s.MaxAltitude, tc.min, tc.max)
			}
		})
	}
}

func TestRoundTripLap(t *testing.T) {
	f := decode(t, fitgen.New().AddLap(func(m *mesgdef.Lap) {
		m.Timestamp = t0.Add(10 * time.Minute)
		m.StartTime = t0
		m.TotalElapsedTime = 600_500
		m.TotalTimerTime = 598_250
		m.TotalDistance = 201_000
		m.AvgHeartRate = 149
		m.MaxHeartRate = 171
		m.AvgCadence = 88
		m.MaxCadence = 97
		m.AvgPower = 271
		m.MaxPower = 455
		m.TotalAscent = 42
		m.TotalDescent = 37
		m.TotalCalories = 151
		m.EnhancedAvgSpeed = 3350
		m.EnhancedMaxSpeed = 4875
		m.AvgStanceTime = 2388
		m.AvgVerticalOscillation = 842
		m.AvgStepLength = 11456
		m.AvgVerticalRatio = 735
		m.LapTrigger = typedef.LapTriggerDistance
		m.Sport = typedef.SportRunning
	}))
	want := fitparser.Lap{
		StartTime:              t0,
		TotalElapsedTime:       600_500,
		TotalTimerTime:         598_250,
		TotalDistance:          201_000,
		AvgHeartRate:           149,
		MaxHeartRate:           171,
		AvgCadence:             88,
		MaxCadence:             97,
		AvgPower:               271,
		MaxPower:               455,
		TotalAscent:            42,
		TotalDescent:           37,
		TotalCalories:          151,
		EnhancedAvgSpeed:       3350,
		EnhancedMaxSpeed:       4875,
		AvgStanceTime:          2388,
		AvgVerticalOscillation: 842,
		AvgStepLength:          11456,
		AvgVerticalRatio:       735,
		LapTrigger:             "distance",
		Sport:                  "running",
	}
	if got := one(t, f.Laps); !reflect.DeepEqual(got, want) {
		t.Errorf("Lap =\n%+v\nwant\n%+v", got, want)
	}
}

func TestRoundTripRecord(t *testing.T) {
	lat, lon := fitgen.Degrees(-33.5), fitgen.Degrees(151.25)
	f := decode(t, fitgen.New().AddRecord(func(m *mesgdef.Record) {
		m.Timestamp = t0.Add(42 * time.Second)
		m.HeartRate = 163
		m.Cadence = 91
		m.Power = 318
		m.EnhancedSpeed = 4125
		m.Distance = 17_345
		m.EnhancedAltitude = fitgen.Altitude(-12.4)
		m.Grade = -250
		m.PositionLat = lat
		m.PositionLong = lon
		m.Temperature = -4
		m.StanceTime = 2215
		m.VerticalOscillation = 912
		m.StepLength = 13250
		m.VerticalRatio = 688
		m.StanceTimeBalance = 4921
		m.EnhancedRespirationRate = 4275
		m.GpsAccuracy = 3
	}))
	want := fitparser.Record{
		Timestamp:               t0.Add(42 * time.Second),
		HeartRate:               163,
		Cadence:                 91,
		Power:                   318,
		EnhancedSpeed:           4125,
		Distance:                17_345,
		EnhancedAltitude:        fitgen.Altitude(-12.4),
		Grade:                   -250,
		PositionLat:             lat,
		PositionLong:            lon,
		Temperature:             -4,
		StanceTime:              2215,
		VerticalOscillation:     912,
		StepLength:              13250,
		VerticalRatio:           688,
		StanceTimeBalance:       4921,
		EnhancedRespirationRate: 4275,
		GpsAccuracy:             3,
	}
	if got := one(t, f.Records); !reflect.DeepEqual(got, want) {
		t.Errorf("Record =\n%+v\nwant\n%+v", got, want)
	}
}

func TestRoundTripRecordZeroPositionAndTemperatureAreValid(t *testing.T) {
	f := decode(t, fitgen.New().AddRecord(func(m *mesgdef.Record) {
		m.Timestamp = t0
		m.PositionLat = 0
		m.PositionLong = 0
		m.Temperature = 0
	}))
	r := one(t, f.Records)
	if lat, lon, ok := r.Position(); !ok || lat != 0 || lon != 0 {
		t.Errorf("Position() = %v, %v, %v; want 0, 0, true", lat, lon, ok)
	}
	if c, ok := r.TemperatureC(); !ok || c != 0 {
		t.Errorf("TemperatureC() = %v, %v; want 0, true", c, ok)
	}
}

func TestRoundTripEvent(t *testing.T) {
	f := decode(t, fitgen.New().AddEvent(func(m *mesgdef.Event) {
		m.Timestamp = t0.Add(5 * time.Second)
		m.Event = typedef.EventTimer
		m.EventType = typedef.EventTypeStopAll
	}))
	want := fitparser.Event{Timestamp: t0.Add(5 * time.Second), Event: "timer", EventType: "stop_all"}
	if got := one(t, f.Events); !reflect.DeepEqual(got, want) {
		t.Errorf("Event = %+v, want %+v", got, want)
	}
}

func TestRoundTripDeviceInfo(t *testing.T) {
	f := decode(t, fitgen.New().AddDeviceInfo(func(m *mesgdef.DeviceInfo) {
		m.Timestamp = t0
		m.DeviceIndex = 2
		m.Manufacturer = typedef.ManufacturerStryd
		m.Product = 7
		m.ProductName = "Synthetic Pod"
		m.SerialNumber = 2000000002
		m.SoftwareVersion = 1234
		m.BatteryLevel = 77
	}))
	want := fitparser.DeviceInfo{
		DeviceIndex:     2,
		Manufacturer:    "stryd",
		ProductName:     "Synthetic Pod",
		SerialNumber:    2000000002,
		SoftwareVersion: 12.34,
		BatteryLevel:    77,
	}
	if got := one(t, f.DeviceInfos); !reflect.DeepEqual(got, want) {
		t.Errorf("DeviceInfo = %+v, want %+v", got, want)
	}
}

func TestRoundTripDeviceInfoCreatorIndexZero(t *testing.T) {
	f := decode(t, fitgen.New().AddDeviceInfo(func(m *mesgdef.DeviceInfo) {
		m.DeviceIndex = typedef.DeviceIndexCreator
	}).AddDeviceInfo(func(m *mesgdef.DeviceInfo) {
		m.BatteryLevel = 50 // no device index
	}))
	if len(f.DeviceInfos) != 2 {
		t.Fatalf("device infos = %d, want 2", len(f.DeviceInfos))
	}
	if got := f.DeviceInfos[0].DeviceIndex; got != 0 {
		t.Errorf("creator DeviceIndex = %d, want 0", got)
	}
	if got := f.DeviceInfos[1].DeviceIndex; got != 255 {
		t.Errorf("absent DeviceIndex = %d, want 255 (kept FIT invalid)", got)
	}
}

func TestRoundTripSplit(t *testing.T) {
	sLat, sLon := fitgen.Degrees(45.5), fitgen.Degrees(-122.25)
	eLat, eLon := fitgen.Degrees(45.75), fitgen.Degrees(-122.5)
	f := decode(t, fitgen.New().AddSplit(func(m *mesgdef.Split) {
		m.SplitType = typedef.SplitTypeRunActive
		m.StartTime = t0
		m.TotalElapsedTime = 245_000
		m.TotalTimerTime = 240_000
		m.TotalMovingTime = 238_500
		m.TotalDistance = 100_000
		m.AvgSpeed = 4166
		m.MaxSpeed = 5012
		m.TotalAscent = 12
		m.TotalDescent = 9
		m.TotalCalories = 66
		m.StartElevation = fitgen.Altitude(12.4)
		m.StartPositionLat = sLat
		m.StartPositionLong = sLon
		m.EndPositionLat = eLat
		m.EndPositionLong = eLon
	}))
	want := fitparser.Split{
		SplitType:         "run_active",
		StartTime:         t0,
		TotalElapsedTime:  245_000,
		TotalTimerTime:    240_000,
		TotalMovingTime:   238_500,
		TotalDistance:     100_000,
		AvgSpeed:          4166,
		MaxSpeed:          5012,
		TotalAscent:       12,
		TotalDescent:      9,
		TotalCalories:     66,
		StartElevation:    fitgen.Altitude(12.4),
		StartPositionLat:  sLat,
		StartPositionLong: sLon,
		EndPositionLat:    eLat,
		EndPositionLong:   eLon,
	}
	if got := one(t, f.Splits); !reflect.DeepEqual(got, want) {
		t.Errorf("Split =\n%+v\nwant\n%+v", got, want)
	}
}

func TestRoundTripTimeInZone(t *testing.T) {
	f := decode(t, fitgen.New().AddTimeInZone(func(m *mesgdef.TimeInZone) {
		m.Timestamp = t0
		m.ReferenceMesg = typedef.MesgNumSession
		m.ReferenceIndex = 0
		m.TimeInHrZone = []uint32{1000, 2000, 3000}
		m.TimeInSpeedZone = []uint32{4000, 5000}
		m.TimeInCadenceZone = []uint32{6000}
		m.TimeInPowerZone = []uint32{7000, basetype.Uint32Invalid, 9000}
		m.HrZoneHighBoundary = []uint8{120, 140, basetype.Uint8Invalid, 180}
		m.PowerZoneHighBoundary = []uint16{150, 220, 300}
		m.MaxHeartRate = 192
		m.RestingHeartRate = 44
		m.ThresholdHeartRate = 171
		m.FunctionalThresholdPower = 285
	}))
	want := fitparser.TimeInZone{
		ReferenceMesg:            "session",
		ReferenceIndex:           0,
		TimeInHrZone:             []uint32{1000, 2000, 3000},
		TimeInSpeedZone:          []uint32{4000, 5000},
		TimeInCadenceZone:        []uint32{6000},
		TimeInPowerZone:          []uint32{7000, 0, 9000},
		HrZoneHighBoundary:       []uint8{120, 140, 0, 180},
		PowerZoneHighBoundary:    []uint16{150, 220, 300},
		MaxHeartRate:             192,
		RestingHeartRate:         44,
		ThresholdHeartRate:       171,
		FunctionalThresholdPower: 285,
	}
	if got := one(t, f.TimeInZones); !reflect.DeepEqual(got, want) {
		t.Errorf("TimeInZone =\n%+v\nwant\n%+v", got, want)
	}
}

func TestRoundTripTimeInZoneReferenceIndexMasked(t *testing.T) {
	f := decode(t, fitgen.New().AddTimeInZone(func(m *mesgdef.TimeInZone) {
		m.ReferenceMesg = typedef.MesgNumLap
		m.ReferenceIndex = 4 | typedef.MessageIndexSelected
	}))
	if got := one(t, f.TimeInZones).ReferenceIndex; got != 4 {
		t.Errorf("ReferenceIndex = %d, want 4 (selected bit stripped)", got)
	}
}

func TestRoundTripWorkout(t *testing.T) {
	f := decode(t, fitgen.New().AddWorkout(func(m *mesgdef.Workout) {
		m.WktName = "Tempo 3x10"
		m.WktDescription = "Three blocks at threshold"
		m.Sport = typedef.SportSwimming
		m.SubSport = typedef.SubSportLapSwimming
		m.NumValidSteps = 7
		m.PoolLength = 2500
	}))
	want := fitparser.Workout{
		WktName:        "Tempo 3x10",
		WktDescription: "Three blocks at threshold",
		Sport:          "swimming",
		SubSport:       "lap_swimming",
		NumValidSteps:  7,
		PoolLength:     2500,
	}
	if got := one(t, f.Workouts); !reflect.DeepEqual(got, want) {
		t.Errorf("Workout = %+v, want %+v", got, want)
	}
}

func TestRoundTripWorkoutStep(t *testing.T) {
	f := decode(t, fitgen.New().AddWorkoutStep(func(m *mesgdef.WorkoutStep) {
		m.WktStepName = "Threshold"
		m.DurationType = typedef.WktStepDurationTime
		m.DurationValue = 600_000
		m.TargetType = typedef.WktStepTargetHeartRate
		m.TargetValue = 0
		m.CustomTargetValueLow = 255
		m.CustomTargetValueHigh = 270
		m.Intensity = typedef.IntensityActive
	}))
	want := fitparser.WorkoutStep{
		WktStepName:           "Threshold",
		DurationType:          "time",
		DurationValue:         600_000,
		TargetType:            "heart_rate",
		TargetValue:           0,
		CustomTargetValueLow:  255,
		CustomTargetValueHigh: 270,
		Intensity:             "active",
	}
	if got := one(t, f.WorkoutSteps); !reflect.DeepEqual(got, want) {
		t.Errorf("WorkoutStep = %+v, want %+v", got, want)
	}
}

func TestRoundTripZonesTarget(t *testing.T) {
	f := decode(t, fitgen.New().AddZonesTarget(func(m *mesgdef.ZonesTarget) {
		m.MaxHeartRate = 193
		m.ThresholdHeartRate = 172
		m.FunctionalThresholdPower = 301
		m.HrCalcType = typedef.HrZoneCalcPercentLthr
		m.PwrCalcType = typedef.PwrZoneCalcPercentFtp
	}))
	want := &fitparser.ZonesTarget{
		MaxHeartRate:             193,
		ThresholdHeartRate:       172,
		FunctionalThresholdPower: 301,
		HrCalcType:               "percent_lthr",
		PwrCalcType:              "percent_ftp",
	}
	if !reflect.DeepEqual(f.ZonesTarget, want) {
		t.Errorf("ZonesTarget = %+v, want %+v", f.ZonesTarget, want)
	}
}

func TestRoundTripHRAndPowerZones(t *testing.T) {
	f := decode(t, fitgen.New().
		AddHRZone(func(m *mesgdef.HrZone) { m.MessageIndex = 0; m.Name = "Z1"; m.HighBpm = 125 }).
		AddHRZone(func(m *mesgdef.HrZone) {
			m.MessageIndex = 1 | typedef.MessageIndexSelected
			m.Name = "Z2"
			m.HighBpm = 145
		}).
		AddPowerZone(func(m *mesgdef.PowerZone) { m.MessageIndex = 0; m.Name = "Recovery"; m.HighValue = 160 }).
		AddPowerZone(func(m *mesgdef.PowerZone) { m.MessageIndex = 5; m.Name = "VO2"; m.HighValue = 2100 }))
	wantHR := []fitparser.HRZone{
		{MessageIndex: 0, Name: "Z1", HighBpm: 125},
		{MessageIndex: 1, Name: "Z2", HighBpm: 145},
	}
	wantPwr := []fitparser.PowerZone{
		{MessageIndex: 0, Name: "Recovery", HighValue: 160},
		{MessageIndex: 5, Name: "VO2", HighValue: 2100},
	}
	if !reflect.DeepEqual(f.HRZones, wantHR) {
		t.Errorf("HRZones = %+v, want %+v", f.HRZones, wantHR)
	}
	if !reflect.DeepEqual(f.PowerZones, wantPwr) {
		t.Errorf("PowerZones = %+v, want %+v", f.PowerZones, wantPwr)
	}
}

func TestRoundTripOrderPreserved(t *testing.T) {
	b := fitgen.New()
	for i := 0; i < 5; i++ {
		b.AddRecord(func(m *mesgdef.Record) {
			m.Timestamp = t0.Add(time.Duration(i) * time.Second)
			m.HeartRate = uint8(100 + i)
		})
		b.AddLap(func(m *mesgdef.Lap) { m.TotalDistance = uint32(i) * 1000 })
	}
	f := decode(t, b)
	if len(f.Records) != 5 || len(f.Laps) != 5 {
		t.Fatalf("records=%d laps=%d, want 5/5", len(f.Records), len(f.Laps))
	}
	for i := range f.Records {
		if f.Records[i].HeartRate != uint8(100+i) || !f.Records[i].Timestamp.Equal(t0.Add(time.Duration(i)*time.Second)) {
			t.Errorf("record %d = %+v", i, f.Records[i])
		}
		if f.Laps[i].TotalDistance != uint32(i)*1000 {
			t.Errorf("lap %d distance = %d", i, f.Laps[i].TotalDistance)
		}
	}
}
