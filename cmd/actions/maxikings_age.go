package actions

import (
	"regexp"
)

type maxikingsAgeAction struct {
	options *Options
}

func newMaxikingsAgeAction() *maxikingsAgeAction {
	return &maxikingsAgeAction{
		options: &Options{
			Name: "maxikings age",
			Re:   regexp.MustCompile(`(?i)\balter maxiking\b`),
		},
	}
}

func (a maxikingsAgeAction) GetOptions() *Options {
	return a.options
}

func (a maxikingsAgeAction) Run(e *Event) error {
	e.Log.Info().Msg("Checking Maxikings age")
	e.Say("Maxiking is still underage PepeLaugh")

	return nil
}
