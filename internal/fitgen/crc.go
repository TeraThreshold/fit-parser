package fitgen

import (
	"encoding/binary"

	"github.com/muktihari/fit/kit/hash/crc16"
)

// FixCRC returns a copy of data in which the header CRC and the file CRC of
// every FIT sequence are recomputed. Sequences are located through their
// header's data size; FixCRC stops at the first header it cannot follow and
// leaves the remaining bytes unchanged. Tests use it to push corrupted
// message bytes past the CRC check so that the message parser itself sees
// them.
func FixCRC(data []byte) []byte {
	out := append([]byte(nil), data...)
	for off := 0; off < len(out); {
		hs := int(out[off])
		if hs < 12 || off+hs > len(out) {
			break
		}
		size := int(binary.LittleEndian.Uint32(out[off+4 : off+8]))
		end := off + hs + size
		if size < 0 || end < off || end+2 > len(out) {
			break
		}
		if hs >= 14 {
			binary.LittleEndian.PutUint16(out[off+12:off+14], checksum(out[off:off+12]))
		}
		binary.LittleEndian.PutUint16(out[end:end+2], checksum(out[off:end]))
		off = end + 2
	}
	return out
}

func checksum(b []byte) uint16 {
	h := crc16.New()
	h.Write(b)
	return h.Sum16()
}
