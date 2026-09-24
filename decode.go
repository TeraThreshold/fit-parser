package fitparser

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/proto"
)

// maxRespirationRate is the largest accepted respiration rate in 0.01
// breaths/min (100 breaths/min). Larger values are near-sentinel junk seen
// in real device files and are dropped.
const maxRespirationRate = 10000

// maxUTCOffset bounds the local time offset derived from the activity
// message (real time zones span UTC-12 to UTC+14).
const maxUTCOffset = 14 * time.Hour

// minValidActivityTime is the earliest activity timestamp trusted for the
// UTC offset. FIT timestamps count from 1989-12-31; anything before 2000 is
// a device-relative or unset time.
var minValidActivityTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Decode reads a FIT file from r and returns its content. Every FIT
// sequence of a chained file is decoded and merged into one File.
//
// A file without session messages is not an error; callers that need an
// activity should check len(f.Sessions) > 0.
//
// Decode returns an error when r is empty, is not a FIT file, is truncated
// or fails its CRC check. Trailing bytes after at least one complete
// sequence that do not start with a FIT file header (for example zero
// padding) are ignored.
func Decode(r io.Reader) (f *File, err error) {
	// The upstream decoder is expected to return errors for malformed
	// input; this guard keeps the no-panic guarantee even if it does not.
	defer func() {
		if p := recover(); p != nil {
			f, err = nil, fmt.Errorf("fitparser: decode: malformed input: %v", p)
		}
	}()

	dec := decoder.New(r)
	b := &builder{file: &File{}}
	seq := 0
	for dec.Next() {
		fit, err := dec.Decode()
		if err != nil {
			if seq > 0 && errors.Is(err, decoder.ErrNotFITFile) {
				break
			}
			if seq == 0 {
				return nil, fmt.Errorf("fitparser: decode: %w", err)
			}
			return nil, fmt.Errorf("fitparser: decode: sequence %d: %w", seq+1, err)
		}
		b.addSequence(fit.Messages)
		seq++
	}
	if seq == 0 {
		return nil, fmt.Errorf("fitparser: decode: %w", io.ErrUnexpectedEOF)
	}
	return b.file, nil
}

// DecodeFile opens the FIT file at path and decodes it with Decode.
func DecodeFile(path string) (*File, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("fitparser: %w", err)
	}
	defer fh.Close()
	return Decode(fh)
}

// builder accumulates messages of all sequences into a File.
type builder struct {
	file *File
}

// addSequence merges the messages of one FIT sequence into the file.
func (b *builder) addSequence(msgs []proto.Message) {
	f := b.file

	// Developer field descriptions are scoped to their sequence and are
	// collected up front. The upstream decoder has already decoded every
	// developer value with the description in effect when the value was
	// read (fields without one are dropped), and like the decoder, the
	// first description for a (developer_data_index, field number) key is
	// the one that applies.
	dev := newDevContext()
	for i := range msgs {
		if msgs[i].Num == typedef.MesgNumFieldDescription {
			f.DeveloperFields = append(f.DeveloperFields, dev.addDescription(mesgdef.NewFieldDescription(&msgs[i])))
		}
	}
	if dev.hasStryd() {
		f.HasStryd = true
	}

	for i := range msgs {
		msg := &msgs[i]
		switch msg.Num {
		case typedef.MesgNumFileId:
			if f.FileID == nil {
				f.FileID = extractFileID(mesgdef.NewFileId(msg))
			}
		case typedef.MesgNumSession:
			f.Sessions = append(f.Sessions, extractSession(mesgdef.NewSession(msg)))
		case typedef.MesgNumLap:
			f.Laps = append(f.Laps, extractLap(mesgdef.NewLap(msg)))
		case typedef.MesgNumRecord:
			rec := extractRecord(mesgdef.NewRecord(msg))
			rec.DeveloperFields, rec.Stryd = dev.recordFields(msg.DeveloperFields)
			f.Records = append(f.Records, rec)
		case typedef.MesgNumEvent:
			f.Events = append(f.Events, extractEvent(mesgdef.NewEvent(msg)))
		case typedef.MesgNumDeviceInfo:
			f.DeviceInfos = append(f.DeviceInfos, extractDeviceInfo(mesgdef.NewDeviceInfo(msg)))
		case typedef.MesgNumSplit:
			f.Splits = append(f.Splits, extractSplit(mesgdef.NewSplit(msg)))
		case typedef.MesgNumTimeInZone:
			f.TimeInZones = append(f.TimeInZones, extractTimeInZone(mesgdef.NewTimeInZone(msg)))
		case typedef.MesgNumWorkout:
			f.Workouts = append(f.Workouts, extractWorkout(mesgdef.NewWorkout(msg)))
		case typedef.MesgNumWorkoutStep:
			f.WorkoutSteps = append(f.WorkoutSteps, extractWorkoutStep(mesgdef.NewWorkoutStep(msg)))
		case typedef.MesgNumZonesTarget:
			f.ZonesTarget = extractZonesTarget(mesgdef.NewZonesTarget(msg))
		case typedef.MesgNumHrZone:
			hz := mesgdef.NewHrZone(msg)
			f.HRZones = append(f.HRZones, HRZone{
				MessageIndex: messageIndex(hz.MessageIndex),
				Name:         hz.Name,
				HighBpm:      u8(hz.HighBpm),
			})
		case typedef.MesgNumPowerZone:
			pz := mesgdef.NewPowerZone(msg)
			f.PowerZones = append(f.PowerZones, PowerZone{
				MessageIndex: messageIndex(pz.MessageIndex),
				Name:         pz.Name,
				HighValue:    u16(pz.HighValue),
			})
		case typedef.MesgNumHrv:
			for _, t := range mesgdef.NewHrv(msg).Time {
				if t > 0 && t != basetype.Uint16Invalid {
					f.RRIntervals = append(f.RRIntervals, t)
				}
			}
		case typedef.MesgNumActivity:
			act := mesgdef.NewActivity(msg)
			if act.LocalTimestamp.After(minValidActivityTime) && act.Timestamp.After(minValidActivityTime) {
				off := act.LocalTimestamp.Sub(act.Timestamp)
				if off >= -maxUTCOffset && off <= maxUTCOffset {
					f.UTCOffset = off
					f.HasUTCOffset = true
				}
			}
		}
	}
}

// Sentinel helpers: the FIT invalid value of each base type becomes the Go
// zero value.

func u8(v uint8) uint8 {
	if v == basetype.Uint8Invalid {
		return 0
	}
	return v
}

func u16(v uint16) uint16 {
	if v == basetype.Uint16Invalid {
		return 0
	}
	return v
}

func u32(v uint32) uint32 {
	if v == basetype.Uint32Invalid {
		return 0
	}
	return v
}

func s16(v int16) int16 {
	if v == basetype.Sint16Invalid {
		return 0
	}
	return v
}

func s32(v int32) int32 {
	if v == basetype.Sint32Invalid {
		return 0
	}
	return v
}

// u32s filters every element of a uint32 array: invalid elements become 0.
// It returns nil when v is empty or when every element is invalid.
func u32s(v []uint32) []uint32 {
	out := make([]uint32, len(v))
	valid := false
	for i, x := range v {
		if x != basetype.Uint32Invalid {
			valid = true
		}
		out[i] = u32(x)
	}
	if !valid {
		return nil
	}
	return out
}

// u16s filters every element of a uint16 array: invalid elements become 0.
// It returns nil when v is empty or when every element is invalid.
func u16s(v []uint16) []uint16 {
	out := make([]uint16, len(v))
	valid := false
	for i, x := range v {
		if x != basetype.Uint16Invalid {
			valid = true
		}
		out[i] = u16(x)
	}
	if !valid {
		return nil
	}
	return out
}

// u8s filters every element of a uint8 array: invalid elements become 0.
// It returns nil when v is empty or when every element is invalid.
func u8s(v []uint8) []uint8 {
	out := make([]uint8, len(v))
	valid := false
	for i, x := range v {
		if x != basetype.Uint8Invalid {
			valid = true
		}
		out[i] = u8(x)
	}
	if !valid {
		return nil
	}
	return out
}

// respiration filters a 0.01 breaths/min value: the sentinel and values
// above 100 breaths/min become 0.
func respiration(v uint16) uint16 {
	if v > maxRespirationRate {
		return 0
	}
	return v
}

// messageIndex keeps the FIT invalid value (0 is a valid index) and strips
// the "selected" and reserved flag bits from valid indexes.
func messageIndex(v typedef.MessageIndex) uint16 {
	if v == typedef.MessageIndexInvalid {
		return uint16(typedef.MessageIndexInvalid)
	}
	return uint16(v & typedef.MessageIndexMask)
}

// utc returns t in UTC, keeping the zero time for invalid timestamps.
func utc(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return t.UTC()
}

func extractFileID(m *mesgdef.FileId) *FileID {
	return &FileID{
		Type:         fileTypeName(m.Type),
		Manufacturer: manufacturerName(m.Manufacturer),
		Product:      u16(m.Product),
		ProductName:  productName(m.ProductName, m.Product, m.GetProduct),
		SerialNumber: serial(m.SerialNumber),
		TimeCreated:  utc(m.TimeCreated),
	}
}

// serial filters a serial number. Its base type is uint32z (invalid 0), but
// some writers use the uint32 sentinel instead, so both become 0.
func serial(v uint32) uint32 {
	if v == basetype.Uint32zInvalid || v == basetype.Uint32Invalid {
		return 0
	}
	return v
}

func extractSession(s *mesgdef.Session) Session {
	sd := Session{
		Sport:                        sportName(s.Sport),
		SubSport:                     subSportName(s.SubSport),
		StartTime:                    utc(s.StartTime),
		TotalElapsedTime:             u32(s.TotalElapsedTime),
		TotalTimerTime:               u32(s.TotalTimerTime),
		TotalMovingTime:              u32(s.TotalMovingTime),
		TotalDistance:                u32(s.TotalDistance),
		EnhancedAvgSpeed:             u32(s.EnhancedAvgSpeed),
		EnhancedMaxSpeed:             u32(s.EnhancedMaxSpeed),
		TotalAscent:                  u16(s.TotalAscent),
		TotalDescent:                 u16(s.TotalDescent),
		TotalCalories:                u16(s.TotalCalories),
		TotalFatCalories:             u16(s.TotalFatCalories),
		TotalTrainingEffect:          u8(s.TotalTrainingEffect),
		TotalAnaerobicTrainingEffect: u8(s.TotalAnaerobicTrainingEffect),
		AvgHeartRate:                 u8(s.AvgHeartRate),
		MaxHeartRate:                 u8(s.MaxHeartRate),
		MinHeartRate:                 u8(s.MinHeartRate),
		AvgCadence:                   u8(s.AvgCadence),
		MaxCadence:                   u8(s.MaxCadence),
		AvgPower:                     u16(s.AvgPower),
		MaxPower:                     u16(s.MaxPower),
		NormalizedPower:              u16(s.NormalizedPower),
		TrainingStressScore:          u16(s.TrainingStressScore),
		IntensityFactor:              u16(s.IntensityFactor),
		TotalWork:                    u32(s.TotalWork),
		AvgStanceTime:                u16(s.AvgStanceTime),
		AvgVerticalOscillation:       u16(s.AvgVerticalOscillation),
		AvgStepLength:                u16(s.AvgStepLength),
		AvgVerticalRatio:             u16(s.AvgVerticalRatio),
		AvgStanceTimePercent:         u16(s.AvgStanceTimePercent),
		AvgStanceTimeBalance:         u16(s.AvgStanceTimeBalance),
		RmssdHrv:                     u8(s.RmssdHrv),
		SdrrHrv:                      u8(s.SdrrHrv),
		EnhancedAvgRespirationRate:   respiration(s.EnhancedAvgRespirationRate),
		EnhancedMaxRespirationRate:   respiration(s.EnhancedMaxRespirationRate),
		EnhancedMinRespirationRate:   respiration(s.EnhancedMinRespirationRate),
		AvgSpo2:                      u8(s.AvgSpo2),
		AvgStress:                    u8(s.AvgStress),
		WorkoutRpe:                   u8(s.WorkoutRpe),
		WorkoutFeel:                  u8(s.WorkoutFeel),
		AvgCoreTemperature:           u16(s.AvgCoreTemperature),
		MaxCoreTemperature:           u16(s.MaxCoreTemperature),
		AvgPosGrade:                  s16(s.AvgPosGrade),
		AvgNegGrade:                  s16(s.AvgNegGrade),
		MaxPosGrade:                  s16(s.MaxPosGrade),
		MaxNegGrade:                  s16(s.MaxNegGrade),
		MinAltitude:                  altitude16(s.EnhancedMinAltitude, s.MinAltitude),
		MaxAltitude:                  altitude16(s.EnhancedMaxAltitude, s.MaxAltitude),
		AvgVam:                       u16(s.AvgVam),
		NumLaps:                      u16(s.NumLaps),
		TotalCycles:                  u32(s.TotalCycles),
		TimeInHrZone:                 u32s(s.TimeInHrZone),
		TimeInPowerZone:              u32s(s.TimeInPowerZone),
		TrainingLoadPeak:             s32(s.TrainingLoadPeak),
		PoolLength:                   u16(s.PoolLength),
		NumActiveLengths:             u16(s.NumActiveLengths),
		AvgStrokeCount:               u32(s.AvgStrokeCount),
		AvgStrokeDistance:            u16(s.AvgStrokeDistance),
		SwimStroke:                   swimStrokeName(s.SwimStroke),
	}
	// The fractional part only means something next to its integer part.
	if sd.MaxCadence != 0 {
		sd.MaxFractionalCadence = u8(s.MaxFractionalCadence)
	}
	return sd
}

// altitude16 returns a session altitude (scale 5, offset 500) as the
// 16-bit raw value. The enhanced 32-bit field is preferred: the decoder fills
// it from the legacy field too, so it covers writers that emit either one.
// The legacy value is the fallback when the enhanced one is absent. A value
// that does not fit 16 bits becomes 0 (absent).
func altitude16(enhanced uint32, legacy uint16) uint16 {
	if enhanced != basetype.Uint32Invalid {
		if enhanced >= uint32(basetype.Uint16Invalid) {
			return 0
		}
		return uint16(enhanced)
	}
	return u16(legacy)
}

func extractLap(l *mesgdef.Lap) Lap {
	return Lap{
		StartTime:              utc(l.StartTime),
		TotalElapsedTime:       u32(l.TotalElapsedTime),
		TotalTimerTime:         u32(l.TotalTimerTime),
		TotalDistance:          u32(l.TotalDistance),
		AvgHeartRate:           u8(l.AvgHeartRate),
		MaxHeartRate:           u8(l.MaxHeartRate),
		AvgCadence:             u8(l.AvgCadence),
		MaxCadence:             u8(l.MaxCadence),
		AvgPower:               u16(l.AvgPower),
		MaxPower:               u16(l.MaxPower),
		TotalAscent:            u16(l.TotalAscent),
		TotalDescent:           u16(l.TotalDescent),
		TotalCalories:          u16(l.TotalCalories),
		EnhancedAvgSpeed:       u32(l.EnhancedAvgSpeed),
		EnhancedMaxSpeed:       u32(l.EnhancedMaxSpeed),
		AvgStanceTime:          u16(l.AvgStanceTime),
		AvgVerticalOscillation: u16(l.AvgVerticalOscillation),
		AvgStepLength:          u16(l.AvgStepLength),
		AvgVerticalRatio:       u16(l.AvgVerticalRatio),
		LapTrigger:             lapTriggerName(l.LapTrigger),
		Sport:                  sportName(l.Sport),
	}
}

func extractRecord(r *mesgdef.Record) Record {
	rd := Record{
		Timestamp:           utc(r.Timestamp),
		HeartRate:           u8(r.HeartRate),
		Cadence:             u8(r.Cadence),
		Power:               u16(r.Power),
		EnhancedSpeed:       u32(r.EnhancedSpeed),
		Distance:            u32(r.Distance),
		EnhancedAltitude:    u32(r.EnhancedAltitude),
		Grade:               s16(r.Grade),
		PositionLat:         r.PositionLat,  // invalid kept: 0 is a real coordinate
		PositionLong:        r.PositionLong, // invalid kept: 0 is a real coordinate
		Temperature:         r.Temperature,  // invalid kept: 0 °C is a real reading
		StanceTime:          u16(r.StanceTime),
		VerticalOscillation: u16(r.VerticalOscillation),
		StepLength:          u16(r.StepLength),
		VerticalRatio:       u16(r.VerticalRatio),
		StanceTimeBalance:   u16(r.StanceTimeBalance),
		GpsAccuracy:         u8(r.GpsAccuracy),
	}
	// The decoder expands legacy speed into enhanced_speed; the legacy value
	// is kept only as a fallback when the enhanced one is missing.
	if rd.EnhancedSpeed == 0 {
		rd.Speed = u16(r.Speed)
	}
	// Enhanced respiration is ×100 breaths/min; the legacy field is whole
	// breaths/min and is normalized to the same scale.
	if v := respiration(r.EnhancedRespirationRate); v > 0 {
		rd.EnhancedRespirationRate = v
	} else if v := u8(r.RespirationRate); v > 0 {
		rd.EnhancedRespirationRate = respiration(uint16(v) * 100)
	}
	return rd
}

func extractEvent(e *mesgdef.Event) Event {
	return Event{
		Timestamp: utc(e.Timestamp),
		Event:     eventName(e.Event),
		EventType: eventTypeName(e.EventType),
	}
}

func extractDeviceInfo(d *mesgdef.DeviceInfo) DeviceInfo {
	di := DeviceInfo{
		DeviceIndex:  uint8(d.DeviceIndex), // invalid kept: 0 is the creator device
		Manufacturer: manufacturerName(d.Manufacturer),
		ProductName:  productName(d.ProductName, d.Product, d.GetProduct),
		SerialNumber: serial(d.SerialNumber),
		BatteryLevel: u8(d.BatteryLevel),
	}
	if v := u16(d.SoftwareVersion); v > 0 {
		di.SoftwareVersion = float64(v) / 100
	}
	return di
}

func extractSplit(s *mesgdef.Split) Split {
	return Split{
		SplitType:         splitTypeName(s.SplitType),
		StartTime:         utc(s.StartTime),
		TotalElapsedTime:  u32(s.TotalElapsedTime),
		TotalTimerTime:    u32(s.TotalTimerTime),
		TotalMovingTime:   u32(s.TotalMovingTime),
		TotalDistance:     u32(s.TotalDistance),
		AvgSpeed:          u32(s.AvgSpeed),
		MaxSpeed:          u32(s.MaxSpeed),
		TotalAscent:       u16(s.TotalAscent),
		TotalDescent:      u16(s.TotalDescent),
		TotalCalories:     u32(s.TotalCalories),
		StartElevation:    u32(s.StartElevation),
		StartPositionLat:  s.StartPositionLat,
		StartPositionLong: s.StartPositionLong,
		EndPositionLat:    s.EndPositionLat,
		EndPositionLong:   s.EndPositionLong,
	}
}

func extractTimeInZone(tz *mesgdef.TimeInZone) TimeInZone {
	return TimeInZone{
		ReferenceMesg:            mesgNumName(tz.ReferenceMesg),
		ReferenceIndex:           messageIndex(tz.ReferenceIndex),
		TimeInHrZone:             u32s(tz.TimeInHrZone),
		TimeInSpeedZone:          u32s(tz.TimeInSpeedZone),
		TimeInCadenceZone:        u32s(tz.TimeInCadenceZone),
		TimeInPowerZone:          u32s(tz.TimeInPowerZone),
		HrZoneHighBoundary:       u8s(tz.HrZoneHighBoundary),
		PowerZoneHighBoundary:    u16s(tz.PowerZoneHighBoundary),
		MaxHeartRate:             u8(tz.MaxHeartRate),
		RestingHeartRate:         u8(tz.RestingHeartRate),
		ThresholdHeartRate:       u8(tz.ThresholdHeartRate),
		FunctionalThresholdPower: u16(tz.FunctionalThresholdPower),
	}
}

func extractWorkout(w *mesgdef.Workout) Workout {
	return Workout{
		WktName:        w.WktName,
		WktDescription: w.WktDescription,
		Sport:          sportName(w.Sport),
		SubSport:       subSportName(w.SubSport),
		NumValidSteps:  u16(w.NumValidSteps),
		PoolLength:     u16(w.PoolLength),
	}
}

func extractWorkoutStep(ws *mesgdef.WorkoutStep) WorkoutStep {
	return WorkoutStep{
		WktStepName:           ws.WktStepName,
		DurationType:          durationTypeName(ws.DurationType),
		DurationValue:         u32(ws.DurationValue),
		TargetType:            targetTypeName(ws.TargetType),
		TargetValue:           u32(ws.TargetValue),
		CustomTargetValueLow:  u32(ws.CustomTargetValueLow),
		CustomTargetValueHigh: u32(ws.CustomTargetValueHigh),
		Intensity:             intensityName(ws.Intensity),
	}
}

func extractZonesTarget(zt *mesgdef.ZonesTarget) *ZonesTarget {
	return &ZonesTarget{
		MaxHeartRate:             u8(zt.MaxHeartRate),
		ThresholdHeartRate:       u8(zt.ThresholdHeartRate),
		FunctionalThresholdPower: u16(zt.FunctionalThresholdPower),
		HrCalcType:               hrCalcTypeName(zt.HrCalcType),
		PwrCalcType:              pwrCalcTypeName(zt.PwrCalcType),
	}
}
