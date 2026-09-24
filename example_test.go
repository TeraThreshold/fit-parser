package fitparser_test

import (
	"bytes"
	"fmt"
	"log"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/proto"
)

// exampleRun returns a synthetic 10-minute run with Stryd developer fields,
// recorded in a UTC+2 time zone.
func exampleRun() []byte {
	return fitgen.New().
		AddStrydDescriptions(0).
		AddRun(fitgen.Run{
			Start:     time.Date(2024, 5, 4, 7, 30, 0, 0, time.UTC),
			Seconds:   600,
			UTCOffset: 2 * time.Hour,
			RecordDev: func(i int) []proto.DeveloperField {
				return fitgen.StrydValues(0, fitgen.StrydSample{
					Power: 255, GroundTime: 238, VerticalOscillation: 8.4,
					FormPower: 68, LegSpringStiffness: 10.5, AirPower: 4,
				})
			},
		}).
		MustBytes()
}

func ExampleDecode() {
	f, err := fitparser.Decode(bytes.NewReader(exampleRun()))
	if err != nil {
		log.Fatal(err)
	}
	if len(f.Sessions) == 0 {
		log.Fatal("not an activity")
	}
	s := f.Sessions[0]
	fmt.Println("sport:", s.Sport, s.SubSport)
	fmt.Println("device:", f.FileID.Manufacturer, f.FileID.ProductName)
	fmt.Println("start:", s.StartTime.In(f.Location()).Format("2006-01-02 15:04 -07:00"))
	fmt.Println("timer:", s.TimerTime())
	fmt.Printf("distance: %.2f km\n", s.DistanceM()/1000)
	fmt.Printf("avg speed: %.1f m/s\n", s.AvgSpeedMps())
	fmt.Println("avg/max HR:", s.AvgHeartRate, s.MaxHeartRate)
	fmt.Println("laps:", len(f.Laps), "records:", len(f.Records))
	fmt.Println("stryd:", f.HasStryd)
	// Output:
	// sport: running generic
	// device: garmin fr965
	// start: 2024-05-04 09:30 +02:00
	// timer: 10m0s
	// distance: 1.80 km
	// avg speed: 3.0 m/s
	// avg/max HR: 150 159
	// laps: 1 records: 600
	// stryd: true
}

func ExampleRecord_Position() {
	f, err := fitparser.Decode(bytes.NewReader(fitgen.New().
		AddRecord(func(r *mesgdef.Record) {
			r.Timestamp = time.Date(2024, 5, 4, 7, 30, 0, 0, time.UTC)
			r.PositionLat = fitgen.Degrees(10)
			r.PositionLong = fitgen.Degrees(20)
		}).
		AddRecord(func(r *mesgdef.Record) {
			r.Timestamp = time.Date(2024, 5, 4, 7, 30, 1, 0, time.UTC)
			// No GPS fix: position left unset.
		}).
		MustBytes()))
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range f.Records {
		if lat, lon, ok := r.Position(); ok {
			fmt.Printf("%s %.5f %.5f\n", r.Timestamp.Format(time.TimeOnly), lat, lon)
		} else {
			fmt.Printf("%s no position\n", r.Timestamp.Format(time.TimeOnly))
		}
	}
	// Output:
	// 07:30:00 10.00000 20.00000
	// 07:30:01 no position
}

func ExampleRecord_SpeedMps() {
	f, err := fitparser.Decode(bytes.NewReader(exampleRun()))
	if err != nil {
		log.Fatal(err)
	}
	r := f.Records[60]
	pace := time.Duration(1000 / r.SpeedMps() * float64(time.Second)).Round(time.Second)
	fmt.Printf("%.1f m/s, pace %s/km, %.0f m\n", r.SpeedMps(), pace, r.DistanceM())
	// Output:
	// 3.0 m/s, pace 5m33s/km, 180 m
}

func ExampleRecord_TemperatureC() {
	f, err := fitparser.Decode(bytes.NewReader(exampleRun()))
	if err != nil {
		log.Fatal(err)
	}
	if c, ok := f.Records[0].TemperatureC(); ok {
		fmt.Printf("%.0f °C\n", c)
	}
	// Output:
	// 21 °C
}

func ExampleFile_Location() {
	f, err := fitparser.Decode(bytes.NewReader(exampleRun()))
	if err != nil {
		log.Fatal(err)
	}
	r := f.Records[0]
	fmt.Println("UTC:  ", r.Timestamp.Format(time.DateTime))
	fmt.Println("local:", r.Timestamp.In(f.Location()).Format(time.DateTime))
	fmt.Println("offset:", f.UTCOffset, f.HasUTCOffset)
	// Output:
	// UTC:   2024-05-04 07:30:00
	// local: 2024-05-04 09:30:00
	// offset: 2h0m0s true
}

func ExampleStryd() {
	f, err := fitparser.Decode(bytes.NewReader(exampleRun()))
	if err != nil {
		log.Fatal(err)
	}
	if s := f.Records[0].Stryd; s != nil {
		fmt.Printf("power %d W, ground time %d ms, LSS %.1f kN/m\n", s.Power, s.GroundTime, s.LegSpringStiffness)
	}
	// Output:
	// power 255 W, ground time 238 ms, LSS 10.5 kN/m
}
