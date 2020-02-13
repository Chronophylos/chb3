package actions

import (
	"regexp"
)

type heartAction struct {
	options *Options
}

func newHeartAction() *heartAction {
	return &heartAction{
		options: &Options{
			Name: "heart",
			Re:   regexp.MustCompile(`^~\<3`),
		},
	}
}

func (a heartAction) GetOptions() *Options {
	return a.options
}

func (a heartAction) Run(e *Event) error {
	e.Say("https://www.wolframalpha.com/input/?i=%7By+%3D+Re%28sqrt%28abs%28x%29%281-abs%28x%29%29%29%29%2C+y+%3D+Re%28-sqrt%281-sqrt%28abs%28x%29%29%29%29%7D")

	return nil
}
