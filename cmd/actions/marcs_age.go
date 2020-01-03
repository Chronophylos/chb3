package actions

import (
	"regexp"
)

type marcsAgeAction struct {
	options *Options
}

func newMarcsAgeAction() *marcsAgeAction {
	return &marcsAgeAction{
		options: &Options{
			Name: "marcs age",
			Re:   regexp.MustCompile(`(?i)(\bmarc alter\b)|(\balter marc\b)`),
		},
	}
}

func (a marcsAgeAction) GetOptions() *Options {
	return a.options
}

func (a marcsAgeAction) Run(e *Event) error {
	e.Log.Info().Msg("Gratulating marc for his birthday")
	e.Say("Marc is heute 16 geworden FeelsBirthdayMan Clap")

	return nil
}
