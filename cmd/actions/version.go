package actions

import (
	"fmt"
	"regexp"
)

type version struct {
	options *Options
}

func newVersion() *version {
	return &version{
		options: &Options{
			Name: "version",
			Re:   regexp.MustCompile(`(?i)^~version`),
		},
	}
}

func (a version) GetOptions() *Options {
	return a.options
}

func (a version) Run(e *Event) error {
	e.Say(fmt.Sprintf(
		"I'm a bot written by Chronophylos in Golang. Current version is %s.",
		e.CHB3Version,
	))

	return nil
}
