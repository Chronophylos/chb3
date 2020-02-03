package actions

import (
	"fmt"
	"regexp"

	"github.com/chronophylos/chb3/buildinfo"
)

type versionAction struct {
	options *Options
}

func newVersionAction() *versionAction {
	return &versionAction{
		options: &Options{
			Name: "version",
			Re:   regexp.MustCompile(`(?i)^~version`),
		},
	}
}

func (a versionAction) GetOptions() *Options {
	return a.options
}

func (a versionAction) Run(e *Event) error {
	e.Say(fmt.Sprintf(
		"I'm a bot written by Chronophylos in Golang. Current version is %s (%s).",
		buildinfo.Version(), buildinfo.Commit(),
	))

	return nil
}
