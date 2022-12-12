package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/diamondburned/solar"
)

var (
	latitude  = 0.0
	longitude = solar.LocalLongitude()
	lowTemp   = float64(solar.DefaultLowTemperature)
	highTemp  = float64(solar.DefaultHighTemperature)
	tformat   = "15:04:05"
	tnow      = time.Now().Unix()
	useIPLoc  = false
	printJSON = false
)

func main() {
	flag.Float64Var(&latitude, "lat", latitude, "latitude")
	flag.Float64Var(&longitude, "long", longitude, "longitude, optional")
	flag.Float64Var(&lowTemp, "lo", lowTemp, "lowest temperature in Kelvin")
	flag.Float64Var(&highTemp, "hi", highTemp, "highest temperature in Kelvin")
	flag.StringVar(&tformat, "t", tformat, "time format")
	flag.Int64Var(&tnow, "now", tnow, "current time in Unix seconds")
	flag.BoolVar(&printJSON, "j", printJSON, "print JSON instead of human-readable")
	flag.BoolVar(&useIPLoc, "ip", useIPLoc, "use IP location instead of coordinates")
	flag.Parse()

	var geocodeResponse *geocodeResponse
	if useIPLoc {
		ip, err := myIP()
		if err != nil {
			log.Fatal("cannot get public IP address: ", err)
		}

		geocodeResponse, err = geocode(ip)
		if err != nil {
			log.Fatalln("cannot geolocate from public IP:", err)
		}

		latitude = geocodeResponse.Latitude
		longitude = geocodeResponse.Longitude
	}

	lo := solar.Temperature(lowTemp)
	hi := solar.Temperature(highTemp)
	temp, sun := solar.CalculateTemperature(time.Unix(tnow, 0), latitude, longitude, lo, hi)

	r := results{
		Latitude:    latitude,
		Longitude:   longitude,
		Geocode:     geocodeResponse,
		Temperature: temp,
		Sun:         sun,
	}

	if printJSON {
		r.PrintJSON(os.Stdout)
	} else {
		r.PrintText(os.Stdout)
	}
}

type results struct {
	Latitude    float64           `json:"latitude"`
	Longitude   float64           `json:"longitude"`
	Geocode     *geocodeResponse  `json:"geocode,omitempty"`
	Temperature solar.Temperature `json:"temperature"`
	Sun         solar.Sun         `json:"sun"`
}

func (r results) PrintText(w io.Writer) {
	printlnf := func(f string, v ...interface{}) {
		fmt.Fprintln(w, fmt.Sprintf(f, v...))
	}
	printTime := func(name string, t time.Time) {
		if !t.IsZero() {
			fmt.Fprintf(w, "%s: %s\n", name, t.Format(tformat))
		}
	}

	printlnf("latitude: %g", r.Latitude)
	printlnf("longitude: %g", r.Longitude)
	if r.Geocode != nil {
		printlnf("location: %s, %s", r.Geocode.City, r.Geocode.Region)
	}
	printlnf("sun condition: %s", r.Sun.Condition)
	printTime("dawn time", r.Sun.Dawn)
	printTime("sunrise time", r.Sun.Sunrise)
	printTime("sunset time", r.Sun.Sunset)
	printTime("dusk time", r.Sun.Dusk)
	printlnf("color temperature: %.0fK", r.Temperature)
}

func (r results) PrintJSON(w io.Writer) {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	if err := e.Encode(r); err != nil {
		log.Panicln("cannot encode JSON:", err)
	}
}

func myIP() (string, error) {
	r, err := http.Get("https://ifconfig.me")
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code %d", r.StatusCode)
	}

	v, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	return string(v), nil
}

type geocodeResponse struct {
	City      string  `json:"city"`
	Province  string  `json:"prov"`
	Region    string  `json:"region"`
	Country   string  `json:"country"`
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
