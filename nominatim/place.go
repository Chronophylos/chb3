package nominatim

import (
	"net/url"
	"strconv"
)

type Place struct {
	Type     string
	Category string
	Name     string

	Lat float64
	Lon float64

	URL string
}

func newPlaceFromAPI(p apiPlace) *Place {
	url, _ := url.Parse("https://www.google.com/maps/search/" + p.Name)

	place := &Place{
		Type:     p.Type,
		Category: p.Category,
		Name:     p.Name,
		URL:      url.String(),
	}
	place.Lat, _ = strconv.ParseFloat(p.Lat, 64)
	place.Lon, _ = strconv.ParseFloat(p.Lon, 64)

	return place
}

type apiPlace struct {
	ID      int    `json:"place_id"`
	License string `json:"license"`

	Lat string `json:"lat"`
	Lon string `json:"lon"`

	Category string `json:"category"`
	Type     string `json:"type"`

	Name string `json:"display_name"`
}
