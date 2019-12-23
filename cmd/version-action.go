package cmd

import (
	"fmt"
)

type Version struct{}

func (v Version) Run(e *Event) {
	e.Say(fmt.Sprintf("I'm a bot written by Chronophylos in Golang. Current version is %s.", e.CHB3.Version))
}
