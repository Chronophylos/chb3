package openweather

import (
	"errors"
)

var compassNames = [16]string{
	"N",
	"NNE",
	"NE",
	"ENE",
	"E",
	"ESE",
	"SE",
	"SSE",
	"S",
	"SSW",
	"SW",
	"WSW",
	"W",
	"WNW",
	"NW",
	"NNW",
}

// DegreeToCompass converts a direction in degrees to a direction on a compass
// e.g. 360° -> N
//      123° -> NSE
func DegreeToCompass(degree int) (string, error) {
	if degree > 360 || degree < 0 {
		return "", errors.New("degrees are out of bounds")
	}

	i := int(degree / 22 % 16)

	return compassNames[i], nil
}
