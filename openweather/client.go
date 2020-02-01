package openweather

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type weatherDataResponse struct {
	Clouds struct {
		All int `json:"all"` // Cloudiness in %
	} `json:"clouds"`

	Coord struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"coord"`

	Main struct {
		Humidity int     `json:"humidity"`
		Pressure int     `json:"pressure"`
		Temp     float64 `json:"temp"`
		TempMax  float64 `json:"temp_max"`
		TempMin  float64 `json:"temp_min"`
	} `json:"main"`

	Sys struct {
		Country string `json:"country"`
		ID      int    `json:"id"`
		Sunrise int64  `json:"sunrise"`
		Sunset  int64  `json:"sunset"`
	} `json:"sys"`

	Rain struct {
		LastHour   float64 `json:"1h"`
		Last3Hours float64 `json:"3h"`
	} `json:"rain"`

	Snow struct {
		LastHour   float64 `json:"1h"`
		Last3Hours float64 `json:"3h"`
	} `json:"snow"`

	WeatherConditions []struct {
		Description string `json:"description"`
		Icon        string `json:"icon"`
		ID          int    `json:"id"`
		Main        string `json:"main"`
	} `json:"weather"`

	Wind struct {
		Deg   int     `json:"deg"`
		Speed float64 `json:"speed"`
	} `json:"wind"`

	Time string `json:"dt_txt"`
}

type currentWeatherResponse struct {
	CityID   int    `json:"id"`
	CityName string `json:"name"`

	weatherDataResponse

	Code    interface{} `json:"cod"`
	Message string      `json:"message"`
}

func (r currentWeatherResponse) GetCode() int {
	switch v := r.Code.(type) {
	case int:
		return v
	case string:
		code, _ := strconv.Atoi(v)
		return code
	case float64:
		return int(v)
	default:
		panic(fmt.Sprintf("code is neither int nor string but: %T", v))
	}
}

func (r currentWeatherResponse) GetMessage() string {
	return r.Message
}

type forecastWeatherResponse struct {
	City struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Country string `json:"country"`

		Coord struct {
			Latitude  float64 `json:"lat"`
			Longitude float64 `json:"lon"`
		} `json:"coord"`
	} `json:"city"`

	List []weatherDataResponse `json:"list"`

	Code    string `json:"cod"`
	Message int    `json:"message"`
}

func (r forecastWeatherResponse) GetCode() int {
	code, _ := strconv.Atoi(r.Code)
	return code
}

func (r forecastWeatherResponse) GetMessage() string {
	return fmt.Sprintf("%v", r.Message)
}

type apiResponse interface {
	GetCode() int
	GetMessage() string
}

type Client struct {
	httpClient *http.Client
	appid      string
	userAgent  string
}

// NewClient creates a new Client
func NewClient(appid, userAgent string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		appid:     appid,
		userAgent: userAgent,
	}
}

func (c *Client) request(url string, params url.Values) ([]byte, error) {
	params.Set("appid", c.appid)
	params.Set("lang", "de")
	params.Set("units", "metric")

	url = url + "?" + params.Encode()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte{}, fmt.Errorf("could not create request: %v", err)
	}

	req.Header.Add("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("could not perform request: %v", err)
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("could not read bytes from response: %v", err)
	}

	return bytes, nil
}

func checkResponse(resp apiResponse) error {
	if resp.GetCode() != http.StatusOK {
		if resp.GetCode() == http.StatusTooManyRequests {
			return errors.New("got ratelimited from OpenWeather API")
		}
		return fmt.Errorf("OpenWeather API returned an error with code %d: %s", resp.GetCode(), resp.GetMessage())
	}

	return nil
}

func (c *Client) GetCurrentWeatherByName(name string) (*Weather, error) {
	var weather *Weather
	var weatherResp currentWeatherResponse

	params := url.Values{}
	params.Set("q", name)

	bytes, err := c.request("https://api.openweathermap.org/data/2.5/weather", params)
	if err != nil {
		return weather, err
	}

	if err = json.Unmarshal(bytes, &weatherResp); err != nil {
		return weather, fmt.Errorf("could not unmarshal bytes: %v", err)
	}

	if err = checkResponse(weatherResp); err != nil {
		return weather, err
	}

	weather, err = NewWeatherFromWeatherDataResponse(weatherResp.weatherDataResponse)
	if err != nil {
		return weather, err
	}

	weather.UnitSystem = MetricSystem

	weather.City.Name = weatherResp.CityName
	weather.City.ID = weatherResp.CityID
	weather.City.Country = weatherResp.Sys.Country

	return weather, nil
}

func (c *Client) GetWeatherForecastByName(name string) ([]*Weather, error) {
	var weatherList []*Weather
	var weatherResp forecastWeatherResponse

	params := url.Values{}
	params.Set("q", name)

	bytes, err := c.request("https://api.openweathermap.org/data/2.5/forecast", params)
	if err != nil {
		return weatherList, err
	}

	if err = json.Unmarshal(bytes, &weatherResp); err != nil {
		return weatherList, fmt.Errorf("could not unmarshal bytes: %v", err)
	}

	if err = checkResponse(weatherResp); err != nil {
		return weatherList, err
	}

	for _, singleWeatherResponse := range weatherResp.List {
		weather, err := NewWeatherFromWeatherDataResponse(singleWeatherResponse)
		if err != nil {
			return weatherList, err
		}

		weather.UnitSystem = MetricSystem

		weather.City.Name = weatherResp.City.Name
		weather.City.ID = weatherResp.City.ID
		weather.City.Country = weatherResp.City.Country

		weather.Position.Latitude = weatherResp.City.Coord.Latitude
		weather.Position.Longitude = weatherResp.City.Coord.Longitude

		weatherList = append(weatherList, weather)
	}

	return weatherList, nil
}
