package fitparser_test

import (
	"testing"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
	"github.com/muktihari/fit/profile/mesgdef"
)

func activity(ts, local time.Time) func(*mesgdef.Activity) {
	return func(m *mesgdef.Activity) {
		m.Timestamp = ts
		m.LocalTimestamp = local
		m.NumSessions = 1
	}
}

func TestUTCOffset(t *testing.T) {
	pre2000 := time.Date(1999, 12, 31, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		ts      time.Time
		local   time.Time
		want    time.Duration
		wantHas bool
	}{
		{"plus two hours", t0, t0.Add(2 * time.Hour), 2 * time.Hour, true},
		{"minus five hours", t0, t0.Add(-5 * time.Hour), -5 * time.Hour, true},
		{"half hour zone", t0, t0.Add(5*time.Hour + 30*time.Minute), 5*time.Hour + 30*time.Minute, true},
		{"zero offset", t0, t0, 0, true},
		{"plus 14h limit", t0, t0.Add(14 * time.Hour), 14 * time.Hour, true},
		{"minus 14h limit", t0, t0.Add(-14 * time.Hour), -14 * time.Hour, true},
		{"above +14h", t0, t0.Add(14*time.Hour + time.Second), 0, false},
		{"below -14h", t0, t0.Add(-14*time.Hour - 15*time.Minute), 0, false},
		{"days apart", t0, t0.Add(72 * time.Hour), 0, false},
		{"timestamp before 2000", pre2000, pre2000.Add(time.Hour), 0, false},
		{"local timestamp before 2000", t0, pre2000, 0, false},
		{"missing local timestamp", t0, time.Time{}, 0, false},
		{"missing timestamp", time.Time{}, t0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := decode(t, fitgen.New().AddActivity(activity(c.ts, c.local)))
			if f.UTCOffset != c.want || f.HasUTCOffset != c.wantHas {
				t.Errorf("UTCOffset, HasUTCOffset = %v, %v; want %v, %v", f.UTCOffset, f.HasUTCOffset, c.want, c.wantHas)
			}
			loc := f.Location()
			if !c.wantHas {
				if loc != time.UTC {
					t.Errorf("Location() = %v, want time.UTC", loc)
				}
				return
			}
			name, off := t0.In(loc).Zone()
			if off != int(c.want/time.Second) || name != "" {
				t.Errorf("Location() zone = %q %d, want \"\" %d", name, off, int(c.want/time.Second))
			}
		})
	}
}

func TestUTCOffsetMissingActivity(t *testing.T) {
	f := decode(t, fitgen.New().AddSession(func(m *mesgdef.Session) { m.StartTime = t0 }))
	if f.HasUTCOffset || f.UTCOffset != 0 {
		t.Errorf("UTCOffset = %v, %v; want 0, false", f.UTCOffset, f.HasUTCOffset)
	}
	if f.Location() != time.UTC {
		t.Errorf("Location() = %v, want UTC", f.Location())
	}
}

func TestUTCOffsetLastValidActivityWins(t *testing.T) {
	f := decode(t, fitgen.New().
		AddActivity(activity(t0, t0.Add(time.Hour))).
		AddActivity(activity(t0, t0.Add(3*time.Hour))).
		AddActivity(activity(t0, t0.Add(20*time.Hour)))) // rejected: keeps previous
	if !f.HasUTCOffset || f.UTCOffset != 3*time.Hour {
		t.Errorf("UTCOffset = %v, %v; want 3h, true", f.UTCOffset, f.HasUTCOffset)
	}
}

func TestLocationNilFile(t *testing.T) {
	var f *fitparser.File
	if f.Location() != time.UTC {
		t.Errorf("nil File Location() = %v, want UTC", f.Location())
	}
	if (&fitparser.File{UTCOffset: time.Hour}).Location() != time.UTC {
		t.Error("Location() without HasUTCOffset should be UTC")
	}
}

func TestTimestampsAreUTC(t *testing.T) {
	berlin := time.FixedZone("CEST", 2*3600)
	start := t0.In(berlin)
	f := decode(t, fitgen.New().AddRun(fitgen.Run{Start: start, Seconds: 3, UTCOffset: 2 * time.Hour}))
	check := func(what string, ts time.Time) {
		t.Helper()
		if ts.Location() != time.UTC {
			t.Errorf("%s location = %v, want UTC", what, ts.Location())
		}
	}
	check("FileID.TimeCreated", f.FileID.TimeCreated)
	check("Session.StartTime", f.Sessions[0].StartTime)
	check("Lap.StartTime", f.Laps[0].StartTime)
	check("Record.Timestamp", f.Records[0].Timestamp)
	check("Event.Timestamp", f.Events[0].Timestamp)
	if !f.Sessions[0].StartTime.Equal(t0) {
		t.Errorf("StartTime = %v, want %v", f.Sessions[0].StartTime, t0)
	}
	if got := f.Records[2].Timestamp.In(f.Location()).Format("15:04:05"); got != "09:30:02" {
		t.Errorf("local time = %s, want 09:30:02", got)
	}
}
