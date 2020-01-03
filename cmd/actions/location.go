package actions

import "regexp"

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

	place, err := e.Location.GetPlace(where)
	if err != nil {
		e.Say("Ich kann " + where + " nicht finden")
		return err
	}

	e.Say(place.URL)

	e.Log.Info().
		Str("where", where).
		Float64("lat", place.Lat).
		Float64("lon", place.Lon).
		Msg("Getting Location")

	return nil
}
