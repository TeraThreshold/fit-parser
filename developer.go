package fitparser

import (
	"math"

	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/proto"
)

// devKey identifies a developer field within one FIT sequence.
type devKey struct {
	index uint8 // developer_data_index
	num   uint8 // field_definition_number
}

// devField is a field_description as needed to interpret values.
type devField struct {
	name     string
	units    string
	baseType basetype.BaseType
}

// devContext holds the developer field descriptions of one FIT sequence.
// Developer data indexes are only meaningful within the sequence that
// declares them, so a new context is built for every chained sequence.
//
// Fields are keyed by (developer_data_index, field_definition_number): two
// Connect IQ apps may both declare field #0, and only the pair tells them
// apart.
type devContext struct {
	fields map[devKey]devField
	// stryd marks the developer data indexes that declare a Stryd-only
	// field ("Leg Spring Stiffness" or "Form Power"). Stryd values are only
	// read from these indexes.
	stryd map[uint8]bool
}

func newDevContext() *devContext {
	return &devContext{
		fields: make(map[devKey]devField),
		stryd:  make(map[uint8]bool),
	}
}

// addDescription records a field_description message and returns its public
// form. When several descriptions share a (developer_data_index,
// field_definition_number) key, the first one is kept for interpreting
// values, because the upstream decoder decodes developer values with the
// first matching description. Every description is still returned so that
// File.DeveloperFields lists them all.
func (c *devContext) addDescription(fd *mesgdef.FieldDescription) DeveloperFieldDescription {
	d := DeveloperFieldDescription{
		DeveloperDataIndex:    fd.DeveloperDataIndex,
		FieldDefinitionNumber: fd.FieldDefinitionNumber,
		FieldName:             firstString(fd.FieldName),
		Units:                 firstString(fd.Units),
		NativeMesgNum:         mesgNumName(fd.NativeMesgNum),
	}
	k := devKey{fd.DeveloperDataIndex, fd.FieldDefinitionNumber}
	if _, dup := c.fields[k]; !dup {
		c.fields[k] = devField{
			name:     d.FieldName,
			units:    d.Units,
			baseType: fd.FitBaseTypeId,
		}
	}
	switch d.FieldName {
	case strydLegSpringStiffness, strydFormPower:
		c.stryd[fd.DeveloperDataIndex] = true
	}
	return d
}

// hasStryd reports whether any developer data index of the sequence
// declares Stryd fields.
func (c *devContext) hasStryd() bool { return len(c.stryd) > 0 }

// recordFields converts a record's developer fields into public values and
// extracts Stryd metrics from the Stryd developer data index.
func (c *devContext) recordFields(dfs []proto.DeveloperField) ([]DeveloperFieldValue, *Stryd) {
	if len(dfs) == 0 {
		return nil, nil
	}
	var (
		out   []DeveloperFieldValue
		stryd Stryd
		found bool
	)
	for i := range dfs {
		df := &dfs[i]
		desc, ok := c.fields[devKey{df.DeveloperDataIndex, df.Num}]
		if !ok {
			// The upstream decoder only keeps developer fields it could
			// interpret through a field_description, so this is defensive.
			continue
		}
		if v, ok := publicValue(df.Value, desc.baseType); ok {
			out = append(out, DeveloperFieldValue{
				DeveloperDataIndex: df.DeveloperDataIndex,
				Num:                df.Num,
				Name:               desc.name,
				Units:              desc.units,
				Value:              v,
			})
		}
		if c.stryd[df.DeveloperDataIndex] && applyStryd(&stryd, desc.name, df.Value, desc.baseType) {
			found = true
		}
	}
	if !found {
		return out, nil
	}
	return out, &stryd
}

// Stryd developer field names.
const (
	strydPower               = "Power"
	strydGroundTime          = "Ground Time"
	strydVerticalOscillation = "Vertical Oscillation"
	strydFormPower           = "Form Power"
	strydLegSpringStiffness  = "Leg Spring Stiffness"
	strydAirPower            = "Air Power"
)

// applyStryd stores a Stryd developer value into s by field name. It
// returns true when the value was valid and positive and was stored.
func applyStryd(s *Stryd, name string, v proto.Value, bt basetype.BaseType) bool {
	f, ok := numericValue(v, bt)
	if !ok || f <= 0 {
		return false
	}
	switch name {
	case strydPower:
		return setUint16(&s.Power, f)
	case strydGroundTime:
		return setUint16(&s.GroundTime, f)
	case strydVerticalOscillation:
		s.VerticalOscillation = f
		return true
	case strydFormPower:
		return setUint16(&s.FormPower, f)
	case strydLegSpringStiffness:
		s.LegSpringStiffness = f
		return true
	case strydAirPower:
		return setUint16(&s.AirPower, f)
	}
	return false
}

// setUint16 rounds f into *dst when it fits a non-sentinel uint16.
func setUint16(dst *uint16, f float64) bool {
	r := math.Round(f)
	if r <= 0 || r >= math.MaxUint16 {
		return false
	}
	*dst = uint16(r)
	return true
}

// numericValue returns a scalar numeric developer value as float64. ok is
// false for non-numeric values, the base type's invalid sentinel and
// non-finite floats.
func numericValue(v proto.Value, bt basetype.BaseType) (float64, bool) {
	if !v.Valid(bt) {
		return 0, false
	}
	var f float64
	switch v.Type() {
	case proto.TypeInt8:
		f = float64(v.Int8())
	case proto.TypeUint8:
		f = float64(v.Uint8())
	case proto.TypeInt16:
		f = float64(v.Int16())
	case proto.TypeUint16:
		f = float64(v.Uint16())
	case proto.TypeInt32:
		f = float64(v.Int32())
	case proto.TypeUint32:
		f = float64(v.Uint32())
	case proto.TypeInt64:
		f = float64(v.Int64())
	case proto.TypeUint64:
		f = float64(v.Uint64())
	case proto.TypeFloat32:
		f = float64(v.Float32())
	case proto.TypeFloat64:
		f = v.Float64()
	default:
		return 0, false
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// publicValue converts a developer value into a plain Go value that does not
// expose upstream types. ok is false for invalid values (the base type's
// sentinel, empty strings) and for floats that are not finite, which cannot
// be represented in JSON.
func publicValue(v proto.Value, bt basetype.BaseType) (any, bool) {
	if !v.Valid(bt) {
		return nil, false
	}
	switch v.Type() {
	case proto.TypeBool:
		return v.Bool() == typedef.BoolTrue, true
	case proto.TypeSliceBool:
		src := v.SliceBool()
		out := make([]bool, len(src))
		for i, b := range src {
			out[i] = b == typedef.BoolTrue
		}
		return out, true
	case proto.TypeFloat32:
		f := v.Float32()
		if !finite(float64(f)) {
			return nil, false
		}
		return f, true
	case proto.TypeFloat64:
		f := v.Float64()
		if !finite(f) {
			return nil, false
		}
		return f, true
	case proto.TypeSliceUint8:
		// encoding/json writes []uint8 as a base64 string; widen it so a
		// uint8 or byte array is written as numbers like every other array.
		src := v.SliceUint8()
		out := make([]uint16, len(src))
		for i, b := range src {
			out[i] = uint16(b)
		}
		return out, true
	case proto.TypeSliceFloat32:
		for _, f := range v.SliceFloat32() {
			if !finite(float64(f)) {
				return nil, false
			}
		}
	case proto.TypeSliceFloat64:
		for _, f := range v.SliceFloat64() {
			if !finite(f) {
				return nil, false
			}
		}
	}
	return v.Any(), true
}

func finite(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) }

func firstString(s []string) string {
	for _, v := range s {
		if v != "" && v != basetype.StringInvalid && v != "\x00" {
			return v
		}
	}
	return ""
}
