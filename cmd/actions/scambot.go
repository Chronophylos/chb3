package actions

import (
	"regexp"
)

type scambotAction struct {
	options *Options
}

func newScambotAction() *scambotAction {
	return &scambotAction{
		options: &Options{
			Name: "scambot",
			Re:   regexp.MustCompile(`(?i)\bscambot\b`),
		},
	}
}

func (a scambotAction) GetOptions() *Options {
	return a.options
}

func (a scambotAction) Run(e *Event) error {
	e.Log.Info().Msg("I'm not a scambot")
	e.Say("FeelsNotsureMan")

	return nil
}
