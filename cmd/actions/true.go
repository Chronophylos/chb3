package actions

import (
	"regexp"
)

type trueAction struct {
	options *Options
}

func newTrueAction() *trueAction {
	return &trueAction{
		options: &Options{
			Name: "true",
			Re:   regexp.MustCompile(`(?i)^~true`),
		},
	}
}

func (a trueAction) GetOptions() *Options {
	return a.options
}

func (a trueAction) Run(e *Event) error {
	e.Say("True aaaaaaaaaaaand… Yeah, that's pretty true. That's true and- yeah that's true. That's true. That's true- That's pretty true. That's pretty true, I mean- *inhales* … That's true. Yeah. That's true. Uhm- That's true. That's fuckin' true. Uhm… That's how it is dude.")

	return nil
}
