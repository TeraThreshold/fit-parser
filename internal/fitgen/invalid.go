package fitgen

import (
	"github.com/muktihari/fit/profile/basetype"
	"github.com/muktihari/fit/profile/factory"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/proto"
)

// Invalid returns a message of type num in which every field defined by the
// FIT profile is present and holds its base type's invalid sentinel. Array
// fields hold two invalid elements. Tests use it to check that explicit
// sentinels in the encoded bytes (not only omitted fields) are treated as
// absent values.
func Invalid(num typedef.MesgNum) proto.Message {
	m := factory.CreateMesg(num)
	fields := m.Fields[:0]
	for _, f := range m.Fields {
		v, ok := invalidValue(f.BaseType, f.Array)
		if !ok {
			continue
		}
		f.Value = v
		fields = append(fields, f)
	}
	m.Fields = fields
	return m
}

// invalidValue returns the invalid sentinel of bt as a proto.Value, or a
// two-element slice of sentinels when array is true.
func invalidValue(bt basetype.BaseType, array bool) (proto.Value, bool) {
	if !array {
		switch bt {
		case basetype.String:
			return proto.String(basetype.StringInvalid), true
		default:
			inv := bt.Invalid()
			if inv == nil {
				return proto.Value{}, false
			}
			return proto.Any(inv), true
		}
	}
	switch bt {
	case basetype.Enum, basetype.Uint8, basetype.Uint8z, basetype.Byte:
		inv := bt.Invalid().(uint8)
		return proto.SliceUint8([]uint8{inv, inv}), true
	case basetype.Sint8:
		return proto.SliceInt8([]int8{basetype.Sint8Invalid, basetype.Sint8Invalid}), true
	case basetype.Uint16, basetype.Uint16z:
		inv := bt.Invalid().(uint16)
		return proto.SliceUint16([]uint16{inv, inv}), true
	case basetype.Sint16:
		return proto.SliceInt16([]int16{basetype.Sint16Invalid, basetype.Sint16Invalid}), true
	case basetype.Uint32, basetype.Uint32z:
		inv := bt.Invalid().(uint32)
		return proto.SliceUint32([]uint32{inv, inv}), true
	case basetype.Sint32:
		return proto.SliceInt32([]int32{basetype.Sint32Invalid, basetype.Sint32Invalid}), true
	case basetype.String:
		return proto.SliceString([]string{basetype.StringInvalid, basetype.StringInvalid}), true
	case basetype.Float32:
		inv := bt.Invalid().(float32)
		return proto.SliceFloat32([]float32{inv, inv}), true
	case basetype.Float64:
		inv := bt.Invalid().(float64)
		return proto.SliceFloat64([]float64{inv, inv}), true
	case basetype.Sint64:
		return proto.SliceInt64([]int64{basetype.Sint64Invalid, basetype.Sint64Invalid}), true
	case basetype.Uint64, basetype.Uint64z:
		inv := bt.Invalid().(uint64)
		return proto.SliceUint64([]uint64{inv, inv}), true
	}
	return proto.Value{}, false
}
