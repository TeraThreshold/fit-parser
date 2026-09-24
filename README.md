# fit-parser

[![Go Reference](https://pkg.go.dev/badge/github.com/TeraThreshold/fit-parser.svg)](https://pkg.go.dev/github.com/TeraThreshold/fit-parser)
[![CI](https://github.com/TeraThreshold/fit-parser/actions/workflows/ci.yml/badge.svg)](https://github.com/TeraThreshold/fit-parser/actions/workflows/ci.yml)

fit-parser decodes Garmin FIT activity files into plain Go structs where
FIT's invalid sentinels become zero values (the few fields where zero is a
real value keep the sentinel behind an `ok` accessor). It is built on the
[muktihari/fit](https://github.com/muktihari/fit) decoder and adds the layer
that most activity-analysis code otherwise writes by hand: it picks out the
messages that matter, exposes enums as readable names, reads developer
fields safely and provides accessors that return SI units.

It was extracted from TeraThreshold's production activity pipeline.

## Why

- **Clean structs.** `File`, `Session`, `Lap`, `Record`, `Event`,
  `DeviceInfo`, `Split`, `TimeInZone`, `Workout`, `WorkoutStep` and the zone
  messages. A missing value is the Go zero value, so `r.Power > 0` is the
  whole check; positions, temperature and a few indexes are the
  [exceptions](#units-and-missing-values). The API exposes no types from the
  upstream decoder.
- **Running dynamics.** Ground contact time, vertical oscillation, step length
  and vertical ratio for records, laps and sessions, plus ground contact time
  balance for records and sessions.
- **Stryd developer fields.** Power, ground time, vertical oscillation, form
  power, leg spring stiffness and air power. They are read only from the
  developer data index that declares the Stryd fields, so a field #0 from
  another Connect IQ app is never mistaken for Stryd power.
- **Every other developer field.** Values are keyed by developer data index
  and field number, and carry the name and units from their field
  description.
- **HRV.** Beat-to-beat R-R intervals from the `hrv` messages, in
  milliseconds.
- **Local time.** The activity's UTC offset, and `File.Location()` for
  showing timestamps in the athlete's local time.
- **Chained files.** Every FIT sequence in the file is decoded and merged.
- **Safe on bad input.** It does not panic on malformed files, never logs
  and never reads the clock. Errors are wrapped with context.
- **A small CLI.** `fitparse` prints a summary of a file or its content as
  JSON.

## Install

```sh
go get github.com/TeraThreshold/fit-parser
```

The command-line tool:

```sh
go install github.com/TeraThreshold/fit-parser/cmd/fitparse@latest
```

Requires Go 1.24 or later.

## Quick start

```go
package main

import (
	"fmt"
	"log"
	"os"

	fitparser "github.com/TeraThreshold/fit-parser"
)

func main() {
	f, err := fitparser.DecodeFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	if len(f.Sessions) == 0 {
		log.Fatal("not an activity: the file has no session message")
	}

	loc := f.Location() // the activity's local time zone, UTC if unknown
	s := &f.Sessions[0]
	fmt.Printf("%s (%s), %s\n", s.Sport, s.SubSport, s.StartTime.In(loc).Format("2006-01-02 15:04 -07:00"))
	fmt.Printf("%.2f km in %s, avg %.2f m/s, avg HR %d bpm\n",
		s.DistanceM()/1000, s.TimerTime(), s.AvgSpeedMps(), s.AvgHeartRate)

	for i := range f.Records {
		r := &f.Records[i]
		lat, lon, ok := r.Position()
		if !ok {
			continue // no GPS fix in this sample
		}
		fmt.Printf("%s  %.5f,%.5f  %.2f m/s  %d bpm\n",
			r.Timestamp.In(loc).Format("15:04:05"), lat, lon, r.SpeedMps(), r.HeartRate)
	}
}
```

The first lines of its output for the synthetic sample run used throughout
this README:

```text
running (generic), 2024-05-04 09:30 +02:00
8.10 km in 45m0s, avg 3.00 m/s, avg HR 150 bpm
09:30:00  10.00000,20.00000  3.00 m/s  140 bpm
09:30:01  10.00000,20.00000  3.00 m/s  141 bpm
09:30:02  10.00000,20.00000  3.00 m/s  142 bpm
...
```

`fitparser.Decode` does the same for any `io.Reader`, such as an HTTP request
body.

### Running dynamics, Stryd, developer fields and HRV

```go
package main

import (
	"fmt"
	"log"
	"os"

	fitparser "github.com/TeraThreshold/fit-parser"
)

func main() {
	f, err := fitparser.DecodeFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	for i := range f.Records {
		r := &f.Records[i]

		// Running dynamics from the watch, a chest strap or a foot pod.
		if r.StanceTime > 0 {
			fmt.Printf("GCT %.0f ms, VO %.1f mm, step %.2f m, VR %.2f %%\n",
				r.StanceTimeMs(), r.VerticalOscillationMm(), r.StepLengthM(), r.VerticalRatioPercent())
		}

		// Stryd metrics: nil unless the record carries Stryd developer fields.
		if s := r.Stryd; s != nil {
			fmt.Printf("Stryd %d W, form power %d W, LSS %.2f kN/m\n",
				s.Power, s.FormPower, s.LegSpringStiffness)
		}

		// All developer fields, Stryd's included, keyed by
		// (developer data index, field number).
		for _, d := range r.DeveloperFields {
			fmt.Printf("dev %d/%d %s = %v %s\n", d.DeveloperDataIndex, d.Num, d.Name, d.Value, d.Units)
		}
	}

	// Beat-to-beat intervals from the hrv messages, in milliseconds.
	fmt.Println(len(f.RRIntervals), "R-R intervals")
}
```

## Units and missing values

**Raw units, SI accessors.** Struct fields hold the raw FIT values, so
decoding is lossless. Each field's documentation states its unit and scale,
and accessor methods return SI values. Every timestamp is a `time.Time` in
UTC; use `t.In(f.Location())` for local time.

**Sentinels become zero.** FIT marks a missing value with the invalid value of
the field's base type: 0xFF for a uint8, 0xFFFF for a uint16, 0x7FFFFFFF for a
sint32 and so on. fitparser turns every such value into the Go zero value, so
zero means "absent", or a real zero where zero can be measured (0 m of
ascent, for example). In arrays such as the time-in-zone fields, invalid
elements become 0, and an array whose elements are all invalid becomes nil.

**The exceptions** are fields where zero is a legitimate value. These keep the
FIT invalid value, and an accessor with an `ok` result tells you whether the
value is present:

- Positions (0°, 0° is a real place): `Record.PositionLat`/`PositionLong` and
  the `Split` positions keep `math.MaxInt32`. Use `Position()`,
  `StartPosition()` and `EndPosition()`.
- Temperature (0 °C): `Record.Temperature` keeps 127. Use `TemperatureC()`.
- Indexes where 0 means something: `DeviceInfo.DeviceIndex` keeps 255 (0 is
  the recording device), and `HRZone.MessageIndex`, `PowerZone.MessageIndex`
  and `TimeInZone.ReferenceIndex` keep 0xFFFF.

The most-used fields:

| Quantity | Field (raw FIT unit) | Accessor |
| --- | --- | --- |
| Time | `Record.Timestamp`, `Session.StartTime`, `Lap.StartTime` (`time.Time`, UTC) | `t.In(f.Location())` |
| Speed | `Record.EnhancedSpeed` (mm/s) | `SpeedMps()` |
| Average / max speed | `Session.EnhancedAvgSpeed`, `EnhancedMaxSpeed` (mm/s) | `AvgSpeedMps()`, `MaxSpeedMps()` |
| Distance | `Record.Distance`, `Session.TotalDistance`, `Lap.TotalDistance` (cm) | `DistanceM()` |
| Durations | `Session.TotalElapsedTime`, `TotalTimerTime`, `TotalMovingTime` (ms) | `ElapsedTime()`, `TimerTime()`, `MovingTime()` |
| Altitude | `Record.EnhancedAltitude` (scale 5, offset 500) | `AltitudeM() (m, ok)` |
| Position | `Record.PositionLat`, `PositionLong` (semicircles) | `Position() (lat, lon, ok)` |
| Temperature | `Record.Temperature` (°C) | `TemperatureC() (c, ok)` |
| Heart rate, cadence, power | `HeartRate` (bpm), `Cadence` (rpm), `Power` (W) | already SI |
| Ground contact time | `Record.StanceTime` (0.1 ms) | `StanceTimeMs()` |
| Vertical oscillation | `Record.VerticalOscillation` (0.1 mm) | `VerticalOscillationMm()` |
| Step length | `Record.StepLength` (0.1 mm) | `StepLengthM()` |
| Vertical ratio | `Record.VerticalRatio` (0.01 %) | `VerticalRatioPercent()` |
| Respiration rate | `Record.EnhancedRespirationRate` (0.01 breaths/min) | `RespirationRateBrpm()` |
| Training stress | `Session.TrainingStressScore` (×10), `IntensityFactor` (×1000) | none |
| R-R intervals | `File.RRIntervals` (ms) | none |
| Stryd | `Record.Stryd` (W, ms, cm, kN/m) | already converted |

`Lap` and `Session` have `Avg`-prefixed running-dynamics accessors, for
example `AvgStanceTimeMs()` and `AvgStepLengthM()`. The package
documentation on [pkg.go.dev](https://pkg.go.dev/github.com/TeraThreshold/fit-parser)
lists every field with its unit.

## CLI: fitparse

```text
Usage: fitparse [flags] FILE...

Prints a summary of each FIT file, or its decoded content as JSON.

Flags:
  -json
    	print the decoded file as JSON instead of a summary
  -pretty
    	indent the JSON output (implies -json)
  -records
    	include records in the JSON output (implies -json)
  -units unit
    	summary distance unit: km or mi (default "km")
```

The summary shows the start time in the file's local time. The sample file
below is synthetic, generated with the test helper in `internal/fitgen`:

```console
$ fitparse run.fit
run.fit
  Type        activity
  Sport       running (generic)
  Start       2024-05-04 09:30:00 +02:00
  Duration    0:45:00
  Distance    8.10 km
  Avg pace    5:33 /km (10.8 km/h)
  Heart rate  avg 150, max 159 bpm
  Power       avg 250, max 250 W
  Ascent      0 m
  Laps        1
  Records     2700
  Device      garmin fr965 (software 12.34)
  Stryd       yes
```

With `-units mi`, distance, pace and speed use miles and ascent uses feet:

```console
$ fitparse -units mi run.fit | grep -E 'Distance|pace|Ascent'
  Distance    5.03 mi
  Avg pace    8:56 /mi (6.7 mph)
  Ascent      0 ft
```

`-json` writes one JSON document for each file that decodes, in argument
order, with the same field names as the library's JSON tags. A file that
fails to decode produces no document, only an error on standard error.
Records are left out unless you pass `-records`:

```console
$ fitparse -json run.fit | jq '.sessions[0] | {sport, start_time, total_distance, enhanced_avg_speed, avg_heart_rate}'
{
  "sport": "running",
  "start_time": "2024-05-04T07:30:00Z",
  "total_distance": 810000,
  "enhanced_avg_speed": 3000,
  "avg_heart_rate": 150
}
$ fitparse -json -records run.fit | jq -c '.records[10] | {timestamp, heart_rate, enhanced_speed, stryd}'
{"timestamp":"2024-05-04T07:30:10Z","heart_rate":150,"enhanced_speed":3000,"stryd":{"power":260,"ground_time":240,"vertical_oscillation":8.5,"form_power":70,"leg_spring_stiffness":10.25,"air_power":3}}
```

Errors go to standard error, prefixed with the file name. The exit status is
1 if any file fails to decode (the other files are still processed) and 2
for invalid usage:

```console
$ fitparse bad.fit
bad.fit: fitparser: decode: decode file header [byte pos: 1]: file header size [110] is invalid: not a FIT file
```

## Notes

- **Raw units plus accessors.** Fields keep the FIT unit (cm, mm/s, 0.1 ms,
  ...) so nothing is lost to rounding. Use the accessors for SI values; do not
  assume a field is already in meters or seconds.
- **The sentinel rule.** An invalid FIT value becomes the Go zero value.
  Positions and temperature are the exception: they keep the FIT invalid
  value (`math.MaxInt32`, 127), and `Position()` / `TemperatureC()` report
  whether a value is present. In JSON these fields are always written, so a
  record without GPS shows `"position_lat": 2147483647`. The same applies to
  the device and message indexes listed above.
- **No plausibility limits, with one exception.** Values are not capped:
  2000 W sprint power or a heart rate of 230 is passed through as recorded,
  and filtering is up to you. The one exception is respiration rate: values
  above 100 breaths/min (a raw value above 10000) are treated as absent,
  because some devices write junk there. The legacy 8-bit `respiration_rate`
  is multiplied by 100 into `EnhancedRespirationRate` when the enhanced field
  is absent.
- **Enums are FIT profile names.** Examples are `"running"`, `"trail"`,
  `"treadmill"`, `"session_end"` and `"stop_all"`. An absent value is `""`,
  and a value the FIT profile does not define is `"<type>_<n>"`, for example
  `"sport_250"`. No app-specific mapping is applied: for example, strength
  workouts arrive as sport `"training"` with sub-sport
  `"strength_training"`. `ProductName` is the message's `product_name` when
  set; otherwise it is the profile name for Garmin-family and Favero products
  (such as `"fr965"`). A product number the profile does not know becomes
  `"garmin_product_<n>"` or `"favero_product_<n>"`.
- **A file without a session is not an error.** Workout, course and settings
  files decode normally. Activity callers should check
  `len(f.Sessions) > 0`.
- **How Stryd is detected.** `File.HasStryd` is true when a developer data
  index declares a field named `"Leg Spring Stiffness"` or `"Form Power"`.
  `Record.Stryd` is filled only from that index, matching fields by name
  (`"Power"`, `"Ground Time"`, `"Vertical Oscillation"`, `"Form Power"`,
  `"Leg Spring Stiffness"`, `"Air Power"`). Integer and float encodings are
  both accepted, and zero, negative or invalid values are dropped.
  `Record.Stryd` is nil when a record has no valid Stryd value.
- **Developer field values are raw.** `DeveloperFieldValue.Value` has the Go
  type of the field's base type (`uint16`, `float32`, `string`, a slice, ...),
  except that a `uint8` or `byte` array becomes `[]uint16` so JSON shows it as
  numbers. The scale and offset from the field description are not applied.
  A scalar equal to the base type's invalid sentinel is dropped. An array is
  dropped only when all its elements are invalid; otherwise it is returned
  as-is, and individual elements may still hold the sentinel. NaN or infinite
  floats are dropped. When a sequence has several descriptions for the same
  developer data index and field number, the first one applies, as in the
  upstream decoder.
- **Chained files.** All FIT sequences are decoded and merged in file order.
  Developer fields are resolved separately in each sequence. `FileID` is the
  first `file_id` message and `ZonesTarget` the last `zones_target` message.
  Trailing bytes after a complete sequence that are not a FIT header, such as
  zero padding, are ignored. A truncated or corrupt later sequence is an
  error.
- **Local time.** `UTCOffset` is `activity.local_timestamp` minus
  `activity.timestamp`. It is set only when both are after 2000-01-01 and the
  offset is at most 14 hours; otherwise `HasUTCOffset` is false and
  `Location()` returns UTC. It is a fixed offset, not a named time zone. In
  JSON, `utc_offset` is a Go `time.Duration`, so it is written as integer
  nanoseconds (`7200000000000` for +02:00).
- **Test data.** All test fixtures are synthetic and built with
  `internal/fitgen`. The repository contains no real activity recordings.

## Acknowledgements

- [muktihari/fit](https://github.com/muktihari/fit) by Hikmatulloh Hari Mukti
  (BSD-3-Clause) does all of the binary FIT decoding. fit-parser is a layer on
  top of it.
- The FIT protocol and profile are developed by Garmin. This project is not
  affiliated with or endorsed by Garmin or Stryd.

## License

MIT. See [LICENSE](LICENSE).
