package fitparser_test

import (
	"strings"
	"testing"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
)

// enumCase describes one enum-valued field: how to write a raw value and
// how to read the decoded name back.
type enumCase struct {
	valid       uint64
	validName   string
	unknown     uint64 // a value the FIT profile does not define
	unknownName string
	invalid     uint64
	stringer    func(v uint64) string // upstream String(), to assert unknown is undefined
	build       func(b *fitgen.Builder, v uint64)
	get         func(f *fitparser.File) string
}

func enumCases() map[string]enumCase {
	return map[string]enumCase{
		"Session.Sport": {
			valid: uint64(typedef.SportRunning), validName: "running",
			unknown: 200, unknownName: "sport_200", invalid: uint64(typedef.SportInvalid),
			stringer: func(v uint64) string { return typedef.Sport(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddSession(func(m *mesgdef.Session) { m.NumLaps = 1; m.Sport = typedef.Sport(v) })
			},
			get: func(f *fitparser.File) string { return f.Sessions[0].Sport },
		},
		"Session.SubSport": {
			valid: uint64(typedef.SubSportTreadmill), validName: "treadmill",
			unknown: 200, unknownName: "sub_sport_200", invalid: uint64(typedef.SubSportInvalid),
			stringer: func(v uint64) string { return typedef.SubSport(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddSession(func(m *mesgdef.Session) { m.NumLaps = 1; m.SubSport = typedef.SubSport(v) })
			},
			get: func(f *fitparser.File) string { return f.Sessions[0].SubSport },
		},
		"Session.SubSport generic": {
			valid: uint64(typedef.SubSportGeneric), validName: "generic",
			unknown: 201, unknownName: "sub_sport_201", invalid: uint64(typedef.SubSportInvalid),
			stringer: func(v uint64) string { return typedef.SubSport(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddSession(func(m *mesgdef.Session) { m.NumLaps = 1; m.SubSport = typedef.SubSport(v) })
			},
			get: func(f *fitparser.File) string { return f.Sessions[0].SubSport },
		},
		"Session.SwimStroke": {
			valid: uint64(typedef.SwimStrokeBreaststroke), validName: "breaststroke",
			unknown: 200, unknownName: "swim_stroke_200", invalid: uint64(typedef.SwimStrokeInvalid),
			stringer: func(v uint64) string { return typedef.SwimStroke(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddSession(func(m *mesgdef.Session) { m.NumLaps = 1; m.SwimStroke = typedef.SwimStroke(v) })
			},
			get: func(f *fitparser.File) string { return f.Sessions[0].SwimStroke },
		},
		"Lap.LapTrigger": {
			valid: uint64(typedef.LapTriggerSessionEnd), validName: "session_end",
			unknown: 42, unknownName: "lap_trigger_42", invalid: uint64(typedef.LapTriggerInvalid),
			stringer: func(v uint64) string { return typedef.LapTrigger(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddLap(func(m *mesgdef.Lap) { m.TotalDistance = 1; m.LapTrigger = typedef.LapTrigger(v) })
			},
			get: func(f *fitparser.File) string { return f.Laps[0].LapTrigger },
		},
		"Lap.Sport": {
			valid: uint64(typedef.SportCycling), validName: "cycling",
			unknown: 250, unknownName: "sport_250", invalid: uint64(typedef.SportInvalid),
			stringer: func(v uint64) string { return typedef.Sport(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddLap(func(m *mesgdef.Lap) { m.TotalDistance = 1; m.Sport = typedef.Sport(v) })
			},
			get: func(f *fitparser.File) string { return f.Laps[0].Sport },
		},
		"Event.Event": {
			valid: uint64(typedef.EventTimer), validName: "timer",
			unknown: 200, unknownName: "event_200", invalid: uint64(typedef.EventInvalid),
			stringer: func(v uint64) string { return typedef.Event(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddEvent(func(m *mesgdef.Event) { m.Timestamp = t0; m.Event = typedef.Event(v) })
			},
			get: func(f *fitparser.File) string { return f.Events[0].Event },
		},
		"Event.EventType": {
			valid: uint64(typedef.EventTypeStopAll), validName: "stop_all",
			unknown: 42, unknownName: "event_type_42", invalid: uint64(typedef.EventTypeInvalid),
			stringer: func(v uint64) string { return typedef.EventType(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddEvent(func(m *mesgdef.Event) { m.Timestamp = t0; m.EventType = typedef.EventType(v) })
			},
			get: func(f *fitparser.File) string { return f.Events[0].EventType },
		},
		"Split.SplitType": {
			valid: uint64(typedef.SplitTypeRunRest), validName: "run_rest",
			unknown: 200, unknownName: "split_type_200", invalid: uint64(typedef.SplitTypeInvalid),
			stringer: func(v uint64) string { return typedef.SplitType(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddSplit(func(m *mesgdef.Split) { m.TotalDistance = 1; m.SplitType = typedef.SplitType(v) })
			},
			get: func(f *fitparser.File) string { return f.Splits[0].SplitType },
		},
		"Workout.Sport": {
			valid: uint64(typedef.SportSwimming), validName: "swimming",
			unknown: 200, unknownName: "sport_200", invalid: uint64(typedef.SportInvalid),
			stringer: func(v uint64) string { return typedef.Sport(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddWorkout(func(m *mesgdef.Workout) { m.NumValidSteps = 1; m.Sport = typedef.Sport(v) })
			},
			get: func(f *fitparser.File) string { return f.Workouts[0].Sport },
		},
		"Workout.SubSport": {
			valid: uint64(typedef.SubSportOpenWater), validName: "open_water",
			unknown: 200, unknownName: "sub_sport_200", invalid: uint64(typedef.SubSportInvalid),
			stringer: func(v uint64) string { return typedef.SubSport(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddWorkout(func(m *mesgdef.Workout) { m.NumValidSteps = 1; m.SubSport = typedef.SubSport(v) })
			},
			get: func(f *fitparser.File) string { return f.Workouts[0].SubSport },
		},
		"WorkoutStep.Intensity": {
			valid: uint64(typedef.IntensityCooldown), validName: "cooldown",
			unknown: 200, unknownName: "intensity_200", invalid: uint64(typedef.IntensityInvalid),
			stringer: func(v uint64) string { return typedef.Intensity(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddWorkoutStep(func(m *mesgdef.WorkoutStep) { m.DurationValue = 1; m.Intensity = typedef.Intensity(v) })
			},
			get: func(f *fitparser.File) string { return f.WorkoutSteps[0].Intensity },
		},
		"WorkoutStep.DurationType": {
			valid: uint64(typedef.WktStepDurationDistance), validName: "distance",
			unknown: 200, unknownName: "wkt_step_duration_200", invalid: uint64(typedef.WktStepDurationInvalid),
			stringer: func(v uint64) string { return typedef.WktStepDuration(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddWorkoutStep(func(m *mesgdef.WorkoutStep) { m.WktStepName = "x"; m.DurationType = typedef.WktStepDuration(v) })
			},
			get: func(f *fitparser.File) string { return f.WorkoutSteps[0].DurationType },
		},
		"WorkoutStep.TargetType": {
			valid: uint64(typedef.WktStepTargetPower), validName: "power",
			unknown: 200, unknownName: "wkt_step_target_200", invalid: uint64(typedef.WktStepTargetInvalid),
			stringer: func(v uint64) string { return typedef.WktStepTarget(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddWorkoutStep(func(m *mesgdef.WorkoutStep) { m.WktStepName = "x"; m.TargetType = typedef.WktStepTarget(v) })
			},
			get: func(f *fitparser.File) string { return f.WorkoutSteps[0].TargetType },
		},
		"ZonesTarget.HrCalcType": {
			valid: uint64(typedef.HrZoneCalcPercentMaxHr), validName: "percent_max_hr",
			unknown: 200, unknownName: "hr_zone_calc_200", invalid: uint64(typedef.HrZoneCalcInvalid),
			stringer: func(v uint64) string { return typedef.HrZoneCalc(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddZonesTarget(func(m *mesgdef.ZonesTarget) { m.MaxHeartRate = 1; m.HrCalcType = typedef.HrZoneCalc(v) })
			},
			get: func(f *fitparser.File) string { return f.ZonesTarget.HrCalcType },
		},
		"ZonesTarget.PwrCalcType": {
			valid: uint64(typedef.PwrZoneCalcCustom), validName: "custom",
			unknown: 200, unknownName: "pwr_zone_calc_200", invalid: uint64(typedef.PwrZoneCalcInvalid),
			stringer: func(v uint64) string { return typedef.PwrZoneCalc(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddZonesTarget(func(m *mesgdef.ZonesTarget) { m.MaxHeartRate = 1; m.PwrCalcType = typedef.PwrZoneCalc(v) })
			},
			get: func(f *fitparser.File) string { return f.ZonesTarget.PwrCalcType },
		},
		"FileID.Type": {
			valid: uint64(typedef.FileWorkout), validName: "workout",
			unknown: 200, unknownName: "file_200", invalid: uint64(typedef.FileInvalid),
			stringer: func(v uint64) string { return typedef.File(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddFileID(func(m *mesgdef.FileId) { m.TimeCreated = t0; m.Type = typedef.File(v) })
			},
			get: func(f *fitparser.File) string { return f.FileID.Type },
		},
		"FileID.Manufacturer": {
			valid: uint64(typedef.ManufacturerWahooFitness), validName: "wahoo_fitness",
			unknown: 5000, unknownName: "manufacturer_5000", invalid: uint64(typedef.ManufacturerInvalid),
			stringer: func(v uint64) string { return typedef.Manufacturer(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddFileID(func(m *mesgdef.FileId) { m.TimeCreated = t0; m.Manufacturer = typedef.Manufacturer(v) })
			},
			get: func(f *fitparser.File) string { return f.FileID.Manufacturer },
		},
		"DeviceInfo.Manufacturer": {
			valid: uint64(typedef.ManufacturerStryd), validName: "stryd",
			unknown: 9999, unknownName: "manufacturer_9999", invalid: uint64(typedef.ManufacturerInvalid),
			stringer: func(v uint64) string { return typedef.Manufacturer(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddDeviceInfo(func(m *mesgdef.DeviceInfo) { m.DeviceIndex = 1; m.Manufacturer = typedef.Manufacturer(v) })
			},
			get: func(f *fitparser.File) string { return f.DeviceInfos[0].Manufacturer },
		},
		"TimeInZone.ReferenceMesg": {
			valid: uint64(typedef.MesgNumLap), validName: "lap",
			unknown: 5000, unknownName: "mesg_num_5000", invalid: uint64(typedef.MesgNumInvalid),
			stringer: func(v uint64) string { return typedef.MesgNum(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddTimeInZone(func(m *mesgdef.TimeInZone) { m.MaxHeartRate = 1; m.ReferenceMesg = typedef.MesgNum(v) })
			},
			get: func(f *fitparser.File) string { return f.TimeInZones[0].ReferenceMesg },
		},
		"FieldDescription.NativeMesgNum": {
			valid: uint64(typedef.MesgNumSession), validName: "session",
			unknown: 5000, unknownName: "mesg_num_5000", invalid: uint64(typedef.MesgNumInvalid),
			stringer: func(v uint64) string { return typedef.MesgNum(v).String() },
			build: func(b *fitgen.Builder, v uint64) {
				b.AddDeveloperDataID(0, nil).AddFieldDescription(0, 0, "Field", "", basetype.Uint8, func(m *mesgdef.FieldDescription) {
					m.NativeMesgNum = typedef.MesgNum(v)
				})
			},
			get: func(f *fitparser.File) string { return f.DeveloperFields[0].NativeMesgNum },
		},
	}
}

func TestEnumNames(t *testing.T) {
	for name, c := range enumCases() {
		t.Run(name, func(t *testing.T) {
			if s := c.stringer(c.unknown); !strings.Contains(s, "Invalid(") {
				t.Fatalf("test value %d is defined upstream as %q; pick an undefined one", c.unknown, s)
			}
			for _, tc := range []struct {
				label string
				raw   uint64
				want  string
			}{
				{"valid", c.valid, c.validName},
				{"unknown", c.unknown, c.unknownName},
				{"invalid", c.invalid, ""},
			} {
				b := fitgen.New()
				c.build(b, tc.raw)
				if got := c.get(decode(t, b)); got != tc.want {
					t.Errorf("%s (%d) = %q, want %q", tc.label, tc.raw, got, tc.want)
				}
			}
		})
	}
}

func TestProductName(t *testing.T) {
	const unknownGarmin = 5000
	if s := typedef.GarminProduct(unknownGarmin).String(); !strings.Contains(s, "Invalid(") {
		t.Fatalf("garmin product %d is defined upstream as %q", unknownGarmin, s)
	}
	if s := typedef.FaveroProduct(unknownGarmin).String(); !strings.Contains(s, "Invalid(") {
		t.Fatalf("favero product %d is defined upstream as %q", unknownGarmin, s)
	}
	cases := []struct {
		name         string
		manufacturer typedef.Manufacturer
		product      uint16
		productName  string
		want         string
	}{
		{"garmin enum", typedef.ManufacturerGarmin, uint16(typedef.GarminProductFr965), "", "fr965"},
		{"string wins over enum", typedef.ManufacturerGarmin, uint16(typedef.GarminProductFr965), "My Watch", "My Watch"},
		{"string for non-garmin", typedef.ManufacturerWahooFitness, 12, "ELEMNT", "ELEMNT"},
		{"garmin unknown product", typedef.ManufacturerGarmin, unknownGarmin, "", "garmin_product_5000"},
		{"garmin invalid product", typedef.ManufacturerGarmin, basetype.Uint16Invalid, "", ""},
		{"dynastream uses garmin names", typedef.ManufacturerDynastream, uint16(typedef.GarminProductFr965), "", "fr965"},
		{"dynastream oem uses garmin names", typedef.ManufacturerDynastreamOem, uint16(typedef.GarminProductFr965), "", "fr965"},
		{"tacx uses garmin names", typedef.ManufacturerTacx, uint16(typedef.GarminProductFr965), "", "fr965"},
		{"favero enum", typedef.ManufacturerFaveroElectronics, uint16(typedef.FaveroProductAssiomaUno), "", typedef.FaveroProductAssiomaUno.String()},
		{"favero unknown product", typedef.ManufacturerFaveroElectronics, unknownGarmin, "", "favero_product_5000"},
		{"other manufacturer", typedef.ManufacturerStryd, 1, "", ""},
		{"no manufacturer", typedef.ManufacturerInvalid, uint16(typedef.GarminProductFr965), "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := decode(t, fitgen.New().
				AddFileID(func(m *mesgdef.FileId) {
					m.Type = typedef.FileActivity
					m.Manufacturer = c.manufacturer
					m.Product = c.product
					m.ProductName = c.productName
				}).
				AddDeviceInfo(func(m *mesgdef.DeviceInfo) {
					m.DeviceIndex = 0
					m.Manufacturer = c.manufacturer
					m.Product = c.product
					m.ProductName = c.productName
				}))
			if got := f.FileID.ProductName; got != c.want {
				t.Errorf("FileID.ProductName = %q, want %q", got, c.want)
			}
			if got := one(t, f.DeviceInfos).ProductName; got != c.want {
				t.Errorf("DeviceInfo.ProductName = %q, want %q", got, c.want)
			}
		})
	}
	if typedef.FaveroProductAssiomaUno.String() != "assioma_uno" {
		t.Errorf("favero name = %q", typedef.FaveroProductAssiomaUno.String())
	}
}
