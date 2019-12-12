package nominatim

import "strconv"

type Place struct {
	Type     string
	Category string
	Name     string

	Lat float64
	Lon float64

	URL string
}

func newPlaceFromAPI(p apiPlace) *Place {
	return &Place{
		Type:     p.Type,
		Category: p.Category,
		Name:     p.Name,
		Lat:      p.Lat,
		Lon:      p.Lon,
		URL:      "https://www.google.com/maps/search/" + strconv.FormatFloat(p.Lat, 'g', -1, 64) + "," + strconv.FormatFloat(p.Lon, 'g', -1, 64),
	}
}

type apiPlace struct {
	ID      int    `json:"place_id"`
	License string `json:"license"`

	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`

	Category string `json:"category"`
	Type     string `json:"type"`

	Name string `json:"display_name"`
}
