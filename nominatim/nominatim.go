package nominatim

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
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

	var p []apiPlace
	json.Unmarshal(body, &p)

	return newPlaceFromAPI(p[0]), nil
}

func getSearchURL(location string) string {
	return "https://nominatim.openstreetmap.org/search?q=" + location + "&format=jsonv2"
}
