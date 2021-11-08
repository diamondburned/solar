package solar

import (
	"fmt"
	"math"
	"testing"
	"time"

	_ "time/tzdata"
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
		panic("cannot load embedded America/Los_Angeles: " + err.Error())
	}

	losAngeles = z
}

const epochDay = 86400

func TestCalculateSun(t *testing.T) {
	t.Run("PDT", func(t *testing.T) {
		// This is the same day as in the DST test but subtracted 1 day off,
		// which was before DST was switched off.
		ts := time.Unix(1636333967-epochDay, 0)
		ts = ts.In(losAngeles)

		exp := Sun{
			Dawn:    timeIn(t, ts, "05:26:25"),
			Sunrise: timeIn(t, ts, "06:08:41"),
			Sunset:  timeIn(t, ts, "16:19:56"),
			Dusk:    timeIn(t, ts, "17:02:12"),
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
			Dawn:    timeIn(t, ts, "05:28:25"),
			Sunrise: timeIn(t, ts, "06:10:36"),
			Sunset:  timeIn(t, ts, "16:18:18"),
			Dusk:    timeIn(t, ts, "17:00:30"),
		}

		assertSun(t, ts, exp)
	})

	t.Run("PST", func(t *testing.T) {
		// This is the same day as in the DST test but added 1 day in, which was
		// after DST was switched off.
		ts := time.Unix(1636333967+epochDay, 0)
		ts = ts.In(losAngeles)

		exp := Sun{
			Dawn:    timeIn(t, ts, "05:28:25"),
			Sunrise: timeIn(t, ts, "06:10:36"),
			Sunset:  timeIn(t, ts, "16:18:18"),
			Dusk:    timeIn(t, ts, "17:00:30"),
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
	sun := CalculateSun(ts, latitude, longitude)
	t.Log("sun =", sun)

	assertTime(t, "dawn", time.Second, exp.Dawn, sun.Dawn)
	assertTime(t, "rise", time.Second, exp.Sunrise, sun.Sunrise)
	assertTime(t, "set ", time.Second, exp.Sunset, sun.Sunset)
	assertTime(t, "dusk", time.Second, exp.Dusk, sun.Dusk)
}

func assertTime(t *testing.T, name string, trunc time.Duration, exp, got time.Time) {
	exp = exp.Truncate(trunc)
	got = got.Truncate(trunc)
	if !exp.Equal(got) {
		t.Errorf("%s: expected %s, got %s", name, exp, got)
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

func TestTimeWithSeconds(t *testing.T) {
	assert := func(t *testing.T, ts time.Time, h, m, s int) {
		t.Helper()
		hh, mm, ss := ts.Clock()
		if hh != h || mm != m || ss != s {
			t.Errorf("expected %d:%d:%d, got %d:%d:%d", h, m, s, hh, mm, ss)
		}
	}

	t.Run("PDT", func(t *testing.T) {
		ts := time.Unix(1636333967-epochDay, 0)
		ts = ts.In(losAngeles)
		ts = timeTruncateDay(ts)
		ts = timeAddSeconds(ts, 1)
		assert(t, ts, 00, 00, 01)
	})

	t.Run("PST", func(t *testing.T) {
		ts := time.Unix(1636333967+epochDay, 0)
		ts = ts.In(losAngeles)
		ts = timeTruncateDay(ts)
		ts = timeAddSeconds(ts, 1)
		assert(t, ts, 00, 00, 01)
	})

	t.Run("DST", func(t *testing.T) {
		ts := time.Unix(1636333967, 0)
		ts = ts.In(losAngeles)
		ts = timeTruncateDay(ts)
		ts = timeAddSeconds(ts, 1)
		assert(t, ts, 01, 00, 01)
	})
}

func TestCalcCondition(t *testing.T) {
	asserter := func(t *testing.T, expect SunCondition) func(f1, f2 float64) {
		return func(f1, f2 float64) {
			t.Helper()
			got := calcCondition(f1, f2)
			t.Logf("lat %3.0f, decl %3.0f: %s", f1, f2, got)
			if expect != got {
				t.Errorf("expected %s, got %s", expect, got)
			}
		}
	}

	t.Run("midnight sun", func(t *testing.T) {
		assert := asserter(t, MidnightSun)

		assert(-math.NaN(), -math.NaN())
		assert(-math.NaN(), -1)
		assert(-1, -math.NaN())

		assert(math.NaN(), math.NaN())
		assert(math.NaN(), 1)
		assert(1, math.NaN())
	})

	t.Run("polar night sun", func(t *testing.T) {
		assert := asserter(t, PolarNightSun)

		assert(-math.NaN(), math.NaN())
		assert(-math.NaN(), 1)
		assert(-1, math.NaN())

		assert(math.NaN(), -math.NaN())
		assert(math.NaN(), -1)
		assert(1, -math.NaN())
	})
}

func TestCalculateWhitepoint(t *testing.T) {
	eq := func(c1, c2 [3]float64) bool {
		for i := range c1 {
			if !feq(c1[i], c2[i]) {
				return false
			}
		}
		return true
	}

	type test struct {
		temperature Temperature
		whitepoint  [3]float64
	}

	var tests = []test{
		{50000, rgb(0.59187, 0.727766, 1)},
		{25000, rgb(0.59187, 0.727766, 1)},
		{6500, rgb(1, 1, 1)},
		{4000, rgb(1, 0.823415, 0.597612)},
		{2500, rgb(1, 0.617219, 0.251946)},
		{1667, rgb(1, 0.462962, 0)},
		{0, rgb(1, 0.462962, 0)},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%.0fK", test.temperature), func(t *testing.T) {
			r, g, b := CalculateWhitepoint(test.temperature)
			if c := rgb(r, g, b); !eq(c, test.whitepoint) {
				t.Errorf("%.0fK: expected %v, got %v", test.temperature, test.whitepoint, c)
			}
		})
	}

	t.Run("0K-50000K", func(t *testing.T) {
		// Ensure that this will never panic.
		for f := Temperature(0); f < 50000; f++ {
			CalculateWhitepoint(f)
		}
	})
}

func rgb(r, g, b float64) [3]float64 {
	return [3]float64{r, g, b}
}

// feq compares 2 floats up to the 5th decimal place.
func feq(f1, f2 float64) bool {
	const accuracy = 1e-5
	return math.Abs(f1-f2) <= accuracy
}
