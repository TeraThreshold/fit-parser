package fitparser_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/proto"
)

// sampleFile is a small synthetic activity with developer fields.
func sampleFile(t testing.TB) []byte {
	t.Helper()
	data, err := fitgen.New().
		AddStrydDescriptions(0).
		AddRun(fitgen.Run{
			Start: t0, Seconds: 8, UTCOffset: time.Hour,
			RecordDev: func(i int) []proto.DeveloperField {
				return fitgen.StrydValues(0, fitgen.StrydSample{Power: uint16(200 + i), GroundTime: 250})
			},
		}).
		AddHRV(800, 810).
		Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// mustNotDecode asserts that Decode fails cleanly on data.
func mustNotDecode(t *testing.T, name string, data []byte) error {
	t.Helper()
	f, err := fitparser.Decode(bytes.NewReader(data))
	if err == nil {
		t.Errorf("%s: Decode succeeded, want error", name)
		return nil
	}
	if f != nil {
		t.Errorf("%s: Decode returned a file with error %v", name, err)
	}
	if !strings.HasPrefix(err.Error(), "fitparser: ") {
		t.Errorf("%s: error %q lacks the fitparser prefix", name, err)
	}
	return err
}

func TestDecodeEmpty(t *testing.T) {
	mustNotDecode(t, "nil", nil)
	mustNotDecode(t, "empty", []byte{})
}

func TestDecodeGarbage(t *testing.T) {
	mustNotDecode(t, "text", []byte("this is definitely not a FIT file, just some text"))
	mustNotDecode(t, "one byte", []byte{14})
	mustNotDecode(t, "zeros", make([]byte, 64))
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		buf := make([]byte, rng.Intn(256))
		rng.Read(buf)
		mustNotDecode(t, "random", buf)
	}
}

func TestDecodeWrongSignature(t *testing.T) {
	data := sampleFile(t)
	copy(data[8:12], ".FTT")
	mustNotDecode(t, "signature", fitgen.FixCRC(data))
}

func TestDecodeTruncatedAtEveryLength(t *testing.T) {
	data := sampleFile(t)
	for n := 0; n < len(data); n++ {
		mustNotDecode(t, "truncated", data[:n])
	}
	if _, err := fitparser.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("full file: %v", err)
	}
}

func TestDecodeBadCRC(t *testing.T) {
	data := sampleFile(t)

	fileCRC := append([]byte(nil), data...)
	fileCRC[len(fileCRC)-1] ^= 0xFF
	mustNotDecode(t, "file crc", fileCRC)

	body := append([]byte(nil), data...)
	body[len(body)/2] ^= 0x01
	mustNotDecode(t, "flipped data byte", body)

	header := append([]byte(nil), data...)
	header[12] ^= 0xFF
	header[13] ^= 0xFF
	mustNotDecode(t, "header crc", header)

	// Recomputing the CRCs makes the originals decodable again, which shows
	// the failures above are CRC failures.
	if _, err := fitparser.Decode(bytes.NewReader(fitgen.FixCRC(fileCRC))); err != nil {
		t.Errorf("fixed file crc: %v", err)
	}
	if _, err := fitparser.Decode(bytes.NewReader(fitgen.FixCRC(header))); err != nil {
		t.Errorf("fixed header crc: %v", err)
	}
}

func TestDecodeTrailingBytes(t *testing.T) {
	data := sampleFile(t)
	want, err := fitparser.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for name, tail := range map[string][]byte{
		"zero padding": make([]byte, 100),
		"text":         []byte("trailing junk that is not a FIT header"),
	} {
		got, err := fitparser.Decode(bytes.NewReader(append(append([]byte(nil), data...), tail...)))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(got.Records) != len(want.Records) || len(got.Sessions) != 1 {
			t.Errorf("%s: records=%d sessions=%d", name, len(got.Records), len(got.Sessions))
		}
	}
}

func TestDecodeChainedSecondSequenceBroken(t *testing.T) {
	first := sampleFile(t)
	second := sampleFile(t)
	chained := append(append([]byte(nil), first...), second...)
	f, err := fitparser.Decode(bytes.NewReader(chained))
	if err != nil {
		t.Fatalf("chained: %v", err)
	}
	if len(f.Sessions) != 2 || len(f.Records) != 16 {
		t.Errorf("chained: sessions=%d records=%d", len(f.Sessions), len(f.Records))
	}

	// Truncate inside the second sequence (past its header).
	for _, cut := range []int{20, len(second) / 2, len(second) - 1} {
		err := mustNotDecode(t, "second truncated", chained[:len(first)+cut])
		if err != nil && !strings.Contains(err.Error(), "sequence 2") {
			t.Errorf("error %q does not name sequence 2", err)
		}
	}
	bad := append([]byte(nil), chained...)
	bad[len(bad)-1] ^= 0xFF
	if err := mustNotDecode(t, "second bad crc", bad); err != nil && !strings.Contains(err.Error(), "sequence 2") {
		t.Errorf("error %q does not name sequence 2", err)
	}
}

func TestDecodeReaderError(t *testing.T) {
	boom := errors.New("boom")
	_, err := fitparser.Decode(iotest.ErrReader(boom))
	if err == nil {
		t.Fatal("want error")
	}
	data := sampleFile(t)
	r := io.MultiReader(bytes.NewReader(data[:40]), iotest.ErrReader(boom))
	if f, err := fitparser.Decode(r); err == nil || f != nil {
		t.Errorf("mid-stream reader error: f=%v err=%v", f, err)
	}
}

func TestDecodeSlowReaders(t *testing.T) {
	data := sampleFile(t)
	for name, r := range map[string]io.Reader{
		"one byte": iotest.OneByteReader(bytes.NewReader(data)),
		"half":     iotest.HalfReader(bytes.NewReader(data)),
		"eof":      iotest.DataErrReader(bytes.NewReader(data)),
	} {
		f, err := fitparser.Decode(r)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(f.Records) != 8 || !f.HasStryd {
			t.Errorf("%s: records=%d stryd=%v", name, len(f.Records), f.HasStryd)
		}
	}
}

func TestDecodeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.fit")
	if err := os.WriteFile(path, sampleFile(t), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := fitparser.DecodeFile(path)
	if err != nil {
		t.Fatalf("DecodeFile: %v", err)
	}
	if len(f.Sessions) != 1 || len(f.Records) != 8 || len(f.RRIntervals) != 2 {
		t.Errorf("sessions=%d records=%d rr=%d", len(f.Sessions), len(f.Records), len(f.RRIntervals))
	}
}

func TestDecodeFileErrors(t *testing.T) {
	dir := t.TempDir()
	f, err := fitparser.DecodeFile(filepath.Join(dir, "missing.fit"))
	if err == nil || f != nil {
		t.Fatalf("missing: f=%v err=%v", f, err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing: error %v does not wrap fs.ErrNotExist", err)
	}
	if !strings.HasPrefix(err.Error(), "fitparser: ") {
		t.Errorf("missing: error %q lacks prefix", err)
	}
	if _, err := fitparser.DecodeFile(dir); err == nil {
		t.Error("directory: want error")
	}
	empty := filepath.Join(dir, "empty.fit")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := fitparser.DecodeFile(empty); err == nil {
		t.Error("empty file: want error")
	}
}

// TestDecodeMutationsNeverPanic flips random bytes in valid files and
// recomputes the CRCs so the corruption reaches the message parser.
func TestDecodeMutationsNeverPanic(t *testing.T) {
	seeds := [][]byte{sampleFile(t), seedChained(t), seedAllInvalid(t)}
	rng := rand.New(rand.NewSource(42))
	iterations := 3000
	if testing.Short() {
		iterations = 300
	}
	for i := 0; i < iterations; i++ {
		data := append([]byte(nil), seeds[i%len(seeds)]...)
		for n := 1 + rng.Intn(8); n > 0; n-- {
			data[rng.Intn(len(data))] = byte(rng.Intn(256))
		}
		for _, in := range [][]byte{data, fitgen.FixCRC(data)} {
			func() {
				defer func() {
					if p := recover(); p != nil {
						t.Fatalf("panic on mutation %d: %v", i, p)
					}
				}()
				f, err := fitparser.Decode(bytes.NewReader(in))
				if (f == nil) == (err == nil) {
					t.Fatalf("mutation %d: f=%v err=%v", i, f != nil, err)
				}
			}()
		}
	}
}
