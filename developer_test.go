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
	"github.com/muktihari/fit/proto"
)

func nativeRecord(m *mesgdef.FieldDescription) { m.NativeMesgNum = typedef.MesgNumRecord }

func recordAt(i int) func(*mesgdef.Record) {
	return func(m *mesgdef.Record) { m.Timestamp = t0; m.HeartRate = uint8(100 + i) }
}

func TestDeveloperFieldDescriptions(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Doughnuts Earned", "doughnuts", basetype.Uint8, nativeRecord).
		AddFieldDescription(0, 1, "Core Temp", "", basetype.Float32, nil).
		AddDeveloperDataID(1, nil).
		AddFieldDescription(1, 0, "Muscle O2", "%", basetype.Uint16, func(m *mesgdef.FieldDescription) {
			m.NativeMesgNum = typedef.MesgNumSession
		}).
		AddFieldDescription(1, 5, "", "", basetype.Sint32, func(m *mesgdef.FieldDescription) {
			m.FieldName = []string{"", "Second Name"} // first non-empty element is used
			m.Units = []string{"", "u2"}
		}))
	want := []fitparser.DeveloperFieldDescription{
		{DeveloperDataIndex: 0, FieldDefinitionNumber: 0, FieldName: "Doughnuts Earned", Units: "doughnuts", NativeMesgNum: "record"},
		{DeveloperDataIndex: 0, FieldDefinitionNumber: 1, FieldName: "Core Temp", Units: "", NativeMesgNum: ""},
		{DeveloperDataIndex: 1, FieldDefinitionNumber: 0, FieldName: "Muscle O2", Units: "%", NativeMesgNum: "session"},
		{DeveloperDataIndex: 1, FieldDefinitionNumber: 5, FieldName: "Second Name", Units: "u2", NativeMesgNum: ""},
	}
	if !reflect.DeepEqual(f.DeveloperFields, want) {
		t.Errorf("DeveloperFields =\n%+v\nwant\n%+v", f.DeveloperFields, want)
	}
	if f.HasStryd {
		t.Error("HasStryd = true without Stryd fields")
	}
}

func TestDeveloperFieldValuesResolvedByIndexAndNum(t *testing.T) {
	// Two apps both declare field 0 with different names and types.
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Doughnuts Earned", "doughnuts", basetype.Uint8, nativeRecord).
		AddFieldDescription(0, 1, "Label", "", basetype.String, nativeRecord).
		AddDeveloperDataID(1, nil).
		AddFieldDescription(1, 0, "Muscle O2", "%", basetype.Uint16, nativeRecord).
		AddFieldDescription(1, 2, "Samples", "ms", basetype.Uint16, func(m *mesgdef.FieldDescription) {
			m.NativeMesgNum = typedef.MesgNumRecord
			m.Array = 3
		}).
		AddFieldDescription(1, 3, "Ratio", "", basetype.Float64, nativeRecord).
		AddFieldDescription(1, 4, "Offset", "", basetype.Sint8, nativeRecord).
		AddRecord(recordAt(0),
			fitgen.Dev(0, 0, uint8(3)),
			fitgen.Dev(1, 0, uint16(612)),
			fitgen.Dev(0, 1, "tempo"),
			fitgen.Dev(1, 2, []uint16{10, 20, 30}),
			fitgen.Dev(1, 3, float64(0.125)),
			fitgen.Dev(1, 4, int8(-7)),
		))
	r := one(t, f.Records)
	want := []fitparser.DeveloperFieldValue{
		{DeveloperDataIndex: 0, Num: 0, Name: "Doughnuts Earned", Units: "doughnuts", Value: uint8(3)},
		{DeveloperDataIndex: 1, Num: 0, Name: "Muscle O2", Units: "%", Value: uint16(612)},
		{DeveloperDataIndex: 0, Num: 1, Name: "Label", Units: "", Value: "tempo"},
		{DeveloperDataIndex: 1, Num: 2, Name: "Samples", Units: "ms", Value: []uint16{10, 20, 30}},
		{DeveloperDataIndex: 1, Num: 3, Name: "Ratio", Units: "", Value: float64(0.125)},
		{DeveloperDataIndex: 1, Num: 4, Name: "Offset", Units: "", Value: int8(-7)},
	}
	if !reflect.DeepEqual(r.DeveloperFields, want) {
		t.Errorf("DeveloperFields =\n%#v\nwant\n%#v", r.DeveloperFields, want)
	}
	if r.Stryd != nil {
		t.Errorf("Stryd = %+v, want nil", r.Stryd)
	}
}

func TestDeveloperFieldScaleNotApplied(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Scaled", "x", basetype.Uint16, func(m *mesgdef.FieldDescription) {
			m.NativeMesgNum = typedef.MesgNumRecord
			m.Scale = 10
			m.Offset = 5
		}).
		AddRecord(recordAt(0), fitgen.Dev(0, 0, uint16(1234))))
	r := one(t, f.Records)
	if len(r.DeveloperFields) != 1 || r.DeveloperFields[0].Value != uint16(1234) {
		t.Errorf("DeveloperFields = %+v, want raw uint16(1234)", r.DeveloperFields)
	}
}

func TestDeveloperFieldInvalidValuesDropped(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "U16", "", basetype.Uint16, nativeRecord).
		AddFieldDescription(0, 1, "F32", "", basetype.Float32, nativeRecord).
		AddFieldDescription(0, 2, "F64", "", basetype.Float64, nativeRecord).
		AddFieldDescription(0, 3, "F32s", "", basetype.Float32, func(m *mesgdef.FieldDescription) { m.Array = 2 }).
		AddFieldDescription(0, 4, "S8", "", basetype.Sint8, nativeRecord).
		AddFieldDescription(0, 5, "Keep", "", basetype.Uint8, nativeRecord).
		AddFieldDescription(0, 6, "F32inv", "", basetype.Float32, nativeRecord).
		AddRecord(recordAt(0),
			fitgen.Dev(0, 0, uint16(basetype.Uint16Invalid)),
			fitgen.Dev(0, 1, float32(math.NaN())),
			fitgen.Dev(0, 2, math.Inf(1)),
			fitgen.Dev(0, 3, []float32{1.5, float32(math.Inf(-1))}),
			fitgen.Dev(0, 4, int8(basetype.Sint8Invalid)),
			fitgen.Dev(0, 5, uint8(9)),
			fitgen.Dev(0, 6, math.Float32frombits(basetype.Float32Invalid)),
		))
	r := one(t, f.Records)
	want := []fitparser.DeveloperFieldValue{{DeveloperDataIndex: 0, Num: 5, Name: "Keep", Value: uint8(9)}}
	if !reflect.DeepEqual(r.DeveloperFields, want) {
		t.Errorf("DeveloperFields = %#v, want only the valid value %#v", r.DeveloperFields, want)
	}
}

func strydSample() fitgen.StrydSample {
	return fitgen.StrydSample{Power: 260, GroundTime: 240, VerticalOscillation: 8.5, FormPower: 70, LegSpringStiffness: 10.25, AirPower: 3}
}

var wantStryd = fitparser.Stryd{Power: 260, GroundTime: 240, VerticalOscillation: 8.5, FormPower: 70, LegSpringStiffness: 10.25, AirPower: 3}

func TestStrydDetectionAndExtraction(t *testing.T) {
	for _, idx := range []uint8{0, 3} {
		f := decode(t, fitgen.New().
			AddStrydDescriptions(idx).
			AddRecord(recordAt(0), fitgen.StrydValues(idx, strydSample())...).
			AddRecord(recordAt(1)))
		if !f.HasStryd {
			t.Fatalf("idx %d: HasStryd = false", idx)
		}
		if len(f.DeveloperFields) != 6 {
			t.Errorf("idx %d: %d descriptions, want 6", idx, len(f.DeveloperFields))
		}
		if len(f.Records) != 2 {
			t.Fatalf("records = %d", len(f.Records))
		}
		if f.Records[0].Stryd == nil || *f.Records[0].Stryd != wantStryd {
			t.Errorf("idx %d: Stryd = %+v, want %+v", idx, f.Records[0].Stryd, wantStryd)
		}
		if f.Records[1].Stryd != nil {
			t.Errorf("idx %d: record without Stryd values: Stryd = %+v, want nil", idx, f.Records[1].Stryd)
		}
		names := []string{"Power", "Ground Time", "Vertical Oscillation", "Form Power", "Leg Spring Stiffness", "Air Power"}
		for i, v := range f.Records[0].DeveloperFields {
			if v.DeveloperDataIndex != idx || v.Name != names[i] {
				t.Errorf("idx %d: dev value %d = %+v, want name %q", idx, i, v, names[i])
			}
		}
	}
}

func TestStrydDetectedByEitherSignatureField(t *testing.T) {
	for _, sig := range []string{"Leg Spring Stiffness", "Form Power"} {
		f := decode(t, fitgen.New().
			AddDeveloperDataID(2, nil).
			AddFieldDescription(2, 0, "Power", "Watts", basetype.Uint16, nativeRecord).
			AddFieldDescription(2, 1, sig, "", basetype.Uint16, nativeRecord).
			AddRecord(recordAt(0), fitgen.Dev(2, 0, uint16(301))))
		if !f.HasStryd {
			t.Errorf("%s: HasStryd = false", sig)
		}
		r := one(t, f.Records)
		if r.Stryd == nil || *r.Stryd != (fitparser.Stryd{Power: 301}) {
			t.Errorf("%s: Stryd = %+v, want Power 301", sig, r.Stryd)
		}
	}
}

func TestStrydNotDetectedForPlainPowerApp(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Power", "Watts", basetype.Uint16, nativeRecord).
		AddFieldDescription(0, 3, "Ground Time", "ms", basetype.Uint16, nativeRecord).
		AddRecord(recordAt(0), fitgen.Dev(0, 0, uint16(300)), fitgen.Dev(0, 3, uint16(250))))
	if f.HasStryd {
		t.Error("HasStryd = true for an app without Stryd signature fields")
	}
	r := one(t, f.Records)
	if r.Stryd != nil {
		t.Errorf("Stryd = %+v, want nil", r.Stryd)
	}
	if len(r.DeveloperFields) != 2 {
		t.Errorf("DeveloperFields = %+v, want 2 values", r.DeveloperFields)
	}
}

func TestNonStrydFileHasNoStryd(t *testing.T) {
	f := decode(t, fitgen.New().AddRun(fitgen.Run{Start: t0, Seconds: 5}))
	if f.HasStryd {
		t.Error("HasStryd = true")
	}
	if f.DeveloperFields != nil {
		t.Errorf("DeveloperFields = %+v, want nil", f.DeveloperFields)
	}
	for i := range f.Records {
		if f.Records[i].Stryd != nil || f.Records[i].DeveloperFields != nil {
			t.Errorf("record %d: Stryd=%+v dev=%+v, want nil", i, f.Records[i].Stryd, f.Records[i].DeveloperFields)
		}
	}
}

// TestStrydCollisionWithSecondApp puts a second Connect IQ app on another
// developer data index that also declares field 0 (and even names it
// "Power"). Its values must never be read as Stryd metrics.
func TestStrydCollisionWithSecondApp(t *testing.T) {
	for _, tc := range []struct {
		name               string
		strydIdx, otherIdx uint8
		otherName          string
		otherFirst         bool
	}{
		{"other app after stryd, other name", 0, 1, "Heart Rate Variability", false},
		{"other app after stryd, same name", 0, 1, "Power", false},
		{"other app before stryd, same name", 1, 0, "Power", true},
		{"other app on high index", 0, 7, "Power", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := fitgen.New()
			addOther := func() {
				b.AddDeveloperDataID(tc.otherIdx, nil).
					AddFieldDescription(tc.otherIdx, 0, tc.otherName, "units", basetype.Uint16, nativeRecord).
					AddFieldDescription(tc.otherIdx, 4, "Vertical Oscillation", "cm", basetype.Float32, nativeRecord)
			}
			if tc.otherFirst {
				addOther()
				b.AddStrydDescriptions(tc.strydIdx)
			} else {
				b.AddStrydDescriptions(tc.strydIdx)
				addOther()
			}
			other := []proto.DeveloperField{
				fitgen.Dev(tc.otherIdx, 0, uint16(999)),
				fitgen.Dev(tc.otherIdx, 4, float32(42.5)),
			}
			strydVals := fitgen.StrydValues(tc.strydIdx, strydSample())
			var both []proto.DeveloperField
			if tc.otherFirst {
				both = append(append(both, strydVals...), other...) // other app last: overwrite-by-num would win
			} else {
				both = append(append(both, other...), strydVals...)
				both = append(both, other...) // and again after Stryd
			}
			b.AddRecord(recordAt(0), both...)
			// A record with only the other app's values must have no Stryd.
			b.AddRecord(recordAt(1), other...)
			f := decode(t, b)

			if !f.HasStryd {
				t.Fatal("HasStryd = false")
			}
			if len(f.Records) != 2 {
				t.Fatalf("records = %d", len(f.Records))
			}
			if got := f.Records[0].Stryd; got == nil || *got != wantStryd {
				t.Errorf("Stryd = %+v, want %+v", got, wantStryd)
			}
			if got := f.Records[1].Stryd; got != nil {
				t.Errorf("record with only the other app's values: Stryd = %+v, want nil", got)
			}
			var sawOther bool
			for _, v := range f.Records[1].DeveloperFields {
				if v.DeveloperDataIndex == tc.otherIdx && v.Num == 0 {
					sawOther = true
					if v.Name != tc.otherName || v.Value != uint16(999) {
						t.Errorf("other app value = %+v", v)
					}
				}
			}
			if !sawOther {
				t.Errorf("other app value missing: %+v", f.Records[1].DeveloperFields)
			}
		})
	}
}

func TestStrydValueKinds(t *testing.T) {
	b := fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Power", "W", basetype.Float32, nativeRecord).
		AddFieldDescription(0, 3, "Ground Time", "ms", basetype.Uint32, nativeRecord).
		AddFieldDescription(0, 4, "Vertical Oscillation", "cm", basetype.Uint8, nativeRecord).
		AddFieldDescription(0, 8, "Form Power", "W", basetype.Sint16, nativeRecord).
		AddFieldDescription(0, 9, "Leg Spring Stiffness", "kN/m", basetype.Float64, nativeRecord).
		AddFieldDescription(0, 11, "Air Power", "W", basetype.Uint8, nativeRecord).
		AddRecord(recordAt(0),
			fitgen.Dev(0, 0, float32(260.6)),
			fitgen.Dev(0, 3, uint32(240)),
			fitgen.Dev(0, 4, uint8(9)),
			fitgen.Dev(0, 8, int16(71)),
			fitgen.Dev(0, 9, float64(11.5)),
			fitgen.Dev(0, 11, uint8(4)),
		)
	r := one(t, decode(t, b).Records)
	want := fitparser.Stryd{Power: 261, GroundTime: 240, VerticalOscillation: 9, FormPower: 71, LegSpringStiffness: 11.5, AirPower: 4}
	if r.Stryd == nil || *r.Stryd != want {
		t.Errorf("Stryd = %+v, want %+v", r.Stryd, want)
	}
}

func TestStrydRejectsNonPositiveAndOutOfRange(t *testing.T) {
	b := fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Power", "W", basetype.Uint32, nativeRecord).
		AddFieldDescription(0, 3, "Ground Time", "ms", basetype.Sint16, nativeRecord).
		AddFieldDescription(0, 4, "Vertical Oscillation", "cm", basetype.Float32, nativeRecord).
		AddFieldDescription(0, 8, "Form Power", "W", basetype.Uint16, nativeRecord).
		AddFieldDescription(0, 9, "Leg Spring Stiffness", "kN/m", basetype.Float32, nativeRecord).
		AddFieldDescription(0, 11, "Air Power", "W", basetype.Float32, nativeRecord)
	// Record 0: every value invalid, non-positive or too large.
	b.AddRecord(recordAt(0),
		fitgen.Dev(0, 0, uint32(70000)),
		fitgen.Dev(0, 3, int16(-5)),
		fitgen.Dev(0, 4, float32(0)),
		fitgen.Dev(0, 8, uint16(basetype.Uint16Invalid)),
		fitgen.Dev(0, 9, float32(-1)),
		fitgen.Dev(0, 11, float32(math.NaN())),
	)
	// Record 1: boundary values that are accepted, next to a rejected one.
	b.AddRecord(recordAt(1),
		fitgen.Dev(0, 0, uint32(65534)),
		fitgen.Dev(0, 3, int16(1)),
		fitgen.Dev(0, 11, float32(0.4)), // rounds to 0: rejected
	)
	// Record 2: 65535 is the uint16 sentinel even when the base type is wider.
	b.AddRecord(recordAt(2), fitgen.Dev(0, 0, uint32(65535)))
	f := decode(t, b)
	if len(f.Records) != 3 {
		t.Fatalf("records = %d", len(f.Records))
	}
	if f.Records[0].Stryd != nil {
		t.Errorf("record 0: Stryd = %+v, want nil", f.Records[0].Stryd)
	}
	want := fitparser.Stryd{Power: 65534, GroundTime: 1}
	if f.Records[1].Stryd == nil || *f.Records[1].Stryd != want {
		t.Errorf("record 1: Stryd = %+v, want %+v", f.Records[1].Stryd, want)
	}
	if f.Records[2].Stryd != nil {
		t.Errorf("record 2: Stryd = %+v, want nil", f.Records[2].Stryd)
	}
}

func TestStrydPowerAbove2000Kept(t *testing.T) {
	s := strydSample()
	s.Power = 2400
	f := decode(t, fitgen.New().AddStrydDescriptions(0).AddRecord(recordAt(0), fitgen.StrydValues(0, s)...))
	if r := one(t, f.Records); r.Stryd == nil || r.Stryd.Power != 2400 {
		t.Errorf("Stryd = %+v, want Power 2400", r.Stryd)
	}
}

// TestDeveloperIndexReusedAcrossSequences: in a chained file, index 0 is
// Stryd in the first sequence and a different app in the second.
func TestDeveloperIndexReusedAcrossSequences(t *testing.T) {
	f := decode(t, fitgen.New().
		AddStrydDescriptions(0).
		AddRecord(recordAt(0), fitgen.StrydValues(0, strydSample())...).
		NewSequence().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Power", "Watts", basetype.Uint16, nativeRecord).
		AddRecord(recordAt(1), fitgen.Dev(0, 0, uint16(777))))
	if !f.HasStryd {
		t.Error("HasStryd = false, want true (first sequence declares Stryd)")
	}
	if len(f.DeveloperFields) != 7 {
		t.Errorf("DeveloperFields = %d, want 7 (6 + 1)", len(f.DeveloperFields))
	}
	if len(f.Records) != 2 {
		t.Fatalf("records = %d", len(f.Records))
	}
	if got := f.Records[0].Stryd; got == nil || *got != wantStryd {
		t.Errorf("sequence 1: Stryd = %+v", got)
	}
	if got := f.Records[1].Stryd; got != nil {
		t.Errorf("sequence 2: Stryd = %+v, want nil (index 0 is another app there)", got)
	}
	want := []fitparser.DeveloperFieldValue{{DeveloperDataIndex: 0, Num: 0, Name: "Power", Units: "Watts", Value: uint16(777)}}
	if !reflect.DeepEqual(f.Records[1].DeveloperFields, want) {
		t.Errorf("sequence 2 dev values = %+v, want %+v", f.Records[1].DeveloperFields, want)
	}
}

// TestStrydDeclaredOnlyInSecondSequence: the second sequence introduces
// Stryd; the first sequence's records stay without Stryd.
func TestStrydDeclaredOnlyInSecondSequence(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Power", "Watts", basetype.Uint16, nativeRecord).
		AddRecord(recordAt(0), fitgen.Dev(0, 0, uint16(777))).
		NewSequence().
		AddStrydDescriptions(0).
		AddRecord(recordAt(1), fitgen.StrydValues(0, strydSample())...))
	if !f.HasStryd {
		t.Error("HasStryd = false")
	}
	if len(f.Records) != 2 {
		t.Fatalf("records = %d", len(f.Records))
	}
	if f.Records[0].Stryd != nil {
		t.Errorf("sequence 1: Stryd = %+v, want nil", f.Records[0].Stryd)
	}
	if got := f.Records[1].Stryd; got == nil || *got != wantStryd {
		t.Errorf("sequence 2: Stryd = %+v, want %+v", got, wantStryd)
	}
}

func TestDeveloperValue64BitAndArrayKinds(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Power", "W", basetype.Sint64, nativeRecord).
		AddFieldDescription(0, 3, "Ground Time", "ms", basetype.Uint64, nativeRecord).
		AddFieldDescription(0, 4, "Vertical Oscillation", "cm", basetype.Sint8, nativeRecord).
		AddFieldDescription(0, 8, "Form Power", "W", basetype.Sint32, nativeRecord).
		AddFieldDescription(0, 9, "Leg Spring Stiffness", "kN/m", basetype.Float64, nativeRecord).
		AddFieldDescription(0, 11, "Air Power", "W", basetype.Float64, func(m *mesgdef.FieldDescription) { m.Array = 2 }).
		AddFieldDescription(0, 20, "F64s", "", basetype.Float64, func(m *mesgdef.FieldDescription) { m.Array = 2 }).
		AddFieldDescription(0, 21, "F64sBad", "", basetype.Float64, func(m *mesgdef.FieldDescription) { m.Array = 2 }).
		AddFieldDescription(0, 22, "Names", "", basetype.String, func(m *mesgdef.FieldDescription) { m.Array = 2 }).
		AddRecord(recordAt(0),
			fitgen.Dev(0, 0, int64(333)),
			fitgen.Dev(0, 3, uint64(222)),
			fitgen.Dev(0, 4, int8(7)),
			fitgen.Dev(0, 8, int32(66)),
			fitgen.Dev(0, 9, float64(12.25)),
			fitgen.Dev(0, 11, []float64{1, 2}), // arrays are not Stryd scalars
			fitgen.Dev(0, 20, []float64{0.5, 1.5}),
			fitgen.Dev(0, 21, []float64{0.5, math.NaN()}),
			fitgen.Dev(0, 22, []string{"a", "b"}),
		))
	r := one(t, f.Records)
	want := fitparser.Stryd{Power: 333, GroundTime: 222, VerticalOscillation: 7, FormPower: 66, LegSpringStiffness: 12.25}
	if r.Stryd == nil || *r.Stryd != want {
		t.Errorf("Stryd = %+v, want %+v", r.Stryd, want)
	}
	got := map[uint8]any{}
	for _, v := range r.DeveloperFields {
		got[v.Num] = v.Value
	}
	wantVals := map[uint8]any{
		0: int64(333), 3: uint64(222), 4: int8(7), 8: int32(66), 9: float64(12.25),
		11: []float64{1, 2}, 20: []float64{0.5, 1.5}, 22: []string{"a", "b"},
	}
	if !reflect.DeepEqual(got, wantVals) {
		t.Errorf("values = %#v\nwant %#v", got, wantVals)
	}
}

func TestStrydIgnoresNonNumericValue(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Power", "", basetype.String, nativeRecord).
		AddFieldDescription(0, 8, "Form Power", "W", basetype.Uint16, nativeRecord).
		AddRecord(recordAt(0), fitgen.Dev(0, 0, "300")))
	r := one(t, f.Records)
	if r.Stryd != nil {
		t.Errorf("Stryd = %+v, want nil for a string Power field", r.Stryd)
	}
	want := []fitparser.DeveloperFieldValue{{DeveloperDataIndex: 0, Num: 0, Name: "Power", Value: "300"}}
	if !reflect.DeepEqual(r.DeveloperFields, want) {
		t.Errorf("DeveloperFields = %+v, want %+v", r.DeveloperFields, want)
	}
}

// TestDeveloperDuplicateDescriptionFirstWins: when two field descriptions
// share a (developer data index, field number) key, the first one applies,
// as in the upstream decoder. Both descriptions are still listed.
func TestDeveloperDuplicateDescriptionFirstWins(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Cadence", "rpm", basetype.Uint8, nativeRecord).
		AddFieldDescription(0, 0, "Power", "Watts", basetype.Uint8, nativeRecord).
		AddRecord(recordAt(0), fitgen.Dev(0, 0, uint8(90))))
	if len(f.DeveloperFields) != 2 {
		t.Errorf("DeveloperFields = %+v, want both descriptions", f.DeveloperFields)
	}
	want := []fitparser.DeveloperFieldValue{{DeveloperDataIndex: 0, Num: 0, Name: "Cadence", Units: "rpm", Value: uint8(90)}}
	if r := one(t, f.Records); !reflect.DeepEqual(r.DeveloperFields, want) {
		t.Errorf("DeveloperFields = %#v, want %#v", r.DeveloperFields, want)
	}
}

// TestDeveloperDuplicateDescriptionDifferentBaseType: a later description
// with another base type does not make the value decoded under the first one
// look invalid.
func TestDeveloperDuplicateDescriptionDifferentBaseType(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "Cadence", "rpm", basetype.Uint8, nativeRecord).
		AddRecord(recordAt(0), fitgen.Dev(0, 0, uint8(90))).
		AddFieldDescription(0, 0, "Power", "Watts", basetype.Uint16, nativeRecord))
	want := []fitparser.DeveloperFieldValue{{DeveloperDataIndex: 0, Num: 0, Name: "Cadence", Units: "rpm", Value: uint8(90)}}
	if r := one(t, f.Records); !reflect.DeepEqual(r.DeveloperFields, want) {
		t.Errorf("DeveloperFields = %#v, want %#v", r.DeveloperFields, want)
	}
}

// TestDeveloperArraySentinels: a uint8 array is returned as []uint16; an
// array is dropped only when every element is invalid, and a partly valid
// array keeps its raw sentinel elements.
func TestDeveloperArraySentinels(t *testing.T) {
	arr := func(m *mesgdef.FieldDescription) {
		m.NativeMesgNum = typedef.MesgNumRecord
		m.Array = 2
	}
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "U8s", "", basetype.Uint8, arr).
		AddFieldDescription(0, 1, "U16s", "", basetype.Uint16, arr).
		AddFieldDescription(0, 2, "U16sInvalid", "", basetype.Uint16, arr).
		AddFieldDescription(0, 3, "U8sInvalid", "", basetype.Uint8, arr).
		AddRecord(recordAt(0),
			fitgen.Dev(0, 0, []uint8{basetype.Uint8Invalid, 5}),
			fitgen.Dev(0, 1, []uint16{basetype.Uint16Invalid, 42}),
			fitgen.Dev(0, 2, []uint16{basetype.Uint16Invalid, basetype.Uint16Invalid}),
			fitgen.Dev(0, 3, []uint8{basetype.Uint8Invalid, basetype.Uint8Invalid}),
		))
	want := []fitparser.DeveloperFieldValue{
		{DeveloperDataIndex: 0, Num: 0, Name: "U8s", Value: []uint16{0xFF, 5}},
		{DeveloperDataIndex: 0, Num: 1, Name: "U16s", Value: []uint16{0xFFFF, 42}},
	}
	if r := one(t, f.Records); !reflect.DeepEqual(r.DeveloperFields, want) {
		t.Errorf("DeveloperFields = %#v, want %#v", r.DeveloperFields, want)
	}
}
