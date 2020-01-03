package actions

import "regexp"

type pingAction struct {
	options *Options
}

func newPingAction() *pingAction {
	return &pingAction{
		options: &Options{
			Name: "ping",
			Re:   regexp.MustCompile(`(?i)^~ping`),
		},
	}
}

func (a pingAction) GetOptions() *Options {
	return a.options
}

func (a pingAction) Run(e *Event) error {
	e.Say("pong")

	return nil
}
