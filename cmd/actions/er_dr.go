package actions

import (
	"regexp"
)

type erdrAction struct {
	options *Options
}

func newErDrAction() *erdrAction {
	return &erdrAction{
		options: &Options{
			Name: "er dr",
			Re:   regexp.MustCompile(`er dr`),
		},
	}
}

func (a erdrAction) GetOptions() *Options {
	return a.options
}

func (a erdrAction) Run(e *Event) error {
	if e.Msg.User.Name == "nightbot" {
		e.Say("ückt voll oft zwei Tasten LuL")
	} else {
		e.Skip()
	}

	return nil
}
