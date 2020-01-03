package actions

import (
	"regexp"
)

type helloStirnbotAction struct {
	options *Options
}

func newHelloStirnbotAction() *helloStirnbotAction {
	return &helloStirnbotAction{
		options: &Options{
			Name: "hello stirnbot",
			Re:   regexp.MustCompile(`^I'm here FeelsGoodMan$`),
		},
	}
}

func (a helloStirnbotAction) GetOptions() *Options {
	return a.options
}

func (a helloStirnbotAction) Run(e *Event) error {
	if e.Msg.User.Name == "stirnbot" {
		e.Log.Info().Msg("Greeting StirnBot")
		e.Say("StirnBot MrDestructoid /")
	} else {
		e.Skip()
	}

	return nil
}
