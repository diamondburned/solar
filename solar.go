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

// SunCondition describes the condition of the Sun.
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

	// Condition determines the validity of the above times. The times are only
	// all valid if the condition is normal (NormalSun).
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

// Temperature is the type for the color temperature in Kelvin.
type Temperature float64

const (
	DefaultLowTemperature  Temperature = 4000 // K
	DefaultHighTemperature Temperature = 6500 // K
)

// CurrentTemperature calls CalculateTemperature with time.Now().
func CurrentTemperature(lat, long float64, lo, hi Temperature) Temperature {
	return CalculateTemperature(time.Now(), lat, long, lo, hi)
}

// WatchCurrentTemperature watches the
// func WatchCurrentTemperature(lat, long float64, lo, hi Temperature) <-chan Temperature {}

// CalculateTemperature calculates the color temperature for the given time. The
// given latitude must be in degrees. The given lo, hi values determine the
// minimum and maximum temperatures.
func CalculateTemperature(t time.Time, lat, long float64, lo, hi Temperature) Temperature {
	current := CalculateSun(t, lat, long)

	switch current.Condition {
	case NormalSun:
		return calcTempNormal(t, current, lo, hi)
	case MidnightSun:
		// Need yesterday's sun condition to determine if we should transition
		// from a normal sun to a midnight sun (always daytime).
		yesterday := CalculateSun(yesterday(t), lat, long)
		if yesterday.Condition == NormalSun && t.Before(current.Sunrise) {
			return calcTempNormal(t, current, lo, hi)
		}
		// Yesterday was not normal sun, so probably polar night or midnight.
		// Keep high.
		return hi
	case PolarNightSun:
		// wlsunset code directly transitions this to low.
		return lo
	default:
		panic("unreachable: unknown sun condition " + current.Condition.String())
	}
}

// // NextTransitionTime calculates the next time instant that the color
// // transitioning will begin.
// func NextTransitionTime(t time.Time, lat, long float64) time.Time {
// 	current := CalculateSun(t, lat, long)
// }

func calcTempNormal(t time.Time, sun Sun, lo, hi Temperature) Temperature {
	switch {
	case t.Before(sun.Dawn):
		return lo
	case t.Before(sun.Sunrise):
		return interpTemp(t, sun.Dawn, sun.Sunrise, lo, hi)
	case t.Before(sun.Sunset):
		return hi
	case t.Before(sun.Dusk):
		return interpTemp(t, sun.Sunset, sun.Dusk, hi, lo)
	default:
		return lo
	}
}

// interpTemp interpolates the temperature to the given time instant and time
// range.
func interpTemp(t, start, stop time.Time, Tstart, Tstop Temperature) Temperature {
	if Tstart == Tstop {
		return Tstop
	}

	timePos := float64(t.Sub(start)) / float64(stop.Sub(start))
	timePos = clamp(timePos)

	tempPos := float64(Tstop-Tstart) * timePos
	return Tstart + Temperature(tempPos)
}

// yesterday returns the same time but yesterday (24 hours ago).
func yesterday(t time.Time) time.Time {
	return t.Add(-24 * time.Hour)
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

// longitudeTimeOffset calculates the longitude offset in seconds.
func longitudeTimeOffset(long float64) float64 {
	const halfDay = 43200
	return long * halfDay / math.Pi
}

// LocalLongitude estimates the longitude using the system timezone.
func LocalLongitude() float64 {
	return TimeLongitude(time.Now())
}

// TimeLongitude estimates the longitude from the given time instant. It uses
// the timezone to estimate.
func TimeLongitude(t time.Time) float64 {
	_, offset := t.Zone()
	if t.IsDST() {
		// DST sets the clock forward 1 hour, so we shift it back.
		offset -= 1 * 60 * 60
	}

	// https://www.timeanddate.com/time/current-number-time-zones.html

	// Offset is in seconds, and the formula needs hour, so we convert it, then
	// multiply.
	return float64(offset) / 60 / 60 * 15
}

// timeTruncateDayLongitude calls timeTruncateDay on the given time instant,
// then adds the longitude time offset.
func timeTruncateDayLongitude(t time.Time, long float64) time.Time {
	offset := longitudeTimeOffset(long)
	// Round the offset to an hour. The offset is, for some reason, longer than
	// an hour. For example, given the longitude of Los Angeles, we get a ~3.5
	// hours offset shifting back, which is unusual. Doing it this way gives us
	// a much more accurate result, however.
	//
	// For example, before giving the longitude, the calculated sunrise time was
	// 6:40AM in the tests, while Google reported 6:19AM. With the right
	// longitude, the calculated time is 6:10AM, which is much closer.
	//
	// The unexpected offset could be attributed to both the halfDay constant in
	// wlsunset and the fact that time.Time already includes timezone
	// information, while time_t operates on Unix epoch (which doesn't have a
	// timezone).
	offset = math.Mod(offset, 1*60*60)

	t = timeTruncateDay(t)
	t = timeAddSeconds(t, offset)

	return t
}

// timeTruncateDay truncates the given time to the start of day using the
// current time instant's timezone.
func timeTruncateDay(t time.Time) time.Time {
	// Calculate the offset from t to get the time the day started relative to
	// this current timezone independently of DST.
	toffset := 0 +
		time.Duration(t.Hour())*time.Hour +
		time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second +
		time.Duration(t.Nanosecond())

	t = t.Add(-toffset)
	return t
}

// timeAddSeconds adds the given seconds in float64 to the given time instant.
// If secs is NaN, then t is returned.
func timeAddSeconds(t time.Time, secs float64) time.Time {
	if math.IsNaN(secs) {
		return time.Time{}
	}

	s, ns := math.Modf(secs)
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
// its condition at the given time and latitude. The given latitude and
// longitude must be in degrees.
//
// The given longitude is only used to improve the accuracy of the result. The
// values of the given longitude will vary the results by +-1 hour.
//
// If the returned Sun data has a non-normal condition, that is, if it's
// midnight sun or polar night sun, then some of the time values may be zero.
func CalculateSun(t time.Time, lat, long float64) Sun {
	t = timeTruncateDayLongitude(t, long)
	latitudeRad := radians(lat)

	orbitAngle := dateOrbitAngle(t)
	decl := sunDeclination(orbitAngle)
	eqtime := equationOfTime(orbitAngle)

	haTwilight := sunHourAngle(latitudeRad, decl, startTwilight)
	haDaylight := sunHourAngle(latitudeRad, decl, endTwilight)

	sun := Sun{
		Dawn:    timeAddSeconds(t, hourAngleToSecondsOffset(+math.Abs(haTwilight), eqtime)),
		Dusk:    timeAddSeconds(t, hourAngleToSecondsOffset(-math.Abs(haTwilight), eqtime)),
		Sunrise: timeAddSeconds(t, hourAngleToSecondsOffset(+math.Abs(haDaylight), eqtime)),
		Sunset:  timeAddSeconds(t, hourAngleToSecondsOffset(-math.Abs(haDaylight), eqtime)),
	}

	if math.IsNaN(haTwilight) || math.IsNaN(haDaylight) {
		sun.Condition = calcCondition(latitudeRad, decl)
	} else {
		sun.Condition = NormalSun
	}

	return sun
}

func throwf(f string, v ...interface{}) {
	panic(fmt.Sprintf(f, v...))
}

// illuminantD, or daylight locus, is a "standard illuminant" used to describe
// natural daylight. It is on this locus that D65, the whitepoint used by most
// monitors and assumed by wlsunset, is defined.
//
// This approximation is strictly speaking only well-defined between 4000K and
// 25000K, but we stretch it a bit further down for transition purposes.
//
// The function will panic if temp is outside the range [2500, 25000] in
// interval notation. Note that CalculateWhitepoint does its own clamping
// already.
func illuminantD(temp float64) (x, y float64) {
	// https://en.wikipedia.org/wiki/Standard_illuminant#Illuminant_series_D
	if temp >= 2500 && temp <= 7000 {
		x = 0.244063 +
			0.09911e3/temp +
			2.9678e6/math.Pow(temp, 2) -
			4.6070e9/math.Pow(temp, 3)
	} else if temp > 7000 && temp <= 25000 {
		x = 0.237040 +
			0.24748e3/temp +
			1.9018e6/math.Pow(temp, 2) -
			2.0064e9/math.Pow(temp, 3)
	} else {
		throwf("unreachable: temp %f out of range [2500, 25000]", temp)
	}

	y = (-3 * math.Pow(x, 2)) + (2.870 * x) - 0.275
	return
}

// planckianLocus, or black body locus, describes the color of a black body at a
// certain temperatures. This is not entirely equivalent to daylight due to
// atmospheric effects.
//
// This approximation is only valid from 1667K to 25000K. The function will
// panic if the given temperature is outside that range.
func planckianLocus(temp float64) (x, y float64) {
	// https://en.wikipedia.org/wiki/Planckian_locus#Approximation
	if temp >= 1667 && temp <= 4000 {
		x = -0.2661239e9/math.Pow(temp, 3) -
			0.2343589e6/math.Pow(temp, 2) +
			0.8776956e3/temp +
			0.179910
		if temp <= 2222 {
			y = -1.1064814*math.Pow(x, 3) -
				1.34811020*math.Pow(x, 2) +
				2.18555832*x -
				0.20219683
		} else {
			y = -0.9549476*math.Pow(x, 3) -
				1.37418593*math.Pow(x, 2) +
				2.09137015*x -
				0.16748867
		}
	} else if temp > 4000 && temp < 25000 {
		// This codepath is never hit.
		x = -3.0258469e9/math.Pow(temp, 3) +
			2.1070379e6/math.Pow(temp, 2) +
			0.2226347e3/temp +
			0.240390
		y = 3.0817580*math.Pow(x, 3) -
			5.87338670*math.Pow(x, 2) +
			3.75112997*x -
			0.37001483
	} else {
		throwf("unreachable: temp %f out of range [1667, 25000]", temp)
	}
	return
}

func srgbGamma(value, gamma float64) float64 {
	// https://en.wikipedia.org/wiki/SRGB
	if value <= 0.0031308 {
		return 12.92 * value
	} else {
		return math.Pow(1.055*value, 1.0/gamma) - 0.055
	}
}

// clamp clamps the given value between [0.0, 1.0].
func clamp(value float64) float64 {
	switch {
	case value > 1.0:
		return 1.0
	case value < 0.0:
		return 0.0
	}
	return value
}

func xyzToSRGB(x, y, z float64) (r, g, b float64) {
	// http://www.brucelindbloom.com/index.html?Eqn_RGB_XYZ_Matrix.html
	r = srgbGamma(clamp((3.2404542*x)-(1.5371385*y)-(0.4985314*z)), 2.2)
	g = srgbGamma(clamp((-0.9692660*x)+(1.8760108*y)+(0.0415560*z)), 2.2)
	b = srgbGamma(clamp((0.0556434*x)-(0.2040259*y)+(1.0572252*z)), 2.2)
	return
}

func srgbNormalize(r, g, b float64) (r1, g1, b1 float64) {
	maxw := math.Max(r, math.Max(g, b))
	r /= maxw
	g /= maxw
	b /= maxw
	return r, g, b
}

// CalculateWhitepoint calculates the whitepoint for the red, green and blue
// channels given the color temperature.
//
// The valid range for temperature is from 1667K to 25000K. Giving a value
// outside that range will make the function clamp that value. The normal white
// temperature is 6500K.
//
// The returned red, green and blue values are within [0.0, 1.0] in interval
// notation. A temperature value of 6500K will return (1.0, 1.0, 1.0) for white.
func CalculateWhitepoint(temp Temperature) (rw, gw, bw float64) {
	if temp == 6500 {
		rw = 1
		gw = 1
		bw = 1
		return
	}

	x, y := 1.0, 1.0

	switch {
	case temp >= 25000:
		x, y = illuminantD(25000)
	case temp >= 4000:
		x, y = illuminantD(float64(temp))
	case temp >= 2500:
		x1, y1 := illuminantD(float64(temp))
		x2, y2 := planckianLocus(float64(temp))
		factor := float64((4000 - temp) / 1500)
		sinefactor := (math.Cos(math.Pi*factor) + 1.0) / 2.0
		x = x1*sinefactor + x2*(1.0-sinefactor)
		y = y1*sinefactor + y2*(1.0-sinefactor)
	case temp >= 1667:
		x, y = planckianLocus(float64(temp))
	default:
		x, y = planckianLocus(1667)
	}

	z := 1.0 - x - y

	rw, gw, bw = xyzToSRGB(x, y, z)
	rw, gw, bw = srgbNormalize(rw, gw, bw)
	return
}
