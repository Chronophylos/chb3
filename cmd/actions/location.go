package actions

import (
	"regexp"
	"strings"
)

type locationAction struct {
	options *Options
}

func newLocationAction() *locationAction {
	return &locationAction{
		options: &Options{
			Name: "location",
			Re:   regexp.MustCompile(`(?i)^wo (ist|liegt) (.*)\?+`),
		},
	}
}

func (a locationAction) GetOptions() *Options {
	return a.options
}

func (a locationAction) Run(e *Event) error {
	where := e.Match[2]

	if strings.ToLower(where) == "bielefeld" {
		e.Say("Ich kann Bielefeld nicht finden")
		return nil
	}

	place, err := e.Location.GetPlace(where)
	if err != nil {
		e.Say("Ich kann " + where + " nicht finden")
		return nil
	}

	e.Say(place.URL)

	e.Log.Info().
		Str("where", where).
		Float64("lat", place.Lat).
		Float64("lon", place.Lon).
		Msg("Getting Location")

	return nil
}
