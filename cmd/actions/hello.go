package actions

import (
	"regexp"
)

type helloAction struct {
	options *Options
}

func newHelloAction() *helloAction {
	return &helloAction{
		options: &Options{
			Name: "hello",
			Re:   regexp.MustCompile(`(?i)(hey|hi|h[ea]llo) @?chrono(phylos(bot)?)?`),
		},
	}
}

func (a helloAction) GetOptions() *Options {
	return a.options
}

func (a helloAction) Run(e *Event) error {
	e.Log.Info().Msg("Greeting User")

	// There is an emoji but my stupid Terminal is refusing to show it peepoMad
	//                                          ↓
	e.Say("Hello " + e.Msg.User.DisplayName + " 👋")

	return nil
}
