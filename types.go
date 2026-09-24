package fitparser

import (
	"encoding/json"
	"time"
)

// File is the decoded content of a FIT file. When the input is a chained FIT
// file (several FIT sequences back to back), the messages of every sequence
// are merged into one File in file order.
//
// A File without sessions is valid: workout, course and settings files carry
// no session message. Callers that expect an activity should check
// len(f.Sessions) > 0.
type File struct {
	// FileID is the first file_id message of the file, or nil when absent.
	FileID *FileID `json:"file_id,omitempty"`
	// Sessions holds one entry per session message (usually one per activity,
	// several for multisport activities).
	Sessions []Session `json:"sessions,omitempty"`
	// Laps holds one entry per lap message.
	Laps []Lap `json:"laps,omitempty"`
	// Records holds one entry per record message (typically one per second).
	Records []Record `json:"records,omitempty"`
	// Events holds one entry per event message (timer start/stop, laps, ...).
	Events []Event `json:"events,omitempty"`
	// DeviceInfos holds one entry per device_info message (the recording
	// device and its connected sensors).
	DeviceInfos []DeviceInfo `json:"device_infos,omitempty"`
	// Splits holds one entry per split message (run/walk/climb splits).
	Splits []Split `json:"splits,omitempty"`
	// TimeInZones holds one entry per time_in_zone message.
	TimeInZones []TimeInZone `json:"time_in_zones,omitempty"`
	// Workouts holds one entry per workout message.
	Workouts []Workout `json:"workouts,omitempty"`
	// WorkoutSteps holds one entry per workout_step message, in file order.
	WorkoutSteps []WorkoutStep `json:"workout_steps,omitempty"`
	// ZonesTarget is the last zones_target message of the file, or nil when
	// absent.
	ZonesTarget *ZonesTarget `json:"zones_target,omitempty"`
	// HRZones holds one entry per hr_zone message, in file order.
	HRZones []HRZone `json:"hr_zones,omitempty"`
	// PowerZones holds one entry per power_zone message, in file order.
	PowerZones []PowerZone `json:"power_zones,omitempty"`
	// RRIntervals holds the beat-to-beat (R-R) intervals of all hrv messages
	// in file order, in milliseconds. Invalid (0xFFFF) and zero values are
	// dropped.
	RRIntervals []uint16 `json:"rr_intervals,omitempty"`
	// DeveloperFields holds every field_description message of the file.
	DeveloperFields []DeveloperFieldDescription `json:"developer_fields,omitempty"`
	// HasStryd reports whether a developer data index of the file describes
	// the Stryd-specific fields "Leg Spring Stiffness" or "Form Power".
	HasStryd bool `json:"has_stryd,omitempty"`
	// UTCOffset is the offset of the activity's local time from UTC, derived
	// from activity.local_timestamp - activity.timestamp. It is only set when
	// both timestamps are after 2000-01-01 and the offset is within ±14h; see
	// HasUTCOffset. When several activity messages qualify, the last one wins.
	// In JSON it is encoded as integer nanoseconds (time.Duration), for
	// example 7200000000000 for +02:00.
	UTCOffset time.Duration `json:"utc_offset,omitempty"`
	// HasUTCOffset reports whether UTCOffset was derived from the file.
	HasUTCOffset bool `json:"has_utc_offset,omitempty"`
}

// FileID is the file_id message that identifies the file and its creator.
type FileID struct {
	// Type is the FIT file type, e.g. "activity", "workout", "course".
	Type string `json:"type,omitempty"`
	// Manufacturer is the FIT manufacturer name, e.g. "garmin", "wahoo_fitness".
	Manufacturer string `json:"manufacturer,omitempty"`
	// Product is the raw manufacturer-specific product number.
	Product uint16 `json:"product,omitempty"`
	// ProductName is the file's product_name string when set, otherwise the
	// product profile name for Garmin-family and Favero manufacturers
	// (e.g. "fr965"), otherwise "".
	ProductName string `json:"product_name,omitempty"`
	// SerialNumber is the creator device's serial number.
	SerialNumber uint32 `json:"serial_number,omitempty"`
	// TimeCreated is when the file was created (UTC).
	TimeCreated time.Time `json:"time_created,omitzero"`
}

// Session is a session message: the summary of one activity (or of one leg
// of a multisport activity). Numeric fields hold raw FIT units; use the
// accessor methods for SI values. A zero value means the field was absent or
// invalid.
type Session struct {
	// Sport is the FIT sport name, e.g. "running", "cycling".
	Sport string `json:"sport,omitempty"`
	// SubSport is the FIT sub-sport name, e.g. "generic", "treadmill",
	// "trail", "track".
	SubSport string `json:"sub_sport,omitempty"`
	// StartTime is when the session started (UTC).
	StartTime time.Time `json:"start_time,omitzero"`
	// TotalElapsedTime is the elapsed time including pauses, in milliseconds.
	TotalElapsedTime uint32 `json:"total_elapsed_time,omitempty"`
	// TotalTimerTime is the timer time excluding pauses, in milliseconds.
	TotalTimerTime uint32 `json:"total_timer_time,omitempty"`
	// TotalMovingTime is the time spent moving, in milliseconds.
	TotalMovingTime uint32 `json:"total_moving_time,omitempty"`
	// TotalDistance is the session distance, in centimeters.
	TotalDistance uint32 `json:"total_distance,omitempty"`
	// EnhancedAvgSpeed is the average speed, in millimeters per second.
	EnhancedAvgSpeed uint32 `json:"enhanced_avg_speed,omitempty"`
	// EnhancedMaxSpeed is the maximum speed, in millimeters per second.
	EnhancedMaxSpeed uint32 `json:"enhanced_max_speed,omitempty"`
	// TotalAscent is the total ascent, in meters.
	TotalAscent uint16 `json:"total_ascent,omitempty"`
	// TotalDescent is the total descent, in meters.
	TotalDescent uint16 `json:"total_descent,omitempty"`
	// TotalCalories is the energy expenditure, in kilocalories.
	TotalCalories uint16 `json:"total_calories,omitempty"`
	// TotalFatCalories is the energy from fat, in kilocalories.
	TotalFatCalories uint16 `json:"total_fat_calories,omitempty"`
	// TotalTrainingEffect is the aerobic training effect ×10 (e.g. 35 = 3.5).
	TotalTrainingEffect uint8 `json:"total_training_effect,omitempty"`
	// TotalAnaerobicTrainingEffect is the anaerobic training effect ×10.
	TotalAnaerobicTrainingEffect uint8 `json:"total_anaerobic_training_effect,omitempty"`
	// AvgHeartRate is the average heart rate, in beats per minute.
	AvgHeartRate uint8 `json:"avg_heart_rate,omitempty"`
	// MaxHeartRate is the maximum heart rate, in beats per minute.
	MaxHeartRate uint8 `json:"max_heart_rate,omitempty"`
	// MinHeartRate is the minimum heart rate, in beats per minute.
	MinHeartRate uint8 `json:"min_heart_rate,omitempty"`
	// AvgCadence is the average cadence, in rpm (running: strides per minute
	// per leg, i.e. half the step rate).
	AvgCadence uint8 `json:"avg_cadence,omitempty"`
	// MaxCadence is the maximum cadence, in rpm.
	MaxCadence uint8 `json:"max_cadence,omitempty"`
	// MaxFractionalCadence is the fractional part of MaxCadence, in 1/128 rpm.
	// It is only set when MaxCadence is set.
	MaxFractionalCadence uint8 `json:"max_fractional_cadence,omitempty"`
	// AvgPower is the average power, in watts.
	AvgPower uint16 `json:"avg_power,omitempty"`
	// MaxPower is the maximum power, in watts.
	MaxPower uint16 `json:"max_power,omitempty"`
	// NormalizedPower is the normalized power, in watts.
	NormalizedPower uint16 `json:"normalized_power,omitempty"`
	// TrainingStressScore is the training stress score ×10.
	TrainingStressScore uint16 `json:"training_stress_score,omitempty"`
	// IntensityFactor is the intensity factor ×1000.
	IntensityFactor uint16 `json:"intensity_factor,omitempty"`
	// TotalWork is the total work, in joules.
	TotalWork uint32 `json:"total_work,omitempty"`
	// AvgStanceTime is the average ground contact time, in 0.1 ms.
	AvgStanceTime uint16 `json:"avg_stance_time,omitempty"`
	// AvgVerticalOscillation is the average vertical oscillation, in 0.1 mm.
	AvgVerticalOscillation uint16 `json:"avg_vertical_oscillation,omitempty"`
	// AvgStepLength is the average step length, in 0.1 mm.
	AvgStepLength uint16 `json:"avg_step_length,omitempty"`
	// AvgVerticalRatio is the average vertical ratio, in 0.01 %.
	AvgVerticalRatio uint16 `json:"avg_vertical_ratio,omitempty"`
	// AvgStanceTimePercent is the average stance time percentage, in 0.01 %.
	AvgStanceTimePercent uint16 `json:"avg_stance_time_percent,omitempty"`
	// AvgStanceTimeBalance is the average ground contact time balance (left
	// share), in 0.01 %.
	AvgStanceTimeBalance uint16 `json:"avg_stance_time_balance,omitempty"`
	// RmssdHrv is the RMSSD heart rate variability, in milliseconds.
	RmssdHrv uint8 `json:"rmssd_hrv,omitempty"`
	// SdrrHrv is the SDRR heart rate variability, in milliseconds.
	SdrrHrv uint8 `json:"sdrr_hrv,omitempty"`
	// EnhancedAvgRespirationRate is the average respiration rate, in 0.01
	// breaths per minute. Values above 10000 (100 breaths/min) are rejected.
	EnhancedAvgRespirationRate uint16 `json:"enhanced_avg_respiration_rate,omitempty"`
	// EnhancedMaxRespirationRate is the maximum respiration rate, in 0.01
	// breaths per minute. Values above 10000 are rejected.
	EnhancedMaxRespirationRate uint16 `json:"enhanced_max_respiration_rate,omitempty"`
	// EnhancedMinRespirationRate is the minimum respiration rate, in 0.01
	// breaths per minute. Values above 10000 are rejected.
	EnhancedMinRespirationRate uint16 `json:"enhanced_min_respiration_rate,omitempty"`
	// AvgSpo2 is the average blood oxygen saturation, in percent.
	AvgSpo2 uint8 `json:"avg_spo2,omitempty"`
	// AvgStress is the average stress level, in percent.
	AvgStress uint8 `json:"avg_stress,omitempty"`
	// WorkoutRpe is the rate of perceived exertion (Borg CR10) ×10.
	WorkoutRpe uint8 `json:"workout_rpe,omitempty"`
	// WorkoutFeel is how the athlete felt, on a 0-100 scale (higher is better).
	WorkoutFeel uint8 `json:"workout_feel,omitempty"`
	// AvgCoreTemperature is the average core body temperature, in 0.01 °C.
	AvgCoreTemperature uint16 `json:"avg_core_temperature,omitempty"`
	// MaxCoreTemperature is the maximum core body temperature, in 0.01 °C.
	MaxCoreTemperature uint16 `json:"max_core_temperature,omitempty"`
	// AvgPosGrade is the average uphill grade, in 0.01 %.
	AvgPosGrade int16 `json:"avg_pos_grade,omitempty"`
	// AvgNegGrade is the average downhill grade, in 0.01 %.
	AvgNegGrade int16 `json:"avg_neg_grade,omitempty"`
	// MaxPosGrade is the maximum uphill grade, in 0.01 %.
	MaxPosGrade int16 `json:"max_pos_grade,omitempty"`
	// MaxNegGrade is the maximum downhill grade, in 0.01 %.
	MaxNegGrade int16 `json:"max_neg_grade,omitempty"`
	// MinAltitude is the minimum altitude, raw FIT value with scale 5 and
	// offset 500 (meters = raw/5 - 500); see MinAltitudeM. It is read from
	// enhanced_min_altitude, which the decoder also fills from the legacy
	// 16-bit min_altitude field, so files that write either field are
	// covered. A value that does not fit 16 bits is treated as absent.
	MinAltitude uint16 `json:"min_altitude,omitempty"`
	// MaxAltitude is the maximum altitude, raw FIT value with scale 5 and
	// offset 500 (meters = raw/5 - 500); see MaxAltitudeM. It is read from
	// enhanced_max_altitude, which the decoder also fills from the legacy
	// 16-bit max_altitude field, so files that write either field are
	// covered. A value that does not fit 16 bits is treated as absent.
	MaxAltitude uint16 `json:"max_altitude,omitempty"`
	// AvgVam is the average vertical ascent speed, in mm/s (0.001 m/s).
	AvgVam uint16 `json:"avg_vam,omitempty"`
	// NumLaps is the number of laps in the session.
	NumLaps uint16 `json:"num_laps,omitempty"`
	// TotalCycles is the total number of cycles (strides for running,
	// crank revolutions for cycling, strokes for swimming).
	TotalCycles uint32 `json:"total_cycles,omitempty"`
	// TimeInHrZone is the time spent in each heart rate zone, in
	// milliseconds. Invalid elements are zero; nil when absent or when
	// every element is invalid.
	TimeInHrZone []uint32 `json:"time_in_hr_zone,omitempty"`
	// TimeInPowerZone is the time spent in each power zone, in milliseconds.
	// Invalid elements are zero; nil when absent or when every element is
	// invalid.
	TimeInPowerZone []uint32 `json:"time_in_power_zone,omitempty"`
	// TrainingLoadPeak is the device's training load peak, in 1/65536 units.
	TrainingLoadPeak int32 `json:"training_load_peak,omitempty"`
	// PoolLength is the swimming pool length, in centimeters.
	PoolLength uint16 `json:"pool_length,omitempty"`
	// NumActiveLengths is the number of active (swimming) pool lengths.
	NumActiveLengths uint16 `json:"num_active_lengths,omitempty"`
	// AvgStrokeCount is the average stroke count per length, in 0.1 strokes.
	AvgStrokeCount uint32 `json:"avg_stroke_count,omitempty"`
	// AvgStrokeDistance is the average distance per stroke, in centimeters.
	AvgStrokeDistance uint16 `json:"avg_stroke_distance,omitempty"`
	// SwimStroke is the FIT swim stroke name, e.g. "freestyle", "mixed".
	SwimStroke string `json:"swim_stroke,omitempty"`
}

// Lap is a lap message. Numeric fields hold raw FIT units; use the accessor
// methods for SI values. A zero value means the field was absent or invalid.
type Lap struct {
	// StartTime is when the lap started (UTC).
	StartTime time.Time `json:"start_time,omitzero"`
	// TotalElapsedTime is the elapsed time including pauses, in milliseconds.
	TotalElapsedTime uint32 `json:"total_elapsed_time,omitempty"`
	// TotalTimerTime is the timer time excluding pauses, in milliseconds.
	TotalTimerTime uint32 `json:"total_timer_time,omitempty"`
	// TotalDistance is the lap distance, in centimeters.
	TotalDistance uint32 `json:"total_distance,omitempty"`
	// AvgHeartRate is the average heart rate, in beats per minute.
	AvgHeartRate uint8 `json:"avg_heart_rate,omitempty"`
	// MaxHeartRate is the maximum heart rate, in beats per minute.
	MaxHeartRate uint8 `json:"max_heart_rate,omitempty"`
	// AvgCadence is the average cadence, in rpm.
	AvgCadence uint8 `json:"avg_cadence,omitempty"`
	// MaxCadence is the maximum cadence, in rpm.
	MaxCadence uint8 `json:"max_cadence,omitempty"`
	// AvgPower is the average power, in watts.
	AvgPower uint16 `json:"avg_power,omitempty"`
	// MaxPower is the maximum power, in watts.
	MaxPower uint16 `json:"max_power,omitempty"`
	// TotalAscent is the total ascent, in meters.
	TotalAscent uint16 `json:"total_ascent,omitempty"`
	// TotalDescent is the total descent, in meters.
	TotalDescent uint16 `json:"total_descent,omitempty"`
	// TotalCalories is the energy expenditure, in kilocalories.
	TotalCalories uint16 `json:"total_calories,omitempty"`
	// EnhancedAvgSpeed is the average speed, in millimeters per second.
	EnhancedAvgSpeed uint32 `json:"enhanced_avg_speed,omitempty"`
	// EnhancedMaxSpeed is the maximum speed, in millimeters per second.
	EnhancedMaxSpeed uint32 `json:"enhanced_max_speed,omitempty"`
	// AvgStanceTime is the average ground contact time, in 0.1 ms.
	AvgStanceTime uint16 `json:"avg_stance_time,omitempty"`
	// AvgVerticalOscillation is the average vertical oscillation, in 0.1 mm.
	AvgVerticalOscillation uint16 `json:"avg_vertical_oscillation,omitempty"`
	// AvgStepLength is the average step length, in 0.1 mm.
	AvgStepLength uint16 `json:"avg_step_length,omitempty"`
	// AvgVerticalRatio is the average vertical ratio, in 0.01 %.
	AvgVerticalRatio uint16 `json:"avg_vertical_ratio,omitempty"`
	// LapTrigger is the FIT lap trigger name, e.g. "manual", "distance",
	// "session_end".
	LapTrigger string `json:"lap_trigger,omitempty"`
	// Sport is the FIT sport name of the lap, e.g. "running".
	Sport string `json:"sport,omitempty"`
}

// Record is a record message: one sample of the activity's time series.
// Numeric fields hold raw FIT units; use the accessor methods for SI values.
// A zero value means the field was absent or invalid, except for the
// position and temperature fields, which keep the FIT invalid value because
// zero is a legitimate measurement (use Position and TemperatureC).
type Record struct {
	// Timestamp is the sample time (UTC).
	Timestamp time.Time `json:"timestamp,omitzero"`
	// HeartRate is the heart rate, in beats per minute.
	HeartRate uint8 `json:"heart_rate,omitempty"`
	// Cadence is the cadence, in rpm (running: strides per minute per leg).
	Cadence uint8 `json:"cadence,omitempty"`
	// Power is the power, in watts. No upper plausibility limit is applied:
	// only the FIT invalid value 0xFFFF is dropped.
	Power uint16 `json:"power,omitempty"`
	// EnhancedSpeed is the speed, in millimeters per second.
	EnhancedSpeed uint32 `json:"enhanced_speed,omitempty"`
	// Speed is the legacy 16-bit speed, in millimeters per second. It is
	// only set when EnhancedSpeed is absent; SpeedMps handles the fallback.
	Speed uint16 `json:"speed,omitempty"`
	// Distance is the cumulative distance, in centimeters.
	Distance uint32 `json:"distance,omitempty"`
	// EnhancedAltitude is the altitude, raw FIT value with scale 5 and offset
	// 500 (meters = raw/5 - 500); see AltitudeM.
	EnhancedAltitude uint32 `json:"enhanced_altitude,omitempty"`
	// Grade is the grade, in 0.01 %.
	Grade int16 `json:"grade,omitempty"`
	// PositionLat is the latitude, in semicircles. math.MaxInt32 (the FIT
	// invalid value) means no position; see Position.
	PositionLat int32 `json:"position_lat"`
	// PositionLong is the longitude, in semicircles. math.MaxInt32 (the FIT
	// invalid value) means no position; see Position.
	PositionLong int32 `json:"position_long"`
	// Temperature is the ambient temperature, in °C. 127 (the FIT invalid
	// value) means no temperature; see TemperatureC.
	Temperature int8 `json:"temperature"`
	// StanceTime is the ground contact time, in 0.1 ms.
	StanceTime uint16 `json:"stance_time,omitempty"`
	// VerticalOscillation is the vertical oscillation, in 0.1 mm.
	VerticalOscillation uint16 `json:"vertical_oscillation,omitempty"`
	// StepLength is the step length, in 0.1 mm.
	StepLength uint16 `json:"step_length,omitempty"`
	// VerticalRatio is the vertical ratio, in 0.01 %.
	VerticalRatio uint16 `json:"vertical_ratio,omitempty"`
	// StanceTimeBalance is the ground contact time balance (left share), in
	// 0.01 %.
	StanceTimeBalance uint16 `json:"stance_time_balance,omitempty"`
	// EnhancedRespirationRate is the respiration rate, in 0.01 breaths per
	// minute. When the enhanced field is absent, the legacy respiration_rate
	// field (whole breaths per minute) is used, multiplied by 100. Values
	// above 10000 (100 breaths/min) are rejected.
	EnhancedRespirationRate uint16 `json:"enhanced_respiration_rate,omitempty"`
	// GpsAccuracy is the GPS accuracy, in meters.
	GpsAccuracy uint8 `json:"gps_accuracy,omitempty"`
	// DeveloperFields holds the record's developer (Connect IQ) field values
	// whose field_description is known.
	DeveloperFields []DeveloperFieldValue `json:"developer_fields,omitempty"`
	// Stryd holds the record's Stryd developer metrics. It is nil unless the
	// file HasStryd and the record carries at least one valid Stryd field.
	Stryd *Stryd `json:"stryd,omitempty"`
}

// Event is an event message, e.g. a timer start or stop.
type Event struct {
	// Timestamp is when the event happened (UTC).
	Timestamp time.Time `json:"timestamp,omitzero"`
	// Event is the FIT event name, e.g. "timer", "lap", "session".
	Event string `json:"event,omitempty"`
	// EventType is the FIT event type name, e.g. "start", "stop_all".
	EventType string `json:"event_type,omitempty"`
}

// DeviceInfo is a device_info message describing the recording device or a
// connected sensor.
type DeviceInfo struct {
	// DeviceIndex identifies the device within the file; 0 is the creator
	// (recording) device. Because 0 is meaningful, an absent device index
	// keeps the FIT invalid value 255.
	DeviceIndex uint8 `json:"device_index"`
	// Manufacturer is the FIT manufacturer name, e.g. "garmin".
	Manufacturer string `json:"manufacturer,omitempty"`
	// ProductName is the message's product_name string when set, otherwise
	// the product profile name for Garmin-family and Favero manufacturers
	// (e.g. "hrm_pro"), otherwise "".
	ProductName string `json:"product_name,omitempty"`
	// SerialNumber is the device's serial number.
	SerialNumber uint32 `json:"serial_number,omitempty"`
	// SoftwareVersion is the device's software version (raw value / 100,
	// e.g. 12.34).
	SoftwareVersion float64 `json:"software_version,omitempty"`
	// BatteryLevel is the battery level, in percent.
	BatteryLevel uint8 `json:"battery_level,omitempty"`
}

// Split is a split message (e.g. a run, walk, climb or rest interval).
// Numeric fields hold raw FIT units; use the accessor methods for SI values.
// A zero value means the field was absent or invalid, except for the
// positions, which keep the FIT invalid value (use StartPosition and
// EndPosition).
type Split struct {
	// SplitType is the FIT split type name, e.g. "run_active", "ascent_split".
	SplitType string `json:"split_type,omitempty"`
	// StartTime is when the split started (UTC).
	StartTime time.Time `json:"start_time,omitzero"`
	// TotalElapsedTime is the elapsed time, in milliseconds.
	TotalElapsedTime uint32 `json:"total_elapsed_time,omitempty"`
	// TotalTimerTime is the timer time, in milliseconds.
	TotalTimerTime uint32 `json:"total_timer_time,omitempty"`
	// TotalMovingTime is the time spent moving, in milliseconds.
	TotalMovingTime uint32 `json:"total_moving_time,omitempty"`
	// TotalDistance is the split distance, in centimeters.
	TotalDistance uint32 `json:"total_distance,omitempty"`
	// AvgSpeed is the average speed, in millimeters per second.
	AvgSpeed uint32 `json:"avg_speed,omitempty"`
	// MaxSpeed is the maximum speed, in millimeters per second.
	MaxSpeed uint32 `json:"max_speed,omitempty"`
	// TotalAscent is the total ascent, in meters.
	TotalAscent uint16 `json:"total_ascent,omitempty"`
	// TotalDescent is the total descent, in meters.
	TotalDescent uint16 `json:"total_descent,omitempty"`
	// TotalCalories is the energy expenditure, in kilocalories.
	TotalCalories uint32 `json:"total_calories,omitempty"`
	// StartElevation is the starting elevation, raw FIT value with scale 5
	// and offset 500 (meters = raw/5 - 500); see StartElevationM.
	StartElevation uint32 `json:"start_elevation,omitempty"`
	// StartPositionLat is the start latitude, in semicircles; math.MaxInt32
	// means absent.
	StartPositionLat int32 `json:"start_position_lat"`
	// StartPositionLong is the start longitude, in semicircles;
	// math.MaxInt32 means absent.
	StartPositionLong int32 `json:"start_position_long"`
	// EndPositionLat is the end latitude, in semicircles; math.MaxInt32
	// means absent.
	EndPositionLat int32 `json:"end_position_lat"`
	// EndPositionLong is the end longitude, in semicircles; math.MaxInt32
	// means absent.
	EndPositionLong int32 `json:"end_position_long"`
}

// TimeInZone is a time_in_zone message: time spent in each zone for the
// session or lap it references, together with the zone boundaries used.
// A zero value means the field was absent or invalid, except ReferenceIndex.
type TimeInZone struct {
	// ReferenceMesg is the FIT message name this summary refers to, e.g.
	// "session" or "lap".
	ReferenceMesg string `json:"reference_mesg,omitempty"`
	// ReferenceIndex is the message_index of the referenced session or lap.
	// Because 0 is a valid index, an absent value keeps the FIT invalid
	// value 0xFFFF.
	ReferenceIndex uint16 `json:"reference_index"`
	// TimeInHrZone is the time per heart rate zone, in milliseconds.
	// Invalid elements are zero; nil when absent or when every element is
	// invalid.
	TimeInHrZone []uint32 `json:"time_in_hr_zone,omitempty"`
	// TimeInSpeedZone is the time per speed zone, in milliseconds.
	// Invalid elements are zero; nil when absent or when every element is
	// invalid.
	TimeInSpeedZone []uint32 `json:"time_in_speed_zone,omitempty"`
	// TimeInCadenceZone is the time per cadence zone, in milliseconds.
	// Invalid elements are zero; nil when absent or when every element is
	// invalid.
	TimeInCadenceZone []uint32 `json:"time_in_cadence_zone,omitempty"`
	// TimeInPowerZone is the time per power zone, in milliseconds.
	// Invalid elements are zero; nil when absent or when every element is
	// invalid.
	TimeInPowerZone []uint32 `json:"time_in_power_zone,omitempty"`
	// HrZoneHighBoundary holds the upper bound of each heart rate zone, in
	// beats per minute. Invalid elements are zero; nil when absent or when
	// every element is invalid. In JSON it is written as an array of
	// numbers, not as the base64 string encoding/json uses for []uint8.
	HrZoneHighBoundary []uint8 `json:"hr_zone_high_boundary,omitempty"`
	// PowerZoneHighBoundary holds the upper bound of each power zone, in
	// watts. Invalid elements are zero; nil when absent or when every
	// element is invalid.
	PowerZoneHighBoundary []uint16 `json:"power_zone_high_boundary,omitempty"`
	// MaxHeartRate is the maximum heart rate used for the zones, in bpm.
	MaxHeartRate uint8 `json:"max_heart_rate,omitempty"`
	// RestingHeartRate is the resting heart rate used for the zones, in bpm.
	RestingHeartRate uint8 `json:"resting_heart_rate,omitempty"`
	// ThresholdHeartRate is the lactate threshold heart rate used for the
	// zones, in bpm.
	ThresholdHeartRate uint8 `json:"threshold_heart_rate,omitempty"`
	// FunctionalThresholdPower is the FTP used for the zones, in watts.
	FunctionalThresholdPower uint16 `json:"functional_threshold_power,omitempty"`
}

// MarshalJSON writes HrZoneHighBoundary as an array of numbers.
// encoding/json would otherwise encode the []uint8 as a base64 string. The
// output still unmarshals into a TimeInZone with the standard decoder.
func (tz TimeInZone) MarshalJSON() ([]byte, error) {
	type plain TimeInZone // same fields, without this method
	var hb []uint16
	if tz.HrZoneHighBoundary != nil {
		hb = make([]uint16, len(tz.HrZoneHighBoundary))
		for i, v := range tz.HrZoneHighBoundary {
			hb[i] = uint16(v)
		}
	}
	return json.Marshal(struct {
		plain
		// HrZoneHighBoundary shadows the embedded []uint8 field.
		HrZoneHighBoundary []uint16 `json:"hr_zone_high_boundary,omitempty"`
	}{plain(tz), hb})
}

// Workout is a workout message: the header of a structured workout.
type Workout struct {
	// WktName is the workout name.
	WktName string `json:"wkt_name,omitempty"`
	// WktDescription is the workout description.
	WktDescription string `json:"wkt_description,omitempty"`
	// Sport is the FIT sport name, e.g. "running".
	Sport string `json:"sport,omitempty"`
	// SubSport is the FIT sub-sport name, e.g. "generic", "track".
	SubSport string `json:"sub_sport,omitempty"`
	// NumValidSteps is the number of workout steps.
	NumValidSteps uint16 `json:"num_valid_steps,omitempty"`
	// PoolLength is the swimming pool length, in centimeters.
	PoolLength uint16 `json:"pool_length,omitempty"`
}

// WorkoutStep is a workout_step message: one step of a structured workout.
// DurationValue and TargetValue are raw FIT values whose meaning depends on
// DurationType and TargetType; see the accessor methods.
type WorkoutStep struct {
	// WktStepName is the step name.
	WktStepName string `json:"wkt_step_name,omitempty"`
	// DurationType is the FIT step duration type name, e.g. "time",
	// "distance", "open", "repeat_until_steps_cmplt".
	DurationType string `json:"duration_type,omitempty"`
	// DurationValue is the raw duration: milliseconds for "time",
	// centimeters for "distance", calories for "calories", a step index for
	// "repeat_until_*" types, etc.
	DurationValue uint32 `json:"duration_value,omitempty"`
	// TargetType is the FIT step target type name, e.g. "heart_rate",
	// "speed", "power", "open".
	TargetType string `json:"target_type,omitempty"`
	// TargetValue is the raw target: a zone number, or 0 when the custom
	// range (CustomTargetValueLow/High) applies.
	TargetValue uint32 `json:"target_value,omitempty"`
	// CustomTargetValueLow is the low end of a custom target range. Units
	// depend on TargetType: mm/s for "speed"; for "heart_rate", 0-100 is a
	// percentage of maximum heart rate and values above 100 are bpm+100;
	// for "power", 0-1000 is a percentage of FTP and values above 1000 are
	// watts+1000; rpm for "cadence".
	CustomTargetValueLow uint32 `json:"custom_target_value_low,omitempty"`
	// CustomTargetValueHigh is the high end of a custom target range, in the
	// same units as CustomTargetValueLow.
	CustomTargetValueHigh uint32 `json:"custom_target_value_high,omitempty"`
	// Intensity is the FIT intensity name, e.g. "active", "rest", "warmup".
	Intensity string `json:"intensity,omitempty"`
}

// ZonesTarget is the zones_target message: the athlete's zone anchors.
type ZonesTarget struct {
	// MaxHeartRate is the maximum heart rate, in beats per minute.
	MaxHeartRate uint8 `json:"max_heart_rate,omitempty"`
	// ThresholdHeartRate is the lactate threshold heart rate, in bpm.
	ThresholdHeartRate uint8 `json:"threshold_heart_rate,omitempty"`
	// FunctionalThresholdPower is the functional threshold power, in watts.
	FunctionalThresholdPower uint16 `json:"functional_threshold_power,omitempty"`
	// HrCalcType is the FIT heart rate zone calculation name, e.g. "custom",
	// "percent_max_hr", "percent_hrr", "percent_lthr".
	HrCalcType string `json:"hr_calc_type,omitempty"`
	// PwrCalcType is the FIT power zone calculation name, e.g. "custom",
	// "percent_ftp".
	PwrCalcType string `json:"pwr_calc_type,omitempty"`
}

// HRZone is an hr_zone message: the upper bound of one heart rate zone.
type HRZone struct {
	// MessageIndex is the zone's index (0 = the lowest zone). Because 0 is a
	// valid index, an absent value keeps the FIT invalid value 0xFFFF.
	MessageIndex uint16 `json:"message_index"`
	// Name is the zone name, if any.
	Name string `json:"name,omitempty"`
	// HighBpm is the zone's inclusive upper bound, in beats per minute.
	HighBpm uint8 `json:"high_bpm,omitempty"`
}

// PowerZone is a power_zone message: the upper bound of one power zone.
type PowerZone struct {
	// MessageIndex is the zone's index (0 = the lowest zone). Because 0 is a
	// valid index, an absent value keeps the FIT invalid value 0xFFFF.
	MessageIndex uint16 `json:"message_index"`
	// Name is the zone name, if any.
	Name string `json:"name,omitempty"`
	// HighValue is the zone's inclusive upper bound, in watts.
	HighValue uint16 `json:"high_value,omitempty"`
}

// DeveloperFieldDescription is a field_description message: it declares a
// developer (Connect IQ) field. A field is identified by the pair
// (DeveloperDataIndex, FieldDefinitionNumber).
type DeveloperFieldDescription struct {
	// DeveloperDataIndex identifies the developer application within the
	// FIT sequence (it matches a developer_data_id message).
	DeveloperDataIndex uint8 `json:"developer_data_index"`
	// FieldDefinitionNumber is the field number within that application.
	FieldDefinitionNumber uint8 `json:"field_definition_number"`
	// FieldName is the field's name, e.g. "Power".
	FieldName string `json:"field_name,omitempty"`
	// Units is the field's unit label, e.g. "Watts".
	Units string `json:"units,omitempty"`
	// NativeMesgNum is the FIT message name the field belongs to, e.g.
	// "record" or "session"; "" when not declared.
	NativeMesgNum string `json:"native_mesg_num,omitempty"`
}

// DeveloperFieldValue is one developer field value carried by a message.
type DeveloperFieldValue struct {
	// DeveloperDataIndex identifies the developer application.
	DeveloperDataIndex uint8 `json:"developer_data_index"`
	// Num is the field definition number within that application.
	Num uint8 `json:"num"`
	// Name is the field name from the matching field description.
	Name string `json:"name,omitempty"`
	// Units is the unit label from the matching field description.
	Units string `json:"units,omitempty"`
	// Value is the raw value as declared by the field description's base
	// type: one of int8, uint8, int16, uint16, int32, uint32, int64, uint64,
	// float32, float64, bool, string, or a slice of one of those, except
	// that a uint8 (or byte) array is returned as []uint16 so that it is
	// written to JSON as numbers rather than as a base64 string. Any scale
	// or offset declared by the field description is not applied. A scalar
	// equal to the base type's invalid sentinel is dropped. An array is
	// dropped only when all its elements are invalid; otherwise it is
	// returned as-is, and individual elements may still hold the sentinel.
	// Non-finite floats, scalar or in an array, are dropped.
	Value any `json:"value"`
}

// Stryd holds the Stryd foot pod metrics of one record, read from the
// developer data index that declares the Stryd fields and matched by field
// name. Integer and floating-point developer values are both accepted;
// integer metrics are rounded. Zero means absent (invalid, zero or negative
// values are dropped); no upper plausibility limit is applied.
type Stryd struct {
	// Power is the running power, in watts.
	Power uint16 `json:"power,omitempty"`
	// GroundTime is the ground contact time, in milliseconds.
	GroundTime uint16 `json:"ground_time,omitempty"`
	// VerticalOscillation is the vertical oscillation, in centimeters.
	VerticalOscillation float64 `json:"vertical_oscillation,omitempty"`
	// FormPower is the form power, in watts.
	FormPower uint16 `json:"form_power,omitempty"`
	// LegSpringStiffness is the leg spring stiffness, in kN/m.
	LegSpringStiffness float64 `json:"leg_spring_stiffness,omitempty"`
	// AirPower is the air power, in watts.
	AirPower uint16 `json:"air_power,omitempty"`
}
