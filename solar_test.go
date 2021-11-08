package solar

import (
	"testing"
	"time"
)

// Coordinates for Los Angeles, CA in degrees.
const (
	latitude  = 34.1
	longitude = -118.2
)

var losAngeles *time.Location

func init() {
	z, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic("cannot load America/Los_Angeles: " + err.Error())
	}
	losAngeles = z
}

func TestCalculateSun(t *testing.T) {
	t.Run("PDT", func(t *testing.T) {
		// This is the same day as in the DST test but subtracted 1 day off,
		// which was before DST was switched off.
		ts := time.Unix(1636333967-86400, 0)
		ts = ts.In(losAngeles)

		exp := Sun{
			Dawn:    timeIn(t, ts, "05:56:52"),
			Sunrise: timeIn(t, ts, "06:39:05"),
			Sunset:  timeIn(t, ts, "16:48:33"),
			Dusk:    timeIn(t, ts, "17:30:47"),
		}

		assertSun(t, ts, exp)
	})

	t.Run("DST", func(t *testing.T) {
		// 11/07/2021 05:12:47 PM
		// Today happens to be the day that Daylight Saving Time switched off.
		// You'll notice that if you don't use t.Truncate and instead use
		// time.UTC to do the seconds conversion math, the time will be 1 hour
		// off.

		ts := time.Unix(1636333967, 0)
		ts = ts.In(losAngeles)

		// Google reports (in PST) that the sunrise time is 6:19AM and sunset
		// time is 4:54PM. The results of these are taken from the code, but
		// they're very close to what Google has.
		exp := Sun{
			Dawn:    timeIn(t, ts, "05:57:52"),
			Sunrise: timeIn(t, ts, "06:40:03"),
			Sunset:  timeIn(t, ts, "16:47:45"),
			Dusk:    timeIn(t, ts, "17:29:56"),
		}

		assertSun(t, ts, exp)
	})

	t.Run("PST", func(t *testing.T) {
		// This is the same day as in the DST test but added 1 day in, which was
		// after DST was switched off.
		ts := time.Unix(1636333967+86400, 0)
		ts = ts.In(losAngeles)

		exp := Sun{
			Dawn:    timeIn(t, ts, "05:58:52"),
			Sunrise: timeIn(t, ts, "06:41:01"),
			Sunset:  timeIn(t, ts, "16:46:58"),
			Dusk:    timeIn(t, ts, "17:29:07"),
		}

		assertSun(t, ts, exp)
	})
}

func timeIn(t *testing.T, ts time.Time, clock string) time.Time {
	v, err := time.Parse(sclockf, clock)
	if err != nil {
		t.Fatalf("cannot parse %s: %v", clock, err)
	}

	return time.Date(
		ts.Year(), ts.Month(), ts.Day(),
		v.Hour(), v.Minute(), v.Second(), 0,
		ts.Location(),
	)
}

func assertSun(t *testing.T, ts time.Time, exp Sun) {
	sun := CalculateSun(ts, latitude)

	assertTime(t, time.Second, exp.Dawn, sun.Dawn)
	assertTime(t, time.Second, exp.Sunrise, sun.Sunrise)
	assertTime(t, time.Second, exp.Sunset, sun.Sunset)
	assertTime(t, time.Second, exp.Dusk, sun.Dusk)
}

func assertTime(t *testing.T, trunc time.Duration, exp, got time.Time) {
	exp = exp.Truncate(trunc)
	got = got.Truncate(trunc)
	if !exp.Equal(got) {
		t.Errorf("expected %s != got %s", exp, got)
	}
}

func TestDaysInYear(t *testing.T) {
	var days int
	assert := func(name string, want int) {
		if days != want {
			t.Fatalf("%s: want %d days, got %d", name, want, days)
		}
	}

	// leap
	days = daysInYear(time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC))
	assert("2020", 366)

	// non-leap
	days = daysInYear(time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC))
	assert("2021", 365)
}
