package actions

import (
	"fmt"
)

type version struct{}

func (v version) Run(e *Event) {
	e.Say(fmt.Sprintf("I'm a bot written by Chronophylos in Golang. Current version is %s.", e.CHB3Version))
}

func (v version) GetName() string { return "version" }
