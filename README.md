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
―❤―▶ go run ./cmd/solar/ --ip
latitude: 34.2729
longitude: -117.22828
location: SAN BERNARDINO, United States of America
sun condition: normal sun
dawn time: 05:42:23
sunrise time: 06:23:44
sunset time: 15:49:22
dusk time: 16:30:43
color temperature: 4000K
```

```
―❤―▶ go run ./cmd/solar/ -a 'Los Angeles' -t 'Mon Jan 2 15:04:05 MST 2006'
latitude: 34.06221
longitude: -118.3367
location: Los Angeles, United States of America
sun condition: normal sun
dawn time: Sun Dec 11 06:27:54 PST 2022
sunrise time: Sun Dec 11 07:09:07 PST 2022
sunset time: Sun Dec 11 16:35:55 PST 2022
dusk time: Sun Dec 11 17:17:08 PST 2022
color temperature: 4000K
```

If the user wishes to manually set the latitude and longitude, they can do so
using certain flags. If the longitude is not set, it'll be estimated from the
system's timezone. Depending on where you're at, this might just be enough.

```
―❤―▶ go run ./cmd/solar/ --lat 34.1 -t 'Mon Jan 2 15:04:05 MST 2006'
latitude: 34.1
longitude: -120
sun condition: normal sun
dawn time: Sun Dec 11 06:06:47 PST 2022
sunrise time: Sun Dec 11 06:48:02 PST 2022
sunset time: Sun Dec 11 16:14:36 PST 2022
dusk time: Sun Dec 11 16:55:51 PST 2022
color temperature: 4000K
```

If the program is consumed in a script, it's best to use `-j` with something
like `jq`:

```
―❤―▶ go run ./cmd/solar/ --lat 34.1 -t 'Mon Jan 2 15:04:05 MST 2006' -j | jq -r .sun.sunset
2022-12-11T16:14:36.985451921-08:00
```

For more information, see the `-h` flag.
