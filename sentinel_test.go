package fitparser_test

import (
	"math"
	"reflect"
	"testing"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/profile/untyped/fieldnum"
	"github.com/muktihari/fit/proto"
)

// allMessageNums are the message types the library extracts.
var allMessageNums = []typedef.MesgNum{
	typedef.MesgNumFileId,
	typedef.MesgNumSession,
	typedef.MesgNumLap,
	typedef.MesgNumRecord,
	typedef.MesgNumEvent,
	typedef.MesgNumDeviceInfo,
	typedef.MesgNumSplit,
	typedef.MesgNumTimeInZone,
	typedef.MesgNumWorkout,
	typedef.MesgNumWorkoutStep,
	typedef.MesgNumZonesTarget,
	typedef.MesgNumHrZone,
	typedef.MesgNumPowerZone,
	typedef.MesgNumHrv,
	typedef.MesgNumActivity,
}

// Expected values when every field holds its FIT invalid sentinel: the Go
// zero value, except positions, temperature and indexes (0 is a real value
// there), which keep the FIT invalid value.
var (
	invalidRecord = fitparser.Record{
		PositionLat:  math.MaxInt32,
		PositionLong: math.MaxInt32,
		Temperature:  math.MaxInt8,
	}
	invalidSplit = fitparser.Split{
		StartPositionLat:  math.MaxInt32,
		StartPositionLong: math.MaxInt32,
		EndPositionLat:    math.MaxInt32,
		EndPositionLong:   math.MaxInt32,
	}
	invalidDeviceInfo = fitparser.DeviceInfo{DeviceIndex: math.MaxUint8}
	invalidTimeInZone = fitparser.TimeInZone{ReferenceIndex: math.MaxUint16}
	invalidHRZone     = fitparser.HRZone{MessageIndex: math.MaxUint16}
	invalidPowerZone  = fitparser.PowerZone{MessageIndex: math.MaxUint16}
)

// checkAllInvalid verifies a File decoded from one message of every type in
// allMessageNums, all of whose scalar fields are invalid. Array fields are
// checked by TestSentinelAllInvalidArrayIsNil.
func checkAllInvalid(t *testing.T, f *fitparser.File) {
	t.Helper()
	if f.FileID == nil || *f.FileID != (fitparser.FileID{}) {
		t.Errorf("FileID = %+v, want zero", f.FileID)
	}
	s := one(t, f.Sessions)
	s.TimeInHrZone, s.TimeInPowerZone = nil, nil
	if !reflect.DeepEqual(s, fitparser.Session{}) {
		t.Errorf("Session = %+v, want zero", s)
	}
	if got := one(t, f.Laps); got != (fitparser.Lap{}) {
		t.Errorf("Lap = %+v, want zero", got)
	}
	if got := one(t, f.Records); !reflect.DeepEqual(got, invalidRecord) {
		t.Errorf("Record = %+v, want %+v", got, invalidRecord)
	}
	if got := one(t, f.Events); got != (fitparser.Event{}) {
		t.Errorf("Event = %+v, want zero", got)
	}
	if got := one(t, f.DeviceInfos); got != invalidDeviceInfo {
		t.Errorf("DeviceInfo = %+v, want %+v", got, invalidDeviceInfo)
	}
	if got := one(t, f.Splits); got != invalidSplit {
		t.Errorf("Split = %+v, want %+v", got, invalidSplit)
	}
	tz := one(t, f.TimeInZones)
	tz.TimeInHrZone, tz.TimeInSpeedZone, tz.TimeInCadenceZone, tz.TimeInPowerZone = nil, nil, nil, nil
	tz.HrZoneHighBoundary, tz.PowerZoneHighBoundary = nil, nil
	if !reflect.DeepEqual(tz, invalidTimeInZone) {
		t.Errorf("TimeInZone = %+v, want %+v", tz, invalidTimeInZone)
	}
	if got := one(t, f.Workouts); got != (fitparser.Workout{}) {
		t.Errorf("Workout = %+v, want zero", got)
	}
	if got := one(t, f.WorkoutSteps); got != (fitparser.WorkoutStep{}) {
		t.Errorf("WorkoutStep = %+v, want zero", got)
	}
	if f.ZonesTarget == nil || *f.ZonesTarget != (fitparser.ZonesTarget{}) {
		t.Errorf("ZonesTarget = %+v, want zero", f.ZonesTarget)
	}
	if got := one(t, f.HRZones); got != invalidHRZone {
		t.Errorf("HRZone = %+v, want %+v", got, invalidHRZone)
	}
	if got := one(t, f.PowerZones); got != invalidPowerZone {
		t.Errorf("PowerZone = %+v, want %+v", got, invalidPowerZone)
	}
	if f.RRIntervals != nil {
		t.Errorf("RRIntervals = %v, want nil", f.RRIntervals)
	}
	if f.HasUTCOffset || f.UTCOffset != 0 {
		t.Errorf("UTCOffset = %v, %v; want 0, false", f.UTCOffset, f.HasUTCOffset)
	}

	r := &f.Records[0]
	if _, _, ok := r.Position(); ok {
		t.Error("Record.Position ok = true for invalid position")
	}
	if _, ok := r.TemperatureC(); ok {
		t.Error("Record.TemperatureC ok = true for invalid temperature")
	}
	if _, ok := r.AltitudeM(); ok {
		t.Error("Record.AltitudeM ok = true for invalid altitude")
	}
	sp := &f.Splits[0]
	if _, _, ok := sp.StartPosition(); ok {
		t.Error("Split.StartPosition ok = true for invalid position")
	}
	if _, _, ok := sp.EndPosition(); ok {
		t.Error("Split.EndPosition ok = true for invalid position")
	}
	if _, ok := sp.StartElevationM(); ok {
		t.Error("Split.StartElevationM ok = true for invalid elevation")
	}
}

// TestSentinelExplicit writes every profile field of every extracted
// message with its invalid sentinel present in the bytes.
func TestSentinelExplicit(t *testing.T) {
	b := fitgen.New()
	for _, n := range allMessageNums {
		b.Add(fitgen.Invalid(n))
	}
	checkAllInvalid(t, decode(t, b))
}

// TestSentinelOmitted writes every extracted message with no fields at all.
func TestSentinelOmitted(t *testing.T) {
	// Lenient: the encoder's validator rejects messages without fields.
	b := fitgen.New().Lenient()
	for _, n := range allMessageNums {
		b.Add(proto.Message{Num: n})
	}
	checkAllInvalid(t, decode(t, b))
}

// TestSentinelAllInvalidArrayIsNil checks that an array field whose
// elements are all the FIT invalid value (the FIT definition of an invalid
// array) becomes nil, the zero value of a slice.
func TestSentinelAllInvalidArrayIsNil(t *testing.T) {
	f := decode(t, fitgen.New().
		Add(fitgen.Invalid(typedef.MesgNumSession)).
		Add(fitgen.Invalid(typedef.MesgNumTimeInZone)))
	s := one(t, f.Sessions)
	tz := one(t, f.TimeInZones)
	for name, v := range map[string]any{
		"Session.TimeInHrZone":             s.TimeInHrZone,
		"Session.TimeInPowerZone":          s.TimeInPowerZone,
		"TimeInZone.TimeInHrZone":          tz.TimeInHrZone,
		"TimeInZone.TimeInSpeedZone":       tz.TimeInSpeedZone,
		"TimeInZone.TimeInCadenceZone":     tz.TimeInCadenceZone,
		"TimeInZone.TimeInPowerZone":       tz.TimeInPowerZone,
		"TimeInZone.HrZoneHighBoundary":    tz.HrZoneHighBoundary,
		"TimeInZone.PowerZoneHighBoundary": tz.PowerZoneHighBoundary,
	} {
		if !reflect.ValueOf(v).IsNil() {
			t.Errorf("%s = %v, want nil for an all-invalid array", name, v)
		}
	}
}

// TestSentinelPerField sets one field at a time to its invalid sentinel
// explicitly, next to a valid heart rate, to show that each filter is
// independent.
func TestSentinelPerField(t *testing.T) {
	type sessionCase struct {
		num   byte
		value proto.Value
		get   func(*fitparser.Session) any
	}
	cases := map[string]sessionCase{
		"TotalAscent":            {fieldnum.SessionTotalAscent, proto.Uint16(basetype.Uint16Invalid), func(s *fitparser.Session) any { return s.TotalAscent }},
		"TotalDescent":           {fieldnum.SessionTotalDescent, proto.Uint16(basetype.Uint16Invalid), func(s *fitparser.Session) any { return s.TotalDescent }},
		"TotalCalories":          {fieldnum.SessionTotalCalories, proto.Uint16(basetype.Uint16Invalid), func(s *fitparser.Session) any { return s.TotalCalories }},
		"TotalTrainingEffect":    {fieldnum.SessionTotalTrainingEffect, proto.Uint8(basetype.Uint8Invalid), func(s *fitparser.Session) any { return s.TotalTrainingEffect }},
		"AvgStanceTime":          {fieldnum.SessionAvgStanceTime, proto.Uint16(basetype.Uint16Invalid), func(s *fitparser.Session) any { return s.AvgStanceTime }},
		"AvgVerticalOscillation": {fieldnum.SessionAvgVerticalOscillation, proto.Uint16(basetype.Uint16Invalid), func(s *fitparser.Session) any { return s.AvgVerticalOscillation }},
		"AvgPosGrade":            {fieldnum.SessionAvgPosGrade, proto.Int16(basetype.Sint16Invalid), func(s *fitparser.Session) any { return s.AvgPosGrade }},
		"TrainingLoadPeak":       {fieldnum.SessionTrainingLoadPeak, proto.Int32(basetype.Sint32Invalid), func(s *fitparser.Session) any { return s.TrainingLoadPeak }},
		"TotalDistance":          {fieldnum.SessionTotalDistance, proto.Uint32(basetype.Uint32Invalid), func(s *fitparser.Session) any { return s.TotalDistance }},
		"StartTime":              {fieldnum.SessionStartTime, proto.Uint32(basetype.Uint32Invalid), func(s *fitparser.Session) any { return s.StartTime.IsZero() }},
		"Sport":                  {fieldnum.SessionSport, proto.Uint8(basetype.EnumInvalid), func(s *fitparser.Session) any { return s.Sport }},
	}
	zero := map[string]any{
		"TotalAscent": uint16(0), "TotalDescent": uint16(0), "TotalCalories": uint16(0),
		"TotalTrainingEffect": uint8(0), "AvgStanceTime": uint16(0), "AvgVerticalOscillation": uint16(0),
		"AvgPosGrade": int16(0), "TrainingLoadPeak": int32(0), "TotalDistance": uint32(0),
		"StartTime": true, "Sport": "",
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			s := mesgdef.NewSession(nil)
			s.AvgHeartRate = 150
			m := s.ToMesg(nil)
			fitgen.SetField(&m, c.num, c.value)
			f := decode(t, fitgen.New().Add(m))
			got := one(t, f.Sessions)
			if v := c.get(&got); v != zero[name] {
				t.Errorf("%s = %v, want %v", name, v, zero[name])
			}
			if got.AvgHeartRate != 150 {
				t.Errorf("AvgHeartRate = %d, want 150", got.AvgHeartRate)
			}
		})
	}
}

func TestSentinelSerialNumber(t *testing.T) {
	for _, sn := range []uint32{basetype.Uint32zInvalid, basetype.Uint32Invalid} {
		fid := mesgdef.NewFileId(nil)
		fid.Type = typedef.FileActivity
		m := fid.ToMesg(nil)
		fitgen.SetField(&m, fieldnum.FileIdSerialNumber, proto.Uint32(sn))
		di := mesgdef.NewDeviceInfo(nil)
		di.DeviceIndex = 0
		dm := di.ToMesg(nil)
		fitgen.SetField(&dm, fieldnum.DeviceInfoSerialNumber, proto.Uint32(sn))
		f := decode(t, fitgen.New().Add(m, dm))
		if f.FileID.SerialNumber != 0 {
			t.Errorf("FileID.SerialNumber(%#x) = %d, want 0", sn, f.FileID.SerialNumber)
		}
		if got := one(t, f.DeviceInfos).SerialNumber; got != 0 {
			t.Errorf("DeviceInfo.SerialNumber(%#x) = %d, want 0", sn, got)
		}
	}
}

func TestSentinelStringFields(t *testing.T) {
	f := decode(t, fitgen.New().
		AddWorkout(func(m *mesgdef.Workout) { m.NumValidSteps = 1 }).
		AddWorkoutStep(func(m *mesgdef.WorkoutStep) { m.DurationValue = 5 }).
		AddHRZone(func(m *mesgdef.HrZone) { m.HighBpm = 100 }).
		AddDeviceInfo(func(m *mesgdef.DeviceInfo) { m.DeviceIndex = 1 }))
	if w := one(t, f.Workouts); w.WktName != "" || w.WktDescription != "" {
		t.Errorf("workout strings = %q, %q", w.WktName, w.WktDescription)
	}
	if s := one(t, f.WorkoutSteps); s.WktStepName != "" {
		t.Errorf("step name = %q", s.WktStepName)
	}
	if z := one(t, f.HRZones); z.Name != "" {
		t.Errorf("zone name = %q", z.Name)
	}
	if d := one(t, f.DeviceInfos); d.ProductName != "" || d.Manufacturer != "" {
		t.Errorf("device = %+v", d)
	}
}
