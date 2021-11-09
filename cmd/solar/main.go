package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/diamondburned/solar"
)

var (
	latitude  = 0.0
	longitude = solar.LocalLongitude()
	lowTemp   = float64(solar.DefaultLowTemperature)
	highTemp  = float64(solar.DefaultHighTemperature)
	address   = ""
	tformat   = time.RFC3339Nano
)

func main() {
	flag.Float64Var(&latitude, "lat", latitude, "latitude")
	flag.Float64Var(&longitude, "long", longitude, "longitude, optional")
	flag.Float64Var(&lowTemp, "lo", lowTemp, "lowest temperature")
	flag.Float64Var(&highTemp, "hi", highTemp, "highest temperature")
	flag.StringVar(&address, "a", address, "address to geoguess using geocode.xyz, uses internet")
	flag.StringVar(&tformat, "t", tformat, "time format")
	flag.Parse()

	if address != "" {
		r, err := geocode(address)
		if err != nil {
			log.Fatalln("cannot resolve address:", err)
		}
		printlnf("location: %s, %s", r.Standard.City, r.Standard.Province)
		latitude = r.Latitude
		longitude = r.Longitude
	}

	printlnf("latitude: %g", latitude)
	printlnf("longitude: %g", longitude)

	now := time.Now()
	sun := solar.CalculateSun(now, latitude, longitude)

	printlnf("sun condition: %s", sun.Condition)
	printTime("dawn time", sun.Dawn)
	printTime("sunrise time", sun.Sunrise)
	printTime("sunset time", sun.Sunset)
	printTime("dusk time", sun.Dusk)

	lo := solar.Temperature(lowTemp)
	hi := solar.Temperature(highTemp)

	printlnf("current color temperature: %.0fK",
		solar.CalculateTemperature(now, latitude, longitude, lo, hi))
}

func printTime(name string, t time.Time) {
	if t.IsZero() {
		printlnf("%s: null", name)
	} else {
		printlnf("%s: %s", name, t.Format(tformat))
	}
}

func printlnf(f string, v ...interface{}) {
	fmt.Printf(f+"\n", v...)
}

type geocodeResponse struct {
	Standard struct {
		City     string `json:"city"`
		Province string `json:"prov"`
		Country  string `json:"countryname"`
	} `json:"standard"`
	Longitude float64 `json:"longt,string"`
	Latitude  float64 `json:"latt,string"`
}

func geocode(address string) (*geocodeResponse, error) {
	u := url.URL{
		Scheme:   "https",
		Host:     "geocode.xyz",
		Path:     address,
		RawQuery: "json=1",
	}

	r, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("cannot GET geocode.xyz: %w", err)
	}
	defer r.Body.Close()

	var resp geocodeResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("cannot decode JSON from geocode.xyz: %w", err)
	}

	return &resp, nil
}
