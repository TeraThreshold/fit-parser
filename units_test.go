package fitparser_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// nearDeg compares degrees to within one semicircle (about 8.4e-8 degrees),
// the resolution lost when fitgen.Degrees truncates.
func nearDeg(a, b float64) bool { return math.Abs(a-b) < 1e-7 }

func TestRecordAccessors(t *testing.T) {
	r := fitparser.Record{
		EnhancedSpeed:           3456,
		Speed:                   1000,
		Distance:                123_456,
		EnhancedAltitude:        fitgen.Altitude(123.4),
		Grade:                   -325,
		StanceTime:              2455,
		VerticalOscillation:     876,
		StepLength:              11234,
		VerticalRatio:           812,
		StanceTimeBalance:       4987,
		EnhancedRespirationRate: 3125,
		PositionLat:             fitgen.Degrees(45),
		PositionLong:            fitgen.Degrees(-90),
		Temperature:             -12,
	}
	checks := []struct {
		name      string
		got, want float64
	}{
		{"SpeedMps", r.SpeedMps(), 3.456},
		{"DistanceM", r.DistanceM(), 1234.56},
		{"GradePercent", r.GradePercent(), -3.25},
		{"StanceTimeMs", r.StanceTimeMs(), 245.5},
		{"VerticalOscillationMm", r.VerticalOscillationMm(), 87.6},
		{"StepLengthM", r.StepLengthM(), 1.1234},
		{"VerticalRatioPercent", r.VerticalRatioPercent(), 8.12},
		{"StanceTimeBalancePercent", r.StanceTimeBalancePercent(), 49.87},
		{"RespirationRateBrpm", r.RespirationRateBrpm(), 31.25},
	}
	for _, c := range checks {
		if !near(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if alt, ok := r.AltitudeM(); !ok || !near(alt, 123.4) {
		t.Errorf("AltitudeM = %v, %v; want 123.4, true", alt, ok)
	}
	if lat, lon, ok := r.Position(); !ok || !nearDeg(lat, 45) || !nearDeg(lon, -90) {
		t.Errorf("Position = %v, %v, %v; want 45, -90, true", lat, lon, ok)
	}
	if c, ok := r.TemperatureC(); !ok || c != -12 {
		t.Errorf("TemperatureC = %v, %v; want -12, true", c, ok)
	}
}

func TestRecordSpeedFallback(t *testing.T) {
	for _, c := range []struct {
		r    fitparser.Record
		want float64
	}{
		{fitparser.Record{Speed: 2500}, 2.5},
		{fitparser.Record{EnhancedSpeed: 4000, Speed: 2500}, 4},
		{fitparser.Record{}, 0},
	} {
		if got := c.r.SpeedMps(); got != c.want {
			t.Errorf("%+v SpeedMps = %v, want %v", c.r, got, c.want)
		}
	}
}

func TestRecordAccessorEdges(t *testing.T) {
	cases := []struct {
		raw    uint32
		want   float64
		wantOK bool
	}{
		{0, 0, false},       // absent
		{2500, 0, true},     // sea level is valid
		{2000, -100, true},  // below sea level
		{1, -499.8, true},   // lowest representable
		{27500, 5000, true}, // high
	}
	for _, c := range cases {
		r := fitparser.Record{EnhancedAltitude: c.raw}
		if got, ok := r.AltitudeM(); ok != c.wantOK || !near(got, c.want) {
			t.Errorf("AltitudeM(%d) = %v, %v; want %v, %v", c.raw, got, ok, c.want, c.wantOK)
		}
	}

	inv := int32(math.MaxInt32)
	for _, p := range [][2]int32{{inv, inv}, {inv, 0}, {0, inv}} {
		r := fitparser.Record{PositionLat: p[0], PositionLong: p[1]}
		if lat, lon, ok := r.Position(); ok || lat != 0 || lon != 0 {
			t.Errorf("Position(%v) = %v, %v, %v; want 0, 0, false", p, lat, lon, ok)
		}
	}
	r := fitparser.Record{PositionLat: math.MinInt32, PositionLong: math.MinInt32}
	if lat, lon, ok := r.Position(); !ok || lat != -180 || lon != -180 {
		t.Errorf("Position(min) = %v, %v, %v; want -180, -180, true", lat, lon, ok)
	}
	for _, c := range []struct {
		raw  int8
		want float64
		ok   bool
	}{{127, 0, false}, {0, 0, true}, {-128, -128, true}, {126, 126, true}} {
		r := fitparser.Record{Temperature: c.raw}
		if got, ok := r.TemperatureC(); ok != c.ok || got != c.want {
			t.Errorf("TemperatureC(%d) = %v, %v; want %v, %v", c.raw, got, ok, c.want, c.ok)
		}
	}
}

func TestSessionAccessors(t *testing.T) {
	s := fitparser.Session{
		TotalElapsedTime:       3_723_456,
		TotalTimerTime:         3_600_001,
		TotalMovingTime:        3_500_500,
		TotalDistance:          1_000_050,
		EnhancedAvgSpeed:       2778,
		EnhancedMaxSpeed:       6543,
		MinAltitude:            2250, // -50 m
		MaxAltitude:            3500, // 200 m
		AvgStanceTime:          2451,
		AvgVerticalOscillation: 873,
		AvgStepLength:          11234,
		AvgVerticalRatio:       812,
		AvgStanceTimeBalance:   4987,
		PoolLength:             2286,
	}
	if got := s.ElapsedTime(); got != 3_723_456*time.Millisecond {
		t.Errorf("ElapsedTime = %v", got)
	}
	if got := s.TimerTime(); got != 3_600_001*time.Millisecond {
		t.Errorf("TimerTime = %v", got)
	}
	if got := s.MovingTime(); got != 3_500_500*time.Millisecond {
		t.Errorf("MovingTime = %v", got)
	}
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"DistanceM", s.DistanceM(), 10000.5},
		{"AvgSpeedMps", s.AvgSpeedMps(), 2.778},
		{"MaxSpeedMps", s.MaxSpeedMps(), 6.543},
		{"AvgStanceTimeMs", s.AvgStanceTimeMs(), 245.1},
		{"AvgVerticalOscillationMm", s.AvgVerticalOscillationMm(), 87.3},
		{"AvgStepLengthM", s.AvgStepLengthM(), 1.1234},
		{"AvgVerticalRatioPercent", s.AvgVerticalRatioPercent(), 8.12},
		{"AvgStanceTimeBalancePercent", s.AvgStanceTimeBalancePercent(), 49.87},
		{"PoolLengthM", s.PoolLengthM(), 22.86},
	} {
		if !near(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if m, ok := s.MinAltitudeM(); !ok || !near(m, -50) {
		t.Errorf("MinAltitudeM = %v, %v", m, ok)
	}
	if m, ok := s.MaxAltitudeM(); !ok || !near(m, 200) {
		t.Errorf("MaxAltitudeM = %v, %v", m, ok)
	}
	var zero fitparser.Session
	if _, ok := zero.MinAltitudeM(); ok {
		t.Error("MinAltitudeM ok on zero session")
	}
	if _, ok := zero.MaxAltitudeM(); ok {
		t.Error("MaxAltitudeM ok on zero session")
	}
	if zero.ElapsedTime() != 0 || zero.DistanceM() != 0 || zero.AvgSpeedMps() != 0 {
		t.Error("zero session accessors not zero")
	}
}

func TestLapAccessors(t *testing.T) {
	l := fitparser.Lap{
		TotalElapsedTime:       600_500,
		TotalTimerTime:         598_250,
		TotalDistance:          201_050,
		EnhancedAvgSpeed:       3350,
		EnhancedMaxSpeed:       4875,
		AvgStanceTime:          2388,
		AvgVerticalOscillation: 842,
		AvgStepLength:          11456,
		AvgVerticalRatio:       735,
	}
	if l.ElapsedTime() != 600_500*time.Millisecond || l.TimerTime() != 598_250*time.Millisecond {
		t.Errorf("durations = %v, %v", l.ElapsedTime(), l.TimerTime())
	}
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"DistanceM", l.DistanceM(), 2010.5},
		{"AvgSpeedMps", l.AvgSpeedMps(), 3.35},
		{"MaxSpeedMps", l.MaxSpeedMps(), 4.875},
		{"AvgStanceTimeMs", l.AvgStanceTimeMs(), 238.8},
		{"AvgVerticalOscillationMm", l.AvgVerticalOscillationMm(), 84.2},
		{"AvgStepLengthM", l.AvgStepLengthM(), 1.1456},
		{"AvgVerticalRatioPercent", l.AvgVerticalRatioPercent(), 7.35},
	} {
		if !near(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestSplitAccessors(t *testing.T) {
	s := fitparser.Split{
		TotalElapsedTime:  245_000,
		TotalTimerTime:    240_000,
		TotalMovingTime:   238_500,
		TotalDistance:     100_025,
		AvgSpeed:          4166,
		MaxSpeed:          5012,
		StartElevation:    fitgen.Altitude(12.4),
		StartPositionLat:  fitgen.Degrees(-45),
		StartPositionLong: fitgen.Degrees(90),
		EndPositionLat:    fitgen.Degrees(30),
		EndPositionLong:   fitgen.Degrees(-60),
	}
	if s.ElapsedTime() != 245*time.Second || s.TimerTime() != 240*time.Second || s.MovingTime() != 238500*time.Millisecond {
		t.Errorf("durations = %v %v %v", s.ElapsedTime(), s.TimerTime(), s.MovingTime())
	}
	if !near(s.DistanceM(), 1000.25) || !near(s.AvgSpeedMps(), 4.166) || !near(s.MaxSpeedMps(), 5.012) {
		t.Errorf("distance/speed = %v %v %v", s.DistanceM(), s.AvgSpeedMps(), s.MaxSpeedMps())
	}
	if e, ok := s.StartElevationM(); !ok || !near(e, 12.4) {
		t.Errorf("StartElevationM = %v, %v", e, ok)
	}
	if lat, lon, ok := s.StartPosition(); !ok || !nearDeg(lat, -45) || !nearDeg(lon, 90) {
		t.Errorf("StartPosition = %v %v %v", lat, lon, ok)
	}
	if lat, lon, ok := s.EndPosition(); !ok || !nearDeg(lat, 30) || !nearDeg(lon, -60) {
		t.Errorf("EndPosition = %v %v %v", lat, lon, ok)
	}
	s.EndPositionLong = math.MaxInt32
	if _, _, ok := s.EndPosition(); ok {
		t.Error("EndPosition ok with invalid longitude")
	}
	s.StartElevation = 0
	if _, ok := s.StartElevationM(); ok {
		t.Error("StartElevationM ok when absent")
	}
}

func TestTimeInZoneAccessors(t *testing.T) {
	tz := fitparser.TimeInZone{
		TimeInHrZone:      []uint32{1000, 0, 2500},
		TimeInSpeedZone:   []uint32{60000},
		TimeInCadenceZone: []uint32{1, 2},
		TimeInPowerZone:   nil,
	}
	ms := time.Millisecond
	if got := tz.HrZoneTimes(); !reflect.DeepEqual(got, []time.Duration{time.Second, 0, 2500 * ms}) {
		t.Errorf("HrZoneTimes = %v", got)
	}
	if got := tz.SpeedZoneTimes(); !reflect.DeepEqual(got, []time.Duration{time.Minute}) {
		t.Errorf("SpeedZoneTimes = %v", got)
	}
	if got := tz.CadenceZoneTimes(); !reflect.DeepEqual(got, []time.Duration{ms, 2 * ms}) {
		t.Errorf("CadenceZoneTimes = %v", got)
	}
	if got := tz.PowerZoneTimes(); got != nil {
		t.Errorf("PowerZoneTimes = %v, want nil", got)
	}
}

func TestWorkoutAccessors(t *testing.T) {
	w := fitparser.Workout{PoolLength: 5000}
	if w.PoolLengthM() != 50 {
		t.Errorf("PoolLengthM = %v", w.PoolLengthM())
	}
	cases := []struct {
		step   fitparser.WorkoutStep
		dur    time.Duration
		durOK  bool
		distM  float64
		distOK bool
	}{
		{fitparser.WorkoutStep{DurationType: "time", DurationValue: 90_500}, 90500 * time.Millisecond, true, 0, false},
		{fitparser.WorkoutStep{DurationType: "distance", DurationValue: 100_050}, 0, false, 1000.5, true},
		{fitparser.WorkoutStep{DurationType: "open", DurationValue: 5}, 0, false, 0, false},
		{fitparser.WorkoutStep{DurationType: "hr_less_than", DurationValue: 150}, 0, false, 0, false},
		{fitparser.WorkoutStep{}, 0, false, 0, false},
	}
	for _, c := range cases {
		if d, ok := c.step.DurationTime(); d != c.dur || ok != c.durOK {
			t.Errorf("%+v DurationTime = %v, %v; want %v, %v", c.step, d, ok, c.dur, c.durOK)
		}
		if m, ok := c.step.DurationDistanceM(); !near(m, c.distM) || ok != c.distOK {
			t.Errorf("%+v DurationDistanceM = %v, %v; want %v, %v", c.step, m, ok, c.distM, c.distOK)
		}
	}
}

func TestLocation(t *testing.T) {
	f := &fitparser.File{UTCOffset: -(9*time.Hour + 30*time.Minute), HasUTCOffset: true}
	name, off := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).In(f.Location()).Zone()
	if name != "" || off != -(9*3600+30*60) {
		t.Errorf("zone = %q %d", name, off)
	}
}

// TestAccessorsOnDecodedRun checks the accessors against a decoded file,
// iterating the way callers do.
func TestAccessorsOnDecodedRun(t *testing.T) {
	f := decode(t, fitgen.New().AddRun(fitgen.Run{Start: t0, Seconds: 30, UTCOffset: time.Hour}))
	for i, r := range f.Records {
		if r.SpeedMps() != 3 || r.DistanceM() != float64(i)*3 {
			t.Errorf("record %d speed=%v dist=%v", i, r.SpeedMps(), r.DistanceM())
		}
		if alt, ok := r.AltitudeM(); !ok || alt != fitgen.RunAltitudeM {
			t.Errorf("record %d altitude = %v, %v", i, alt, ok)
		}
		lat, lon, ok := r.Position()
		wantLat := float64(fitgen.Degrees(fitgen.RunLat)+int32(i)*10) * 180 / (1 << 31)
		if !ok || !near(lat, wantLat) || math.Abs(lon-fitgen.RunLon) > 1e-6 {
			t.Errorf("record %d position = %v %v %v", i, lat, lon, ok)
		}
		if r.StanceTimeMs() != 250 || r.VerticalOscillationMm() != 85 || !near(r.StepLengthM(), 1.05) ||
			!near(r.VerticalRatioPercent(), 8.1) || !near(r.StanceTimeBalancePercent(), 50.1) || r.RespirationRateBrpm() != 30 {
			t.Errorf("record %d running dynamics = %+v", i, r)
		}
		if c, ok := r.TemperatureC(); !ok || c != 21 {
			t.Errorf("record %d temperature = %v %v", i, c, ok)
		}
		if r.HeartRate != fitgen.RunBaseHR+uint8(i%20) {
			t.Errorf("record %d HR = %d", i, r.HeartRate)
		}
	}
	s := &f.Sessions[0]
	if s.TimerTime() != 30*time.Second || s.DistanceM() != 90 || s.AvgSpeedMps() != 3 || s.MaxSpeedMps() != 3 {
		t.Errorf("session = %+v", s)
	}
	l := &f.Laps[0]
	if l.ElapsedTime() != 30*time.Second || l.DistanceM() != 90 {
		t.Errorf("lap = %+v", l)
	}
}
