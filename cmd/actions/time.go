package actions

import (
	"fmt"
	"regexp"
	"time"
)

type timeAction struct {
	options *Options
}

func newTimeAction() *timeAction {
	return &timeAction{
		options: &Options{
			Name: "time",
			Re:   regexp.MustCompile(`(?i)^~time`),
		},
	}
}

func (a timeAction) GetOptions() *Options {
	return a.options
}

func (a timeAction) Run(e *Event) error {
	e.Say(fmt.Sprintf(
		"Twitch Time: %s Server Time: %s",
		e.Msg.Time.Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
	))

	return nil
}
