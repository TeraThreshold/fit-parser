// Package fitgen builds synthetic FIT files for tests.
//
// It wraps the upstream encoder and message definitions so tests can write
// FIT files with exactly the values they need. Every Add* method starts from
// a message whose fields all hold the FIT invalid sentinel, lets the caller
// set fields through a callback, and appends the result; fields left
// untouched are omitted from the encoded message, which a decoder reads as
// the invalid sentinel. SetField writes an explicit value (including a
// sentinel) into a raw message when a test needs the bytes to be present.
//
//	data := fitgen.New().
//		AddFileID(func(m *mesgdef.FileId) { m.Type = typedef.FileActivity }).
//		AddRecord(func(r *mesgdef.Record) { r.HeartRate = 150 }).
//		MustBytes()
//
// NewSequence starts another FIT sequence; Bytes concatenates all sequences
// into one chained FIT file.
package fitgen

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/muktihari/fit/encoder"
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/factory"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/proto"
)

// Builder accumulates messages into one or more FIT sequences. The zero
// value is not usable; call New.
type Builder struct {
	seqs    [][]proto.Message
	lenient bool
}

// New returns a Builder with one empty FIT sequence.
func New() *Builder {
	return &Builder{seqs: [][]proto.Message{nil}}
}

// NewSequence starts a new FIT sequence. Messages added afterwards go into
// it, and Bytes emits it after the previous ones (a chained FIT file).
func (b *Builder) NewSequence() *Builder {
	b.seqs = append(b.seqs, nil)
	return b
}

// Lenient disables the encoder's message validation, so tests can write
// developer fields without a matching developer_data_id or
// field_description, or values whose type does not match the profile.
func (b *Builder) Lenient() *Builder {
	b.lenient = true
	return b
}

// Add appends raw messages to the current sequence.
func (b *Builder) Add(msgs ...proto.Message) *Builder {
	last := len(b.seqs) - 1
	b.seqs[last] = append(b.seqs[last], msgs...)
	return b
}

// Messages returns the messages of every sequence, in order. The slices are
// the builder's own; callers may modify them before calling Bytes.
func (b *Builder) Messages() [][]proto.Message { return b.seqs }

// Bytes encodes all sequences (FIT protocol 2.0, little-endian) and returns
// the concatenated file.
func (b *Builder) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	opts := []encoder.Option{encoder.WithProtocolVersion(proto.V2)}
	if b.lenient {
		opts = append(opts, encoder.WithMessageValidator(noValidation{}))
	}
	enc := encoder.New(&buf, opts...)
	for i, seq := range b.seqs {
		// The encoder rewrites message headers and field order in place;
		// work on a copy so Bytes can be called repeatedly.
		msgs := make([]proto.Message, len(seq))
		for j := range seq {
			msgs[j] = seq[j]
			msgs[j].Fields = slices.Clone(seq[j].Fields)
			msgs[j].DeveloperFields = slices.Clone(seq[j].DeveloperFields)
		}
		if err := enc.Encode(&proto.FIT{Messages: msgs}); err != nil {
			return nil, fmt.Errorf("fitgen: sequence %d: %w", i, err)
		}
	}
	return buf.Bytes(), nil
}

// MustBytes is like Bytes but panics on error. It is meant for tests, where
// an encoding error is a bug in the test itself.
func (b *Builder) MustBytes() []byte {
	data, err := b.Bytes()
	if err != nil {
		panic(err)
	}
	return data
}

// noValidation is a message validator that accepts everything.
type noValidation struct{}

func (noValidation) Validate(*proto.Message) error { return nil }
func (noValidation) Reset()                        {}

// SetField sets field num of m to v, replacing an existing field with the
// same number. Use it to write explicit values the typed Add* methods would
// omit, such as a field holding its invalid sentinel. v must match the
// field's profile base type (e.g. proto.Uint16 for record power) unless the
// Builder is Lenient.
func SetField(m *proto.Message, num byte, v proto.Value) {
	for i := range m.Fields {
		if m.Fields[i].Num == num {
			m.Fields[i].Value = v
			return
		}
	}
	f := factory.CreateField(m.Num, num)
	f.Value = v
	m.Fields = append(m.Fields, f)
}

// Dev returns a developer field value for developer data index idx and field
// number num. v must be a FIT-representable Go value (int8, uint8, int16,
// uint16, int32, uint32, int64, uint64, float32, float64, string or a slice
// of those) matching the base type declared by the field description.
func Dev(idx, num uint8, v any) proto.DeveloperField {
	return proto.DeveloperField{DeveloperDataIndex: idx, Num: num, Value: proto.Any(v)}
}

// toMesger is implemented by every upstream message definition.
type toMesger interface {
	ToMesg(options *mesgdef.Options) proto.Message
}

// add applies fn to m (when non-nil), appends dev to the encoded message and
// adds it to the current sequence.
func add[T toMesger](b *Builder, m T, fn func(T), dev []proto.DeveloperField) *Builder {
	if fn != nil {
		fn(m)
	}
	msg := m.ToMesg(nil)
	if len(dev) > 0 {
		msg.DeveloperFields = append(msg.DeveloperFields, dev...)
	}
	return b.Add(msg)
}

// AddFileID adds a file_id message.
func (b *Builder) AddFileID(fn func(*mesgdef.FileId)) *Builder {
	return add(b, mesgdef.NewFileId(nil), fn, nil)
}

// AddDeviceInfo adds a device_info message.
func (b *Builder) AddDeviceInfo(fn func(*mesgdef.DeviceInfo)) *Builder {
	return add(b, mesgdef.NewDeviceInfo(nil), fn, nil)
}

// AddZonesTarget adds a zones_target message.
func (b *Builder) AddZonesTarget(fn func(*mesgdef.ZonesTarget)) *Builder {
	return add(b, mesgdef.NewZonesTarget(nil), fn, nil)
}

// AddHRZone adds an hr_zone message.
func (b *Builder) AddHRZone(fn func(*mesgdef.HrZone)) *Builder {
	return add(b, mesgdef.NewHrZone(nil), fn, nil)
}

// AddPowerZone adds a power_zone message.
func (b *Builder) AddPowerZone(fn func(*mesgdef.PowerZone)) *Builder {
	return add(b, mesgdef.NewPowerZone(nil), fn, nil)
}

// AddEvent adds an event message.
func (b *Builder) AddEvent(fn func(*mesgdef.Event)) *Builder {
	return add(b, mesgdef.NewEvent(nil), fn, nil)
}

// AddRecord adds a record message carrying the given developer field values.
func (b *Builder) AddRecord(fn func(*mesgdef.Record), dev ...proto.DeveloperField) *Builder {
	return add(b, mesgdef.NewRecord(nil), fn, dev)
}

// AddLap adds a lap message.
func (b *Builder) AddLap(fn func(*mesgdef.Lap)) *Builder {
	return add(b, mesgdef.NewLap(nil), fn, nil)
}

// AddSession adds a session message.
func (b *Builder) AddSession(fn func(*mesgdef.Session)) *Builder {
	return add(b, mesgdef.NewSession(nil), fn, nil)
}

// AddActivity adds an activity message.
func (b *Builder) AddActivity(fn func(*mesgdef.Activity)) *Builder {
	return add(b, mesgdef.NewActivity(nil), fn, nil)
}

// AddHRV adds an hrv message holding the given R-R intervals (raw FIT
// value: milliseconds, 0xFFFF = invalid).
func (b *Builder) AddHRV(ms ...uint16) *Builder {
	return add(b, mesgdef.NewHrv(nil), func(m *mesgdef.Hrv) { m.Time = ms }, nil)
}

// AddSplit adds a split message.
func (b *Builder) AddSplit(fn func(*mesgdef.Split)) *Builder {
	return add(b, mesgdef.NewSplit(nil), fn, nil)
}

// AddTimeInZone adds a time_in_zone message.
func (b *Builder) AddTimeInZone(fn func(*mesgdef.TimeInZone)) *Builder {
	return add(b, mesgdef.NewTimeInZone(nil), fn, nil)
}

// AddWorkout adds a workout message.
func (b *Builder) AddWorkout(fn func(*mesgdef.Workout)) *Builder {
	return add(b, mesgdef.NewWorkout(nil), fn, nil)
}

// AddWorkoutStep adds a workout_step message.
func (b *Builder) AddWorkoutStep(fn func(*mesgdef.WorkoutStep)) *Builder {
	return add(b, mesgdef.NewWorkoutStep(nil), fn, nil)
}

// AddDeveloperDataID adds a developer_data_id message for developer data
// index idx. The application id defaults to 16 synthetic bytes derived from
// idx; fn may override any field.
func (b *Builder) AddDeveloperDataID(idx uint8, fn func(*mesgdef.DeveloperDataId)) *Builder {
	m := mesgdef.NewDeveloperDataId(nil)
	m.DeveloperDataIndex = idx
	appID := make([]byte, 16)
	for i := range appID {
		appID[i] = idx + byte(i)
	}
	m.ApplicationId = appID
	m.ApplicationVersion = 1
	return add(b, m, fn, nil)
}

// AddFieldDescription adds a field_description message declaring field num
// of developer data index idx with the given name, units ("" to omit) and
// base type. native_mesg_num is left unset; fn may override any field.
func (b *Builder) AddFieldDescription(idx, num uint8, name, units string, bt basetype.BaseType, fn func(*mesgdef.FieldDescription)) *Builder {
	m := mesgdef.NewFieldDescription(nil)
	m.DeveloperDataIndex = idx
	m.FieldDefinitionNumber = num
	m.FitBaseTypeId = bt
	m.FieldName = []string{name}
	if units != "" {
		m.Units = []string{units}
	}
	return add(b, m, fn, nil)
}

// Stryd developer field numbers used by AddStrydDescriptions and
// StrydValues. They mirror the layout of the Stryd Connect IQ app.
const (
	StrydPowerNum               uint8 = 0
	StrydGroundTimeNum          uint8 = 3
	StrydVerticalOscillationNum uint8 = 4
	StrydFormPowerNum           uint8 = 8
	StrydLegSpringStiffnessNum  uint8 = 9
	StrydAirPowerNum            uint8 = 11
)

// AddStrydDescriptions adds a developer_data_id for idx and the Stryd field
// descriptions (Power, Ground Time, Vertical Oscillation, Form Power, Leg
// Spring Stiffness, Air Power), all declared as record fields.
func (b *Builder) AddStrydDescriptions(idx uint8) *Builder {
	native := func(m *mesgdef.FieldDescription) { m.NativeMesgNum = typedef.MesgNumRecord }
	return b.AddDeveloperDataID(idx, nil).
		AddFieldDescription(idx, StrydPowerNum, "Power", "Watts", basetype.Uint16, native).
		AddFieldDescription(idx, StrydGroundTimeNum, "Ground Time", "Milliseconds", basetype.Uint16, native).
		AddFieldDescription(idx, StrydVerticalOscillationNum, "Vertical Oscillation", "Centimeters", basetype.Float32, native).
		AddFieldDescription(idx, StrydFormPowerNum, "Form Power", "Watts", basetype.Uint16, native).
		AddFieldDescription(idx, StrydLegSpringStiffnessNum, "Leg Spring Stiffness", "KN/m", basetype.Float32, native).
		AddFieldDescription(idx, StrydAirPowerNum, "Air Power", "Watts", basetype.Uint16, native)
}

// StrydSample holds one record's Stryd values for StrydValues.
type StrydSample struct {
	Power               uint16  // W
	GroundTime          uint16  // ms
	VerticalOscillation float32 // cm
	FormPower           uint16  // W
	LegSpringStiffness  float32 // kN/m
	AirPower            uint16  // W
}

// StrydValues returns the developer field values of s for developer data
// index idx, matching the descriptions written by AddStrydDescriptions.
func StrydValues(idx uint8, s StrydSample) []proto.DeveloperField {
	return []proto.DeveloperField{
		Dev(idx, StrydPowerNum, s.Power),
		Dev(idx, StrydGroundTimeNum, s.GroundTime),
		Dev(idx, StrydVerticalOscillationNum, s.VerticalOscillation),
		Dev(idx, StrydFormPowerNum, s.FormPower),
		Dev(idx, StrydLegSpringStiffnessNum, s.LegSpringStiffness),
		Dev(idx, StrydAirPowerNum, s.AirPower),
	}
}
