// Package solar provides functions for calculating the sunset/sunrise time
// based on the location, as well as a whitepoint calculation function.
package solar

import (
	"fmt"
	"math"
	"time"
)

// Below code are primarily ported directly from this C code:
// https://git.sr.ht/~kennylevinsen/wlsunset/tree/master/item/color_math.c

// Constants in radian, converted from degrees.
const (
	startTwilight = (90.833 + 6) * math.Pi / 180
	endTwilight   = (90.833 - 3) * math.Pi / 180
)

func degrees(rad float64) float64 { return rad * 180 / math.Pi }
func radians(deg float64) float64 { return deg * math.Pi / 180 }

// SunCondition describes the conditioon of the Sun.
type SunCondition uint8

const (
	NormalSun SunCondition = iota
	MidnightSun
	PolarNightSun
	sunConditionMax
)

// String formats SunCondition into all-lower-cased English strings.
func (c SunCondition) String() string {
	switch c {
	case NormalSun:
		return "normal sun"
	case MidnightSun:
		return "midnight sun"
	case PolarNightSun:
		return "polar night sun"
	default:
		return fmt.Sprintf("SunCondition(%d)", c)
	}
}

// Sun describes the times for various positions of the sun. The dates of the
// timestamps will be whatever the date that was given to CalculateSun.
type Sun struct {
	Dawn    time.Time
	Sunrise time.Time
	Sunset  time.Time
	Dusk    time.Time

	Condition SunCondition
}

// String formats Sun into a human-readable one-lined string.
func (s Sun) String() string {
	return fmt.Sprintf(
		"dawn at %s, sunrise at %s, sunset at %s, dusk at %s, condition: %s",
		sclock(s.Dawn), sclock(s.Sunrise), sclock(s.Sunset), sclock(s.Dusk), s.Condition,
	)
}

const sclockf = "15:04:05"

func sclock(t time.Time) string {
	return t.Format(sclockf)
}

func daysInYear(t time.Time) int {
	// Get the January 1st time for the next year, then subtract 1ns. That'll
	// take us back to right before New Year, which is the final day of the
	// year.
	jan1 := time.Date(t.Year()+1, time.January, 1, 0, 0, 0, 0, t.Location())
	jan1 = jan1.Add(-1)
	return jan1.YearDay()
}

func dateOrbitAngle(t time.Time) float64 {
	return (2.0 * math.Pi / float64(daysInYear(t))) * float64(t.YearDay())
}

// equationOfTime calculates the equation of time (eqtime) from the given orbit
// angle (fractional year).
func equationOfTime(orbitAngle float64) float64 {
	// https://www.esrl.noaa.gov/gmd/grad/solcalc/solareqns.PDF
	return 4 * (0.000075 +
		0.001868*math.Cos(orbitAngle) -
		0.032077*math.Sin(orbitAngle) -
		0.014615*math.Cos(2*orbitAngle) -
		0.040849*math.Sin(2*orbitAngle))
}

func sunDeclination(orbitAngle float64) float64 {
	// https://www.esrl.noaa.gov/gmd/grad/solcalc/solareqns.PDF
	return 0.006918 -
		0.399912*math.Cos(orbitAngle) +
		0.070257*math.Sin(orbitAngle) -
		0.006758*math.Cos(2*orbitAngle) +
		0.000907*math.Sin(2*orbitAngle) -
		0.002697*math.Cos(3*orbitAngle) +
		0.001480*math.Sin(3*orbitAngle)
}

func sunHourAngle(latitude, declination, targetSun float64) float64 {
	// https://www.esrl.noaa.gov/gmd/grad/solcalc/solareqns.PDF
	return math.Acos(math.Cos(targetSun)/
		math.Cos(latitude)*math.Cos(declination) -
		math.Tan(latitude)*math.Tan(declination))
}

// hourAngleToSecondsOffset calculates the seconds offset from the given hour
// angle and the equation of time.
func hourAngleToSecondsOffset(hourAngle, eqtime float64) float64 {
	// https://www.esrl.noaa.gov/gmd/grad/solcalc/solareqns.PDF
	// The results of the inner math is in minute radians, so we convert it to
	// minute degrees before multiplying by 60 (seconds a minute).
	return degrees((4.0*math.Pi - 4*hourAngle - eqtime) * 60)
}

func timeWithSeconds(t time.Time, secs float64) time.Time {
	s, ns := math.Modf(secs)

	// Calculate the offset from t to get the time the day started relative to
	// this current timezone independently of DST.
	toffset := 0 +
		time.Duration(t.Hour())*time.Hour +
		time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second +
		time.Duration(t.Nanosecond())

	t = t.Add(-toffset)
	t = t.Add(time.Duration(s) * time.Second)
	t = t.Add(time.Duration(ns * float64(time.Second)))

	return t
}

func calcCondition(latitudeRad, sunDeclination float64) SunCondition {
	signLat := math.Signbit(latitudeRad)
	signDecl := math.Signbit(sunDeclination)
	if signLat == signDecl {
		return MidnightSun
	}
	return PolarNightSun
}

// CalculateSun calculates the times for various positions of the sun as well as
// its condition at the given time and latitude.
func CalculateSun(t time.Time, latitudeDeg float64) Sun {
	latitudeRad := radians(latitudeDeg)

	orbitAngle := dateOrbitAngle(t)
	decl := sunDeclination(orbitAngle)
	eqtime := equationOfTime(orbitAngle)

	haTwilight := sunHourAngle(latitudeRad, decl, startTwilight)
	haDaylight := sunHourAngle(latitudeRad, decl, endTwilight)

	sun := Sun{
		Dawn:    timeWithSeconds(t, hourAngleToSecondsOffset(+math.Abs(haTwilight), eqtime)),
		Dusk:    timeWithSeconds(t, hourAngleToSecondsOffset(-math.Abs(haTwilight), eqtime)),
		Sunrise: timeWithSeconds(t, hourAngleToSecondsOffset(+math.Abs(haDaylight), eqtime)),
		Sunset:  timeWithSeconds(t, hourAngleToSecondsOffset(-math.Abs(haDaylight), eqtime)),
	}

	if math.IsNaN(haTwilight) || math.IsNaN(haDaylight) {
		sun.Condition = calcCondition(latitudeRad, decl)
	} else {
		sun.Condition = NormalSun
	}

	return sun
}
