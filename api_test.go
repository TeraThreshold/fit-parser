package fitparser_test

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
)

// TestPublicAPINoUpstreamTypes walks every exported type reachable from
// File and fails if any of them comes from the upstream module.
func TestPublicAPINoUpstreamTypes(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type, string)
	walk = func(typ reflect.Type, path string) {
		if seen[typ] {
			return
		}
		seen[typ] = true
		if strings.Contains(typ.PkgPath(), "muktihari") {
			t.Errorf("%s has upstream type %s", path, typ)
		}
		switch typ.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			walk(typ.Elem(), path+"[]")
		case reflect.Map:
			walk(typ.Key(), path+".key")
			walk(typ.Elem(), path+".value")
		case reflect.Struct:
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				if f.IsExported() {
					walk(f.Type, path+"."+f.Name)
				}
			}
		}
		for i := 0; i < typ.NumMethod(); i++ {
			m := typ.Method(i)
			for j := 0; j < m.Type.NumIn(); j++ {
				walk(m.Type.In(j), path+"."+m.Name+"()")
			}
			for j := 0; j < m.Type.NumOut(); j++ {
				walk(m.Type.Out(j), path+"."+m.Name+"()")
			}
		}
	}
	walk(reflect.TypeOf(&fitparser.File{}), "File")
}

// TestDeveloperValuesArePlainGoTypes checks that decoded developer values
// hold only basic Go types, not upstream ones.
func TestDeveloperValuesArePlainGoTypes(t *testing.T) {
	data := seedEverything(t)
	g, err := fitparser.Decode(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range g.Records {
		for _, v := range r.DeveloperFields {
			if p := reflect.TypeOf(v.Value).PkgPath(); p != "" {
				t.Errorf("value %v has named type from %q", v.Value, p)
			}
		}
	}
}

func TestJSON(t *testing.T) {
	f := decode(t, fitgen.New().
		AddStrydDescriptions(0).
		AddRecord(func(m *mesgdef.Record) { m.Timestamp = t0; m.HeartRate = 150 },
			fitgen.StrydValues(0, fitgen.StrydSample{Power: 250, VerticalOscillation: 8.25})...).
		AddRecord(func(m *mesgdef.Record) { m.HeartRate = 151 }).
		AddDeviceInfo(func(m *mesgdef.DeviceInfo) { m.DeviceIndex = 0 }).
		AddSession(func(m *mesgdef.Session) { m.TotalDistance = 100 }).
		AddActivity(activity(t0, t0.Add(90*time.Minute))))
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	recs := m["records"].([]any)
	r0 := recs[0].(map[string]any)
	r1 := recs[1].(map[string]any)
	if r0["timestamp"] != "2024-05-04T07:30:00Z" {
		t.Errorf("timestamp = %v", r0["timestamp"])
	}
	if _, ok := r1["timestamp"]; ok {
		t.Error("zero timestamp not omitted")
	}
	// Positions and temperature keep the FIT invalid value in JSON too.
	if r1["position_lat"] != float64(math.MaxInt32) || r1["position_long"] != float64(math.MaxInt32) || r1["temperature"] != float64(127) {
		t.Errorf("invalid position/temperature = %v %v %v", r1["position_lat"], r1["position_long"], r1["temperature"])
	}
	for _, k := range []string{"power", "cadence", "stryd", "developer_fields", "enhanced_speed"} {
		if _, ok := r1[k]; ok {
			t.Errorf("absent %q not omitted", k)
		}
	}
	stryd := r0["stryd"].(map[string]any)
	if stryd["power"] != float64(250) || stryd["vertical_oscillation"] != 8.25 {
		t.Errorf("stryd = %v", stryd)
	}
	dev := r0["developer_fields"].([]any)
	// StrydValues writes all six fields; zeros are valid developer values.
	if len(dev) != 6 || dev[0].(map[string]any)["name"] != "Power" || dev[0].(map[string]any)["value"] != float64(250) {
		t.Errorf("developer_fields = %v", dev)
	}
	if m["utc_offset"] != float64(90*time.Minute) || m["has_utc_offset"] != true || m["has_stryd"] != true {
		t.Errorf("file flags = %v %v %v", m["utc_offset"], m["has_utc_offset"], m["has_stryd"])
	}
	di := m["device_infos"].([]any)[0].(map[string]any)
	if di["device_index"] != float64(0) {
		t.Errorf("device_index = %v, want 0 kept", di["device_index"])
	}
	s := m["sessions"].([]any)[0].(map[string]any)
	if len(s) != 1 || s["total_distance"] != float64(100) {
		t.Errorf("session JSON = %v, want only total_distance", s)
	}

	var back fitparser.File
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal into File: %v", err)
	}
	if back.Records[1].PositionLat != math.MaxInt32 || back.UTCOffset != 90*time.Minute || !back.Records[0].Timestamp.Equal(t0) {
		t.Errorf("round trip = %+v", back)
	}
}

func TestJSONEveryMessage(t *testing.T) {
	for name, data := range map[string][]byte{
		"everything":  seedEverything(t),
		"all invalid": seedAllInvalid(t),
		"chained":     seedChained(t),
	} {
		f, err := fitparser.Decode(strings.NewReader(string(data)))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := json.Marshal(f); err != nil {
			t.Errorf("%s: Marshal: %v", name, err)
		}
	}
}

// TestJSONUint8ArraysAreNumbers checks that uint8 arrays (TimeInZone
// HrZoneHighBoundary and uint8/byte developer arrays) are written as JSON
// arrays of numbers, not as base64 strings, and still round-trip.
func TestJSONUint8ArraysAreNumbers(t *testing.T) {
	f := decode(t, fitgen.New().
		AddDeveloperDataID(0, nil).
		AddFieldDescription(0, 0, "U8s", "", basetype.Uint8, func(m *mesgdef.FieldDescription) {
			m.NativeMesgNum = typedef.MesgNumRecord
			m.Array = 3
		}).
		AddFieldDescription(0, 1, "Bytes", "", basetype.Byte, func(m *mesgdef.FieldDescription) {
			m.NativeMesgNum = typedef.MesgNumRecord
			m.Array = 2
		}).
		AddRecord(func(m *mesgdef.Record) { m.Timestamp = t0 },
			fitgen.Dev(0, 0, []uint8{1, 2, 3}),
			fitgen.Dev(0, 1, []uint8{7, 8}),
		).
		AddTimeInZone(func(m *mesgdef.TimeInZone) {
			m.ReferenceMesg = typedef.MesgNumSession
			m.ReferenceIndex = 0
			m.HrZoneHighBoundary = []uint8{120, 140, 160}
			m.PowerZoneHighBoundary = []uint16{200, 250}
		}))

	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m struct {
		Records []struct {
			DeveloperFields []struct {
				Value json.RawMessage `json:"value"`
			} `json:"developer_fields"`
		} `json:"records"`
		TimeInZones []map[string]json.RawMessage `json:"time_in_zones"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	tz := one(t, m.TimeInZones)
	if got := string(tz["hr_zone_high_boundary"]); got != "[120,140,160]" {
		t.Errorf("hr_zone_high_boundary = %s, want [120,140,160]", got)
	}
	if got := string(tz["power_zone_high_boundary"]); got != "[200,250]" {
		t.Errorf("power_zone_high_boundary = %s, want [200,250]", got)
	}
	if got := string(tz["reference_mesg"]); got != `"session"` {
		t.Errorf("reference_mesg = %s", got)
	}
	if got := string(tz["reference_index"]); got != "0" {
		t.Errorf("reference_index = %s, want 0 kept", got)
	}
	dev := one(t, m.Records).DeveloperFields
	if len(dev) != 2 || string(dev[0].Value) != "[1,2,3]" || string(dev[1].Value) != "[7,8]" {
		t.Errorf("developer values = %s", data)
	}

	var back fitparser.File
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal into File: %v", err)
	}
	if got := one(t, back.TimeInZones); !reflect.DeepEqual(got, f.TimeInZones[0]) {
		t.Errorf("round trip TimeInZone = %+v, want %+v", got, f.TimeInZones[0])
	}

	// An absent boundary array stays omitted.
	data, err = json.Marshal(fitparser.TimeInZone{ReferenceIndex: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != `{"reference_index":1}` {
		t.Errorf("empty TimeInZone JSON = %s", got)
	}
}
