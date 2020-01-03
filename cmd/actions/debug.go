package actions

import (
	"errors"
	"os"
	"regexp"
)

type debugAction struct {
	options *Options
}

func newDebugAction() *debugAction {
	return &debugAction{
		options: &Options{
			Name: "debug",
			Re:   regexp.MustCompile(`(?i)^~debug (\w+)`),
			Perm: Owner,
		},
	}
}

func (a debugAction) GetOptions() *Options {
	return a.options
}

func (a debugAction) Run(e *Event) error {
	action := e.Match[1]

	switch action {
	case "enable", "disable", "reconnect":
		return errors.New("Not yet implemented")
	case "exit":
		e.Log.Info().Msg("Exiting")
		os.Exit(0)
	}

	return nil
}
