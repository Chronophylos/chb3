package actions

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chronophylos/chb3/openweather"
)

type weatherAction struct {
	options *Options
}

func newWeatherAction1() *weatherAction {
	return &weatherAction{
		options: &Options{
			Name: "weather",
			Re:   regexp.MustCompile(`(?i)^~weather (.*)`),
		},
	}
}

func newWeatherAction2() *weatherAction {
	return &weatherAction{
		options: &Options{
			Name: "weather",
			Re:   regexp.MustCompile(`(?i)^wie ist das wetter in (.*)\?`),
		},
	}
}

func (a weatherAction) GetOptions() *Options {
	return a.options
}

func (a weatherAction) Run(e *Event) error {
	where := e.Match[1]

	e.Log.Info().
		Str("where", where).
		Msg("Checking the weather")

	err, weatherMessage := getWeather(e.Weather, where)
	if err != nil {
		return nil
	}

	e.Say(weatherMessage)

	return nil
}

const weatherText = "Das aktuelle Wetter für %s, %s: %s bei %.1f°C. Der Wind kommt aus %s mit %.1fm/s bei einer Luftfeuchtigkeit von %d%%. Die Wettervorhersagen für morgen: %s bei %.1f°C bis %.1f°C."

func getWeather(c *openweather.OpenWeatherClient, where string) (error, string) {
	currentWeather, err := c.GetCurrentWeatherByName(where)
	if err != nil {
		if err.Error() == "OpenWeather API returned an error with code 404: city not found" {
			return nil, fmt.Sprintf("Ich kann %s nicht finden", where)
		}
		return err, ""
	}

	conditions := []string{}
	for _, condition := range currentWeather.Conditions {
		conditions = append(conditions, condition.Description)
	}

	currentCondition := strings.Join(conditions, " und ")

	weatherForecast, err := c.GetWeatherForecastByName(where)
	if err != nil {
		return err, ""
	}

	var tomorrowsWeather *openweather.Weather
	year, month, day := time.Now().Date()
	tomorrow := time.Date(year, month, day+1, 12, 0, 0, 0, time.UTC)
	for _, weather := range weatherForecast {
		if weather.Time == tomorrow {
			tomorrowsWeather = weather
			break
		}
	}

	conditions = []string{}
	for _, condition := range tomorrowsWeather.Conditions {
		conditions = append(conditions, condition.Description)
	}

	tomorrowsConditions := strings.Join(conditions, " und ")

	return nil, fmt.Sprintf(weatherText,
		currentWeather.City.Name,
		currentWeather.City.Country,
		currentCondition,
		currentWeather.Temperature.Current,
		currentWeather.Wind.Direction,
		currentWeather.Wind.Speed,
		currentWeather.Humidity,
		tomorrowsConditions,
		tomorrowsWeather.Temperature.Min,
		tomorrowsWeather.Temperature.Max,
	)
}
