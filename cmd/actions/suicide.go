package actions

import (
	"regexp"
)

type suicideAction struct {
	options *Options
}

func newSuicideAction() *suicideAction {
	return &suicideAction{
		options: &Options{
			Name: "suicide",
			Re:   regexp.MustCompile(`(?i)^~suicide`),
		},
	}
}

func (a suicideAction) GetOptions() *Options {
	return a.options
}

func (a suicideAction) Run(e *Event) error {
	e.Say("/timeout " + e.Msg.User.Name + " 1")

	return nil
}
