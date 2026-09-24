package fitgen_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/profile/untyped/fieldnum"
	"github.com/muktihari/fit/proto"
)

// decodeAll decodes every sequence of data with the upstream decoder.
func decodeAll(t *testing.T, data []byte) []*proto.FIT {
	t.Helper()
	dec := decoder.New(bytes.NewReader(data))
	var fits []*proto.FIT
	for dec.Next() {
		fit, err := dec.Decode()
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		fits = append(fits, fit)
	}
	return fits
}

func TestBuilderRoundTrip(t *testing.T) {
	ts := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	b := fitgen.New().
		AddFileID(func(m *mesgdef.FileId) { m.Type = typedef.FileActivity; m.TimeCreated = ts }).
		AddDeviceInfo(func(m *mesgdef.DeviceInfo) { m.DeviceIndex = 1 }).
		AddZonesTarget(func(m *mesgdef.ZonesTarget) { m.MaxHeartRate = 190 }).
		AddHRZone(func(m *mesgdef.HrZone) { m.MessageIndex = 0; m.HighBpm = 120 }).
		AddPowerZone(func(m *mesgdef.PowerZone) { m.MessageIndex = 0; m.HighValue = 200 }).
		AddEvent(func(m *mesgdef.Event) { m.Timestamp = ts; m.Event = typedef.EventTimer }).
		AddDeveloperDataID(3, nil).
		AddFieldDescription(3, 0, "Power", "W", basetype.Uint16, nil).
		AddRecord(func(m *mesgdef.Record) { m.Timestamp = ts; m.Speed = 2500 }, fitgen.Dev(3, 0, uint16(300))).
		AddLap(func(m *mesgdef.Lap) { m.StartTime = ts }).
		AddSession(func(m *mesgdef.Session) { m.Sport = typedef.SportRunning }).
		AddHRV(800, 0xFFFF).
		AddSplit(func(m *mesgdef.Split) { m.SplitType = typedef.SplitTypeRunActive }).
		AddTimeInZone(func(m *mesgdef.TimeInZone) { m.TimeInHrZone = []uint32{1000, 2000} }).
		AddWorkout(func(m *mesgdef.Workout) { m.WktName = "Intervals" }).
		AddWorkoutStep(func(m *mesgdef.WorkoutStep) { m.DurationType = typedef.WktStepDurationTime; m.DurationValue = 60000 }).
		AddActivity(func(m *mesgdef.Activity) { m.Timestamp = ts; m.LocalTimestamp = ts.Add(time.Hour) })

	// Explicit invalid value in the bytes.
	rec := mesgdef.NewRecord(nil)
	rec.Timestamp = ts
	m := rec.ToMesg(nil)
	fitgen.SetField(&m, fieldnum.RecordPower, proto.Uint16(basetype.Uint16Invalid))
	b.Add(m)

	b.NewSequence().AddFileID(func(m *mesgdef.FileId) { m.Type = typedef.FileWorkout })

	data := b.MustBytes()
	if again := b.MustBytes(); !bytes.Equal(data, again) {
		t.Fatal("Bytes is not repeatable")
	}
	fits := decodeAll(t, data)
	if len(fits) != 2 {
		t.Fatalf("sequences = %d, want 2", len(fits))
	}
	if got := len(fits[0].Messages); got != 18 {
		t.Errorf("messages in sequence 1 = %d, want 18", got)
	}
	var sawDev, sawInvalidPower bool
	for i := range fits[0].Messages {
		msg := &fits[0].Messages[i]
		if msg.Num != typedef.MesgNumRecord {
			continue
		}
		if len(msg.DeveloperFields) == 1 && msg.DeveloperFields[0].Value.Uint16() == 300 {
			sawDev = true
		}
		if f := msg.FieldByNum(fieldnum.RecordPower); f != nil && f.Value.Uint16() == basetype.Uint16Invalid {
			sawInvalidPower = true
		}
	}
	if !sawDev || !sawInvalidPower {
		t.Errorf("dev=%v invalidPower=%v", sawDev, sawInvalidPower)
	}
}

func TestBuilderLenientAllowsUndeclaredDeveloperFields(t *testing.T) {
	strict := fitgen.New().AddRecord(func(m *mesgdef.Record) { m.HeartRate = 100 }, fitgen.Dev(7, 1, uint8(5)))
	if _, err := strict.Bytes(); err == nil {
		t.Fatal("expected an error for a developer field without developer_data_id")
	}
	lenient := fitgen.New().Lenient().AddRecord(func(m *mesgdef.Record) { m.HeartRate = 100 }, fitgen.Dev(7, 1, uint8(5)))
	if _, err := lenient.Bytes(); err != nil {
		t.Fatalf("lenient: %v", err)
	}
}

func TestInvalidMessageHoldsSentinels(t *testing.T) {
	m := fitgen.Invalid(typedef.MesgNumRecord)
	if len(m.Fields) == 0 {
		t.Fatal("no fields")
	}
	for _, f := range m.Fields {
		if f.Value.Valid(f.BaseType) {
			t.Errorf("field %s (%d) = %v is valid", f.Name, f.Num, f.Value.Any())
		}
	}
	b := fitgen.New()
	for _, n := range []typedef.MesgNum{typedef.MesgNumSession, typedef.MesgNumTimeInZone, typedef.MesgNumRecord, typedef.MesgNumFieldDescription} {
		b.Add(fitgen.Invalid(n))
	}
	fits := decodeAll(t, b.MustBytes())
	if len(fits) != 1 || len(fits[0].Messages) != 4 {
		t.Fatalf("decoded %d sequences", len(fits))
	}
}

func TestFixCRC(t *testing.T) {
	data := fitgen.New().
		AddFileID(func(m *mesgdef.FileId) { m.Type = typedef.FileActivity }).
		NewSequence().
		AddFileID(func(m *mesgdef.FileId) { m.Type = typedef.FileWorkout }).
		MustBytes()
	if got := fitgen.FixCRC(data); !bytes.Equal(got, data) {
		t.Fatal("FixCRC changed a valid file")
	}
	broken := append([]byte(nil), data...)
	broken[12] ^= 0xFF            // header CRC of sequence 1
	broken[len(broken)-1] ^= 0xFF // file CRC of sequence 2
	fixed := fitgen.FixCRC(broken)
	if !bytes.Equal(fixed, data) {
		t.Fatal("FixCRC did not restore the CRCs")
	}
	if broken[12] == data[12] {
		t.Fatal("FixCRC modified its input")
	}
	for _, in := range [][]byte{nil, {1}, {14, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0x7F}, data[:20]} {
		if got := fitgen.FixCRC(in); !bytes.Equal(got, in) {
			t.Errorf("FixCRC(%v) = %v, want unchanged", in, got)
		}
	}
	if len(decodeAll(t, fixed)) != 2 {
		t.Error("fixed file does not decode")
	}
}

func TestAddRun(t *testing.T) {
	start := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	data := fitgen.New().
		AddStrydDescriptions(1).
		AddRun(fitgen.Run{
			Start: start, Seconds: 3, UTCOffset: time.Hour,
			Record: func(i int, r *mesgdef.Record) {
				if i == 2 {
					r.HeartRate = basetype.Uint8Invalid
				}
			},
			RecordDev: func(i int) []proto.DeveloperField {
				return fitgen.StrydValues(1, fitgen.StrydSample{Power: 250})
			},
		}).MustBytes()
	fits := decodeAll(t, data)
	counts := map[typedef.MesgNum]int{}
	for i := range fits[0].Messages {
		msg := &fits[0].Messages[i]
		counts[msg.Num]++
		if msg.Num != typedef.MesgNumRecord {
			continue
		}
		r := mesgdef.NewRecord(msg)
		n := counts[typedef.MesgNumRecord] - 1
		wantHR := fitgen.RunBaseHR + uint8(n)
		if n == 2 {
			wantHR = basetype.Uint8Invalid
		}
		if r.HeartRate != wantHR || r.PositionLat != fitgen.Degrees(fitgen.RunLat)+int32(n)*10 ||
			r.EnhancedAltitude != fitgen.Altitude(fitgen.RunAltitudeM) || len(msg.DeveloperFields) != 6 {
			t.Errorf("record %d = %+v", n, r)
		}
	}
	want := map[typedef.MesgNum]int{
		typedef.MesgNumDeveloperDataId: 1, typedef.MesgNumFieldDescription: 6, typedef.MesgNumFileId: 1,
		typedef.MesgNumDeviceInfo: 1, typedef.MesgNumEvent: 2, typedef.MesgNumRecord: 3,
		typedef.MesgNumLap: 1, typedef.MesgNumSession: 1, typedef.MesgNumActivity: 1,
	}
	for n, c := range want {
		if counts[n] != c {
			t.Errorf("%v messages = %d, want %d", n, counts[n], c)
		}
	}
	if fitgen.Degrees(90) != 1<<30 || fitgen.Altitude(0) != 2500 || fitgen.Altitude(-500) != 0 {
		t.Errorf("Degrees(90)=%d Altitude(0)=%d", fitgen.Degrees(90), fitgen.Altitude(0))
	}
}
