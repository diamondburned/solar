# solar

A library for calculating the sunset/sunrise time from a given location, as well
as a function to calculate the whitepoint.

It is a port of [~kennylevinsen/wlsunset][wlsunset]'s color math files, which is
an implementation of the algorithm described in the [General Solar Position
Calculations][solareqns] page from [NOAA][noaa].

[wlsunset]: https://git.sr.ht/~kennylevinsen/wlsunset
[solareqns]: https://www.esrl.noaa.gov/gmd/grad/solcalc/solareqns.PDF
[noaa]: https://www.noaa.gov/

## CLI

solar provides a CLI application that prints basic information about the sun
conditions and the calculated color temperature. It can also lookup the
longitude and latitude from the given location using the [geocode][geocode] API.

[geocode]: https://geocode.xyz/api

Example usage:

```
―❤―▶ go run ./cmd/solar/ -a 'Los Angeles' -t 'Mon Jan 2 15:04:05 MST 2006'
location: Los Angeles, US
latitude: 34.04014
longitude: -118.29757
sun condition: normal sun
dawn time: Mon Nov 8 05:05:59 PST 2021
sunrise time: Mon Nov 8 05:48:08 PST 2021
sunset time: Mon Nov 8 15:56:03 PST 2021
dusk time: Mon Nov 8 16:38:13 PST 2021
current color temperature: 4727K
```

If the user wishes to manually set the latitude and longitude, they can do so
using certain flags. If the longitude is not set, it'll be estimated from the
system's timezone. Depending on where you're at, this might just be enough.

```
―❤―▶ go run ./cmd/solar/ -lat 34.1 -t 'Mon Jan 2 15:04:05 MST 2006'
latitude: 34.1
longitude: -120
sun condition: normal sun
dawn time: Mon Nov 8 05:35:53 PST 2021
sunrise time: Mon Nov 8 06:18:05 PST 2021
sunset time: Mon Nov 8 16:25:46 PST 2021
dusk time: Mon Nov 8 17:07:58 PST 2021
current color temperature: 6330K
```

For more information, see the `-h` flag.
