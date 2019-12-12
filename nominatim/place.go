package nominatim

import (
	"net/url"
	"strconv"
)

type Place struct {
	Name string

	Lat float64
	Lon float64

	URL string
}

func newPlaceFromAPI(p *apiPlace) *Place {
	url, _ := url.Parse("https://www.openstreetmap.org/" + p.Type + "/" + p.ID)

	place := &Place{
		Name: p.Name,
		URL:  url.String(),
	}
	place.Lat, _ = strconv.ParseFloat(p.Lat, 64)
	place.Lon, _ = strconv.ParseFloat(p.Lon, 64)

	return place
}

type apiPlace struct {
	License string `json:"license"`

	Lat string `json:"lat"`
	Lon string `json:"lon"`

	Category string `json:"category"`
	Type     string `json:"osm_type"`
	ID       int    `json:"osm_id"`
	Rank     int    `json:"place_rank"`

	Name string `json:"display_name"`
}

type apiPlaces []*apiPlace

func (p apiPlaces) Len() int           { return len(p) }
func (p apiPlaces) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p apiPlaces) Less(i, j int) bool { return p[i].Rank < p[j].Rank }
