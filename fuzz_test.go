package fitparser_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/proto"
)

// FuzzDecode checks that Decode never panics, returns either a file or an
// error, and that every decoded file can be marshaled to JSON. Each input
// is also decoded with its CRCs recomputed, so mutations reach the message
// parser instead of stopping at the checksum. The seed corpus is synthetic
// fitgen output only.
func FuzzDecode(f *testing.F) {
	f.Add(sampleFile(f))
	f.Add(seedChained(f))
	f.Add(seedAllInvalid(f))
	f.Add(seedEverything(f))
	f.Add(fitgen.New().AddStrydDescriptions(2).AddRun(fitgen.Run{
		Start: t0, Seconds: 3, UTCOffset: 5 * time.Hour,
		RecordDev: func(i int) []proto.DeveloperField {
			return fitgen.StrydValues(2, fitgen.StrydSample{Power: 300, LegSpringStiffness: 11})
		},
	}).MustBytes())
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, in := range [][]byte{data, fitgen.FixCRC(data)} {
			file, err := fitparser.Decode(bytes.NewReader(in))
			if err != nil {
				if file != nil {
					t.Fatalf("Decode returned a file and error %v", err)
				}
				continue
			}
			if file == nil {
				t.Fatal("Decode returned nil file and nil error")
			}
			if _, err := json.Marshal(file); err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			_ = file.Location()
			for i := range file.Records {
				r := &file.Records[i]
				r.Position()
				r.TemperatureC()
				r.AltitudeM()
				r.SpeedMps()
			}
		}
	})
}
