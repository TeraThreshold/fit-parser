package fitgen

import (
	"time"

	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/proto"
)

// Degrees converts degrees to FIT semicircles.
func Degrees(deg float64) int32 {
	return int32(deg * (1 << 31) / 180)
}

// Altitude converts meters to the raw FIT altitude (scale 5, offset 500).
func Altitude(m float64) uint32 {
	return uint32((m + 500) * 5)
}

// Run describes a synthetic running activity for AddRun. All values are
// made up; nothing is taken from a real recording.
type Run struct {
	// Start is the activity start time; it should be after 2000-01-01.
	Start time.Time
	// Seconds is the number of 1 Hz records (and the activity duration).
	Seconds int
	// UTCOffset is written as activity.local_timestamp - timestamp.
	UTCOffset time.Duration
	// Record, when non-nil, may adjust record i after the defaults are set
	// (set a field to its invalid sentinel to drop it).
	Record func(i int, r *mesgdef.Record)
	// RecordDev, when non-nil, returns developer values for record i.
	RecordDev func(i int) []proto.DeveloperField
}

// Record defaults written by AddRun. Record i has speed RunSpeed, distance
// i*RunSpeed/10 cm, heart rate RunBaseHR + i%20 and position RunLat/RunLon
// moving north by i*10 semicircles.
const (
	RunSpeed      uint32  = 3000  // mm/s
	RunBaseHR     uint8   = 140   // bpm
	RunCadence    uint8   = 85    // rpm
	RunPower      uint16  = 250   // W
	RunAltitudeM  float64 = 100   // m
	RunStanceTime uint16  = 2500  // 0.1 ms
	RunVertOsc    uint16  = 850   // 0.1 mm
	RunStepLength uint16  = 10500 // 0.1 mm
	RunVertRatio  uint16  = 810   // 0.01 %
	RunGCTBalance uint16  = 5010  // 0.01 %
	RunLat        float64 = 10    // degrees, synthetic
	RunLon        float64 = 20    // degrees, synthetic
)

// AddRun appends a complete synthetic running activity to the current
// sequence: file_id, device_info (creator), timer start event, one record
// per second, timer stop event, one lap, one session and an activity
// message with a local timestamp.
func (b *Builder) AddRun(r Run) *Builder {
	start := r.Start
	end := start.Add(time.Duration(r.Seconds) * time.Second)
	elapsedMs := uint32(r.Seconds) * 1000
	distCm := uint32(r.Seconds) * RunSpeed / 10

	b.AddFileID(func(m *mesgdef.FileId) {
		m.Type = typedef.FileActivity
		m.Manufacturer = typedef.ManufacturerGarmin
		m.Product = uint16(typedef.GarminProductFr965)
		m.SerialNumber = 1000000001
		m.TimeCreated = start
	})
	b.AddDeviceInfo(func(m *mesgdef.DeviceInfo) {
		m.Timestamp = start
		m.DeviceIndex = typedef.DeviceIndexCreator
		m.Manufacturer = typedef.ManufacturerGarmin
		m.Product = uint16(typedef.GarminProductFr965)
		m.SerialNumber = 1000000001
		m.SoftwareVersion = 1234
		m.BatteryLevel = 90
	})
	b.AddEvent(func(m *mesgdef.Event) {
		m.Timestamp = start
		m.Event = typedef.EventTimer
		m.EventType = typedef.EventTypeStart
	})
	for i := 0; i < r.Seconds; i++ {
		var dev []proto.DeveloperField
		if r.RecordDev != nil {
			dev = r.RecordDev(i)
		}
		b.AddRecord(func(m *mesgdef.Record) {
			m.Timestamp = start.Add(time.Duration(i) * time.Second)
			m.PositionLat = Degrees(RunLat) + int32(i)*10
			m.PositionLong = Degrees(RunLon)
			m.HeartRate = RunBaseHR + uint8(i%20)
			m.Cadence = RunCadence
			m.Power = RunPower
			m.EnhancedSpeed = RunSpeed
			m.Distance = uint32(i) * RunSpeed / 10
			m.EnhancedAltitude = Altitude(RunAltitudeM)
			m.Grade = 0
			m.Temperature = 21
			m.StanceTime = RunStanceTime
			m.VerticalOscillation = RunVertOsc
			m.StepLength = RunStepLength
			m.VerticalRatio = RunVertRatio
			m.StanceTimeBalance = RunGCTBalance
			m.EnhancedRespirationRate = 3000
			if r.Record != nil {
				r.Record(i, m)
			}
		}, dev...)
	}
	b.AddEvent(func(m *mesgdef.Event) {
		m.Timestamp = end
		m.Event = typedef.EventTimer
		m.EventType = typedef.EventTypeStopAll
	})
	b.AddLap(func(m *mesgdef.Lap) {
		m.Timestamp = end
		m.StartTime = start
		m.TotalElapsedTime = elapsedMs
		m.TotalTimerTime = elapsedMs
		m.TotalDistance = distCm
		m.EnhancedAvgSpeed = RunSpeed
		m.EnhancedMaxSpeed = RunSpeed
		m.AvgHeartRate = RunBaseHR + 10
		m.MaxHeartRate = RunBaseHR + 19
		m.AvgCadence = RunCadence
		m.MaxCadence = RunCadence
		m.AvgPower = RunPower
		m.MaxPower = RunPower
		m.LapTrigger = typedef.LapTriggerSessionEnd
		m.Sport = typedef.SportRunning
	})
	b.AddSession(func(m *mesgdef.Session) {
		m.Timestamp = end
		m.StartTime = start
		m.Sport = typedef.SportRunning
		m.SubSport = typedef.SubSportGeneric
		m.TotalElapsedTime = elapsedMs
		m.TotalTimerTime = elapsedMs
		m.TotalMovingTime = elapsedMs
		m.TotalDistance = distCm
		m.EnhancedAvgSpeed = RunSpeed
		m.EnhancedMaxSpeed = RunSpeed
		m.AvgHeartRate = RunBaseHR + 10
		m.MaxHeartRate = RunBaseHR + 19
		m.MinHeartRate = RunBaseHR
		m.AvgCadence = RunCadence
		m.MaxCadence = RunCadence
		m.AvgPower = RunPower
		m.MaxPower = RunPower
		m.TotalAscent = 0
		m.TotalDescent = 0
		m.TotalCalories = uint16(r.Seconds / 6)
		m.NumLaps = 1
	})
	return b.AddActivity(func(m *mesgdef.Activity) {
		m.Timestamp = end
		m.LocalTimestamp = end.Add(r.UTCOffset)
		m.TotalTimerTime = elapsedMs
		m.NumSessions = 1
		m.Type = typedef.ActivityManual
		m.Event = typedef.EventActivity
		m.EventType = typedef.EventTypeStop
	})
}
