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
	address   = ""
	useIPLoc  = false
	printJSON = false
)

func main() {
	flag.Float64Var(&latitude, "lat", latitude, "latitude")
	flag.Float64Var(&longitude, "long", longitude, "longitude, optional")
	flag.Float64Var(&lowTemp, "lo", lowTemp, "lowest temperature in Kelvin")
	flag.Float64Var(&highTemp, "hi", highTemp, "highest temperature in Kelvin")
	flag.StringVar(&tformat, "t", tformat, "time format")
	flag.StringVar(&address, "a", address, "address to geocode, takes precedence over --lat, --long and --ip")
	flag.Int64Var(&tnow, "now", tnow, "current time in Unix seconds")
	flag.BoolVar(&printJSON, "j", printJSON, "print JSON instead of human-readable")
	flag.BoolVar(&useIPLoc, "ip", useIPLoc, "use IP location instead of coordinates")
	flag.Parse()

	var geocodeResponse *geocodeResponse
	var geocodeResults *GeocodeResults

	if useIPLoc || address != "" {
		var err error
		geocodeInput := address

		switch {
		case useIPLoc:
			geocodeInput, err = myIP()
			if err != nil {
				log.Fatal("cannot get public IP address: ", err)
			}
		case address != "":
			geocodeInput = address
		}

		geocodeResponse, err = geocode(geocodeInput)
		if err != nil {
			log.Fatalln("cannot geolocate from public IP:", err)
		}

		latitude = geocodeResponse.Latitude
		longitude = geocodeResponse.Longitude

		geocodeResults = &GeocodeResults{
			City:    geocodeResponse.City,
			Country: geocodeResponse.Country,
		}
	}

	lo := solar.Temperature(lowTemp)
	hi := solar.Temperature(highTemp)
	temp, sun := solar.CalculateTemperature(time.Unix(tnow, 0), latitude, longitude, lo, hi)

	r := Results{
		Latitude:    latitude,
		Longitude:   longitude,
		Geocode:     geocodeResults,
		Temperature: temp,
		Sun: SunResults{
			Dawn:      sun.Dawn,
			Sunrise:   sun.Sunrise,
			Sunset:    sun.Sunset,
			Dusk:      sun.Dusk,
			Condition: sun.Condition.String(),
		},
	}

	if printJSON {
		r.PrintJSON(os.Stdout)
	} else {
		r.PrintText(os.Stdout)
	}
}

type Results struct {
	Latitude    float64           `json:"latitude"`
	Longitude   float64           `json:"longitude"`
	Geocode     *GeocodeResults   `json:"geocode,omitempty"`
	Temperature solar.Temperature `json:"temperature"`
	Sun         SunResults        `json:"sun"`
}

type SunResults struct {
	Dawn      time.Time `json:"dawn,omitempty"`
	Sunrise   time.Time `json:"sunrise,omitempty"`
	Sunset    time.Time `json:"sunset,omitempty"`
	Dusk      time.Time `json:"dusk,omitempty"`
	Condition string    `json:"condition"`
}

type GeocodeResults struct {
	City    string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
}

func (r Results) PrintText(w io.Writer) {
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
		printlnf("location: %s, %s", r.Geocode.City, r.Geocode.Country)
	}
	printlnf("sun condition: %s", r.Sun.Condition)
	printTime("dawn time", r.Sun.Dawn)
	printTime("sunrise time", r.Sun.Sunrise)
	printTime("sunset time", r.Sun.Sunset)
	printTime("dusk time", r.Sun.Dusk)
	printlnf("color temperature: %.0fK", r.Temperature)
}

func (r Results) PrintJSON(w io.Writer) {
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
	City      string
	Country   string
	Longitude float64
	Latitude  float64
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

	var resp struct {
		City     string `json:"city,omitempty"`
		Country  string `json:"country,omitempty"`
		Standard struct {
			City    string `json:"city"`
			Country string `json:"countryname"`
		} `json:"standard"`
		Longitude float64 `json:"longt,string"`
		Latitude  float64 `json:"latt,string"`
	}

	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("cannot decode JSON from geocode.xyz: %w", err)
	}

	if resp.Standard.Country != "" {
		resp.City = resp.Standard.City
		resp.Country = resp.Standard.Country
	}

	return &geocodeResponse{
		City:      resp.City,
		Country:   resp.Country,
		Longitude: resp.Longitude,
		Latitude:  resp.Latitude,
	}, nil
}
