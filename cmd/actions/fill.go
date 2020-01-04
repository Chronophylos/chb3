package actions

import (
	"math/rand"
	"regexp"
	"strings"
	"time"
)

type fillAction struct {
	options *Options
}

func newFillAction() *fillAction {
	rand.Seed(time.Now().Unix())

	return &fillAction{
		options: &Options{
			Name: "fill",
			Re:   regexp.MustCompile(`(?i)^~fill (.*)`),
		},
	}
}

func (a fillAction) GetOptions() *Options {
	return a.options
}

func (a fillAction) Run(e *Event) error {
	filler := strings.Split(e.Match[1], " ")

	for i, v := range filler {
		filler[i] = strings.TrimSpace(v)
	}

	var m string

	for len(m) < 490 {
		m += filler[rand.Intn(len(filler))] + " "
	}

	e.Log.Info().
		Strs("filler", filler).
		Msg("Filling message")
	e.Say(m)

	return nil
}
