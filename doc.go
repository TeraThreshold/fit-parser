// Package fitparser decodes Garmin FIT activity files into plain Go structs.
//
// It is a thin, opinionated layer over github.com/muktihari/fit: it extracts
// the messages most activity analysis needs (file_id, session, lap, record,
// event, device_info, split, time_in_zone, workout, workout_step,
// zones_target, hr_zone, power_zone, hrv, activity and developer fields,
// including Stryd running power) and exposes them without any upstream
// types.
//
// # Units
//
// Struct fields hold raw FIT values, so decoding is lossless; each field's
// documentation states its unit and scale (for example Record.Distance is in
// centimeters and Record.EnhancedSpeed in millimeters per second). Accessor
// methods return SI values: Record.SpeedMps, Record.DistanceM,
// Session.ElapsedTime and so on. Timestamps are time.Time values in UTC; use
// File.Location to convert them to the activity's local time.
//
// # Missing values
//
// A field whose FIT value is the base type's invalid sentinel (0xFF, 0xFFFF,
// 0xFFFFFFFF, 0x7FFF, ...) is reported as the Go zero value. In array fields
// invalid elements become zero, and an array whose elements are all invalid
// is nil. Where zero is a legitimate measurement the FIT invalid value is
// kept instead and an accessor reports validity: Record.Position and
// Record.TemperatureC, the Split position accessors, and the device and
// message indexes, where 0 is a valid index. The library applies no
// plausibility limits beyond one: a respiration rate above 100 breaths/min
// is treated as absent.
//
// # Enumerations
//
// Enumerated values (sport, sub-sport, lap trigger, event, ...) are FIT
// profile names such as "running", "trail", "session_end" or "stop_all". An
// absent value is "". A value the FIT profile does not define is reported as
// "<type>_<n>", for example "sport_250".
//
// # Chained files and non-activity files
//
// A FIT file may contain several FIT sequences back to back; Decode reads
// all of them and merges their messages in file order. Files without a
// session message (workouts, courses, settings) decode without error, so
// callers that expect an activity should check len(f.Sessions) > 0.
//
// # Developer fields
//
// Developer (Connect IQ) fields are identified by the pair of developer data
// index and field number, so fields of different apps never collide. Record
// developer values are available in Record.DeveloperFields. Stryd metrics
// are read by field name, and only from the developer data index that
// declares the Stryd-specific fields "Leg Spring Stiffness" or "Form Power".
//
// # Example
//
//	f, err := fitparser.DecodeFile("activity.fit")
//	if err != nil {
//		log.Fatal(err)
//	}
//	if len(f.Sessions) == 0 {
//		log.Fatal("not an activity")
//	}
//	s := f.Sessions[0]
//	fmt.Printf("%s: %.2f km in %s\n", s.Sport, s.DistanceM()/1000, s.TimerTime())
//	for _, r := range f.Records {
//		if lat, lon, ok := r.Position(); ok {
//			fmt.Println(r.Timestamp.In(f.Location()), lat, lon, r.HeartRate)
//		}
//	}
package fitparser
