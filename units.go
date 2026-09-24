package fitparser

import (
	"time"

	"github.com/muktihari/fit/profile/basetype"
)

// semicirclesToDegrees converts FIT semicircles to degrees (2^31
// semicircles = 180°).
const semicirclesToDegrees = 180.0 / (1 << 31)

// position converts a semicircle pair into degrees. ok is false when either
// coordinate holds the FIT invalid value.
func position(lat, lon int32) (latDeg, lonDeg float64, ok bool) {
	if lat == basetype.Sint32Invalid || lon == basetype.Sint32Invalid {
		return 0, 0, false
	}
	return float64(lat) * semicirclesToDegrees, float64(lon) * semicirclesToDegrees, true
}

// altitudeM converts a raw FIT altitude (scale 5, offset 500) to meters.
// A zero raw value means absent.
func altitudeM(raw uint32) (float64, bool) {
	if raw == 0 {
		return 0, false
	}
	return float64(raw)/5 - 500, true
}

func msDuration(ms uint32) time.Duration { return time.Duration(ms) * time.Millisecond }

func msDurations(ms []uint32) []time.Duration {
	if len(ms) == 0 {
		return nil
	}
	out := make([]time.Duration, len(ms))
	for i, v := range ms {
		out[i] = msDuration(v)
	}
	return out
}

// Location returns a fixed time zone for the file's UTCOffset, or time.UTC
// when the file carries no usable local timestamp. Use it to show record
// and session times in the activity's local time:
// rec.Timestamp.In(f.Location()).
func (f *File) Location() *time.Location {
	if f == nil || !f.HasUTCOffset {
		return time.UTC
	}
	return time.FixedZone("", int(f.UTCOffset/time.Second))
}

// SpeedMps returns the speed in m/s, preferring EnhancedSpeed and falling
// back to the legacy Speed field; 0 when neither is set.
func (r *Record) SpeedMps() float64 {
	if r.EnhancedSpeed > 0 {
		return float64(r.EnhancedSpeed) / 1000
	}
	return float64(r.Speed) / 1000
}

// DistanceM returns the cumulative distance in meters.
func (r *Record) DistanceM() float64 { return float64(r.Distance) / 100 }

// AltitudeM returns the altitude in meters. ok is false when the record has
// no altitude.
func (r *Record) AltitudeM() (m float64, ok bool) { return altitudeM(r.EnhancedAltitude) }

// GradePercent returns the grade in percent.
func (r *Record) GradePercent() float64 { return float64(r.Grade) / 100 }

// StanceTimeMs returns the ground contact time in milliseconds.
func (r *Record) StanceTimeMs() float64 { return float64(r.StanceTime) / 10 }

// VerticalOscillationMm returns the vertical oscillation in millimeters.
func (r *Record) VerticalOscillationMm() float64 { return float64(r.VerticalOscillation) / 10 }

// StepLengthM returns the step length in meters.
func (r *Record) StepLengthM() float64 { return float64(r.StepLength) / 10000 }

// VerticalRatioPercent returns the vertical ratio in percent.
func (r *Record) VerticalRatioPercent() float64 { return float64(r.VerticalRatio) / 100 }

// StanceTimeBalancePercent returns the ground contact time balance (left
// share) in percent.
func (r *Record) StanceTimeBalancePercent() float64 { return float64(r.StanceTimeBalance) / 100 }

// RespirationRateBrpm returns the respiration rate in breaths per minute.
func (r *Record) RespirationRateBrpm() float64 { return float64(r.EnhancedRespirationRate) / 100 }

// Position returns the record's position in degrees (WGS84). ok is false
// when the record has no position.
func (r *Record) Position() (lat, lon float64, ok bool) {
	return position(r.PositionLat, r.PositionLong)
}

// TemperatureC returns the ambient temperature in °C. ok is false when the
// record has no temperature.
func (r *Record) TemperatureC() (c float64, ok bool) {
	if r.Temperature == basetype.Sint8Invalid {
		return 0, false
	}
	return float64(r.Temperature), true
}

// ElapsedTime returns the elapsed time including pauses.
func (s *Session) ElapsedTime() time.Duration { return msDuration(s.TotalElapsedTime) }

// TimerTime returns the timer time excluding pauses.
func (s *Session) TimerTime() time.Duration { return msDuration(s.TotalTimerTime) }

// MovingTime returns the time spent moving.
func (s *Session) MovingTime() time.Duration { return msDuration(s.TotalMovingTime) }

// DistanceM returns the session distance in meters.
func (s *Session) DistanceM() float64 { return float64(s.TotalDistance) / 100 }

// AvgSpeedMps returns the average speed in m/s.
func (s *Session) AvgSpeedMps() float64 { return float64(s.EnhancedAvgSpeed) / 1000 }

// MaxSpeedMps returns the maximum speed in m/s.
func (s *Session) MaxSpeedMps() float64 { return float64(s.EnhancedMaxSpeed) / 1000 }

// MinAltitudeM returns the minimum altitude in meters. ok is false when the
// session has no minimum altitude.
func (s *Session) MinAltitudeM() (m float64, ok bool) { return altitudeM(uint32(s.MinAltitude)) }

// MaxAltitudeM returns the maximum altitude in meters. ok is false when the
// session has no maximum altitude.
func (s *Session) MaxAltitudeM() (m float64, ok bool) { return altitudeM(uint32(s.MaxAltitude)) }

// AvgStanceTimeMs returns the average ground contact time in milliseconds.
func (s *Session) AvgStanceTimeMs() float64 { return float64(s.AvgStanceTime) / 10 }

// AvgVerticalOscillationMm returns the average vertical oscillation in
// millimeters.
func (s *Session) AvgVerticalOscillationMm() float64 { return float64(s.AvgVerticalOscillation) / 10 }

// AvgStepLengthM returns the average step length in meters.
func (s *Session) AvgStepLengthM() float64 { return float64(s.AvgStepLength) / 10000 }

// AvgVerticalRatioPercent returns the average vertical ratio in percent.
func (s *Session) AvgVerticalRatioPercent() float64 { return float64(s.AvgVerticalRatio) / 100 }

// AvgStanceTimeBalancePercent returns the average ground contact time
// balance (left share) in percent.
func (s *Session) AvgStanceTimeBalancePercent() float64 {
	return float64(s.AvgStanceTimeBalance) / 100
}

// PoolLengthM returns the swimming pool length in meters.
func (s *Session) PoolLengthM() float64 { return float64(s.PoolLength) / 100 }

// ElapsedTime returns the elapsed time including pauses.
func (l *Lap) ElapsedTime() time.Duration { return msDuration(l.TotalElapsedTime) }

// TimerTime returns the timer time excluding pauses.
func (l *Lap) TimerTime() time.Duration { return msDuration(l.TotalTimerTime) }

// DistanceM returns the lap distance in meters.
func (l *Lap) DistanceM() float64 { return float64(l.TotalDistance) / 100 }

// AvgSpeedMps returns the average speed in m/s.
func (l *Lap) AvgSpeedMps() float64 { return float64(l.EnhancedAvgSpeed) / 1000 }

// MaxSpeedMps returns the maximum speed in m/s.
func (l *Lap) MaxSpeedMps() float64 { return float64(l.EnhancedMaxSpeed) / 1000 }

// AvgStanceTimeMs returns the average ground contact time in milliseconds.
func (l *Lap) AvgStanceTimeMs() float64 { return float64(l.AvgStanceTime) / 10 }

// AvgVerticalOscillationMm returns the average vertical oscillation in
// millimeters.
func (l *Lap) AvgVerticalOscillationMm() float64 { return float64(l.AvgVerticalOscillation) / 10 }

// AvgStepLengthM returns the average step length in meters.
func (l *Lap) AvgStepLengthM() float64 { return float64(l.AvgStepLength) / 10000 }

// AvgVerticalRatioPercent returns the average vertical ratio in percent.
func (l *Lap) AvgVerticalRatioPercent() float64 { return float64(l.AvgVerticalRatio) / 100 }

// ElapsedTime returns the split's elapsed time.
func (s *Split) ElapsedTime() time.Duration { return msDuration(s.TotalElapsedTime) }

// TimerTime returns the split's timer time.
func (s *Split) TimerTime() time.Duration { return msDuration(s.TotalTimerTime) }

// MovingTime returns the time spent moving during the split.
func (s *Split) MovingTime() time.Duration { return msDuration(s.TotalMovingTime) }

// DistanceM returns the split distance in meters.
func (s *Split) DistanceM() float64 { return float64(s.TotalDistance) / 100 }

// AvgSpeedMps returns the average speed in m/s.
func (s *Split) AvgSpeedMps() float64 { return float64(s.AvgSpeed) / 1000 }

// MaxSpeedMps returns the maximum speed in m/s.
func (s *Split) MaxSpeedMps() float64 { return float64(s.MaxSpeed) / 1000 }

// StartElevationM returns the starting elevation in meters. ok is false
// when the split has no starting elevation.
func (s *Split) StartElevationM() (m float64, ok bool) { return altitudeM(s.StartElevation) }

// StartPosition returns the split's start position in degrees. ok is false
// when absent.
func (s *Split) StartPosition() (lat, lon float64, ok bool) {
	return position(s.StartPositionLat, s.StartPositionLong)
}

// EndPosition returns the split's end position in degrees. ok is false when
// absent.
func (s *Split) EndPosition() (lat, lon float64, ok bool) {
	return position(s.EndPositionLat, s.EndPositionLong)
}

// HrZoneTimes returns TimeInHrZone as durations.
func (t *TimeInZone) HrZoneTimes() []time.Duration { return msDurations(t.TimeInHrZone) }

// SpeedZoneTimes returns TimeInSpeedZone as durations.
func (t *TimeInZone) SpeedZoneTimes() []time.Duration { return msDurations(t.TimeInSpeedZone) }

// CadenceZoneTimes returns TimeInCadenceZone as durations.
func (t *TimeInZone) CadenceZoneTimes() []time.Duration {
	return msDurations(t.TimeInCadenceZone)
}

// PowerZoneTimes returns TimeInPowerZone as durations.
func (t *TimeInZone) PowerZoneTimes() []time.Duration { return msDurations(t.TimeInPowerZone) }

// PoolLengthM returns the swimming pool length in meters.
func (w *Workout) PoolLengthM() float64 { return float64(w.PoolLength) / 100 }

// DurationTime returns the step duration when DurationType is "time". ok is
// false for other duration types.
func (w *WorkoutStep) DurationTime() (d time.Duration, ok bool) {
	if w.DurationType != "time" {
		return 0, false
	}
	return msDuration(w.DurationValue), true
}

// DurationDistanceM returns the step distance in meters when DurationType is
// "distance". ok is false for other duration types.
func (w *WorkoutStep) DurationDistanceM() (m float64, ok bool) {
	if w.DurationType != "distance" {
		return 0, false
	}
	return float64(w.DurationValue) / 100, true
}
