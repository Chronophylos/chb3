package actions

import (
	"fmt"
)

type version struct{}

func (a version) GetOptions() *Options {
	return &Options{
		Name: "version",
	}
}

func (a version) Run(e *Event) error {
	e.Say(fmt.Sprintf(
		"I'm a bot written by Chronophylos in Golang. Current version is %s.",
		e.CHB3Version,
	))

	return nil
}
