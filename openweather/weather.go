package openweather

import "time"

// Unit Systems available
const (
	StandardSystem = iota + 1
	MetricSystem
	ImperialSystem
)

type Weather struct {
	UnitSystem int

	Sunrise    time.Time
	Sunset     time.Time
	Humidity   int
	Pressure   int
	CloudCover int

	Temperature struct {
		Current float64
		Min     float64
		Max     float64
	}

	City struct {
		Name    string
		ID      int
		Country string
	}

	Position struct {
		Latitude  float64
		Longitude float64
	}

	Conditions []WeatherCondition

	Snow struct {
		LastHour   float64
		Last3Hours float64
	}

	Rain struct {
		LastHour   float64
		Last3Hours float64
	}

	Wind struct {
		Direction string
		Degree    int
		Speed     float64
	}
}

type WeatherCondition struct {
	Description string
	Icon        string
	ID          int
	Group       string
}

// NewWeatherFromWeatherDataResponse maps a weatherDataResponse to Weather.
// Everything is optinal and some values may get transformed or processed.
// TODO: make everything optional
func NewWeatherFromWeatherDataResponse(resp weatherDataResponse) (*Weather, error) {
	w := &Weather{
		Sunrise:    time.Unix(resp.Sys.Sunrise, 0),
		Sunset:     time.Unix(resp.Sys.Sunset, 0),
		Humidity:   resp.Main.Humidity,
		Pressure:   resp.Main.Pressure,
		CloudCover: resp.Clouds.All,
		Conditions: []WeatherCondition{},
	}

	w.Temperature.Current = resp.Main.Temp
	w.Temperature.Max = resp.Main.TempMax
	w.Temperature.Min = resp.Main.TempMin

	w.Position.Latitude = resp.Coord.Lat
	w.Position.Longitude = resp.Coord.Lon

	w.Rain.LastHour = resp.Rain.LastHour
	w.Rain.Last3Hours = resp.Rain.Last3Hours

	w.Snow.LastHour = resp.Snow.LastHour
	w.Snow.Last3Hours = resp.Snow.Last3Hours

	w.Wind.Degree = resp.Wind.Deg
	w.Wind.Speed = resp.Wind.Speed

	direction, err := DegreeToCompass(resp.Wind.Deg)
	if err != nil {
		return w, err
	}
	w.Wind.Direction = direction

	for _, c := range resp.WeatherConditions {
		w.Conditions = append(w.Conditions, WeatherCondition{
			Description: c.Description,
			Icon:        c.Icon,
			ID:          c.ID,
			Group:       c.Main,
		})
	}

	return w, nil
}
