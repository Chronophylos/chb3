package nominatim

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sort"
	"time"
)

type Client struct {
	UserAgent string
}

func (c *Client) GetPlace(location string) (*Place, error) {
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	req, err := http.NewRequest("GET", getSearchURL(location), nil)
	if err != nil {
		return &Place{}, err
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept-Language", "de_DE")
	req.Header.Set("Referer", "irc.twitch.tv")
	req.Header.Set("DNT", "1")

	resp, err := client.Do(req)
	if err != nil {
		return &Place{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &Place{}, err
	}

	// TODO: sort places
	var p apiPlaces
	json.Unmarshal(body, &p)

	if len(p) == 0 {
		return &Place{}, errors.New("no place found")
	}

	sort.Sort(p)

	return newPlaceFromAPI(p[0]), nil
}

// TODO: parse better
func getSearchURL(location string) string {
	return "https://nominatim.openstreetmap.org/search?format=jsonv2&q=" + location
}
