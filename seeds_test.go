package fitparser_test

import (
	"testing"
	"time"

	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
)

// Synthetic inputs shared by the robustness tests and the fuzz seed corpus.

func seedChained(t testing.TB) []byte {
	t.Helper()
	data, err := fitgen.New().
		AddRun(fitgen.Run{Start: t0, Seconds: 4, UTCOffset: -3 * time.Hour}).
		NewSequence().
		AddFileID(func(m *mesgdef.FileId) { m.Type = typedef.FileWorkout }).
		AddWorkout(func(m *mesgdef.Workout) { m.WktName = "W"; m.NumValidSteps = 1 }).
		AddWorkoutStep(func(m *mesgdef.WorkoutStep) {
			m.DurationType = typedef.WktStepDurationTime
			m.DurationValue = 60000
		}).
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Power", "W", basetype.Uint16, nil).
		AddRecord(func(m *mesgdef.Record) { m.Timestamp = t0 }, fitgen.Dev(0, 0, uint16(5))).
		Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func seedAllInvalid(t testing.TB) []byte {
	t.Helper()
	b := fitgen.New()
	for _, n := range allMessageNums {
		b.Add(fitgen.Invalid(n))
	}
	data, err := b.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func seedEverything(t testing.TB) []byte {
	t.Helper()
	data, err := fitgen.New().
		AddFileID(func(m *mesgdef.FileId) {
			m.Type = typedef.FileActivity
			m.Manufacturer = typedef.ManufacturerGarmin
			m.Product = 1
		}).
		AddDeviceInfo(func(m *mesgdef.DeviceInfo) { m.DeviceIndex = 0; m.SoftwareVersion = 100 }).
		AddZonesTarget(func(m *mesgdef.ZonesTarget) { m.MaxHeartRate = 190 }).
		AddHRZone(func(m *mesgdef.HrZone) { m.MessageIndex = 0; m.HighBpm = 120 }).
		AddPowerZone(func(m *mesgdef.PowerZone) { m.MessageIndex = 0; m.HighValue = 200 }).
		AddSplit(func(m *mesgdef.Split) { m.SplitType = typedef.SplitTypeRunActive; m.TotalDistance = 100000 }).
		AddTimeInZone(func(m *mesgdef.TimeInZone) { m.TimeInHrZone = []uint32{1000, 2000} }).
		AddHRV(800, 900).
		AddStrydDescriptions(1).
		AddRecord(func(m *mesgdef.Record) { m.Timestamp = t0; m.RespirationRate = 20 },
			fitgen.StrydValues(1, fitgen.StrydSample{Power: 250, LegSpringStiffness: 9.5})...).
		AddSession(func(m *mesgdef.Session) { m.Sport = typedef.SportRunning; m.TimeInHrZone = []uint32{1, 2} }).
		AddActivity(func(m *mesgdef.Activity) { m.Timestamp = t0; m.LocalTimestamp = t0.Add(time.Hour) }).
		Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return data
}
