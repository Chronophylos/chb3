package openweather

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentWeatherByName(t *testing.T) {

	ow := NewOpenWeatherClient(os.Getenv("OPENWEATHERMAP_APPID"))

	t.Run("existing city", func(t *testing.T) {
		assert := assert.New(t)

		got, err := ow.GetCurrentWeatherByName("London")

		if !assert.NoError(err) {
			t.FailNow()
		}
		assert.Equal("London", got.City.Name)
		assert.Equal("GB", got.City.Country)
		assert.Equal(51.51, got.Position.Latitude)
		assert.Equal(-0.13, got.Position.Longitude)
	})

	t.Run("nonexisting city", func(t *testing.T) {
		assert := assert.New(t)

		_, err := ow.GetCurrentWeatherByName("calu321")

		assert.EqualError(err, "OpenWeather API returned an error with code 404: city not found")
	})

}

func TestGetWeatherForecastByName(t *testing.T) {
	assert := assert.New(t)

	ow := NewOpenWeatherClient(os.Getenv("OPENWEATHERMAP_APPID"))
	got, err := ow.GetWeatherForecastByName("London")

	if !assert.NoError(err) {
		t.FailNow()
	}

	for _, w := range got {
		assert.Equal("London", w.City.Name)
		assert.Equal("GB", w.City.Country)
		assert.Equal(51.5073, w.Position.Latitude)
		assert.Equal(-0.1277, w.Position.Longitude)
	}
}
