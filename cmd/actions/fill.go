package actions

import (
	"math/rand"
	"regexp"
	"strings"
	"time"
)

// The code here closely represents supinics code
// Retrieved on 2020-01-04 01:01
// Link: https://supinic.com/bot/command/175/code
type fillAction struct {
	options *Options
	limit   int
}

func newFillAction() *fillAction {
	rand.Seed(time.Now().Unix())

	return &fillAction{
		options: &Options{
			Name: "fill",
			Re:   regexp.MustCompile(`(?i)^~fill(o?) (.*)`),
		},
		limit: 400,
	}
}

func (a fillAction) GetOptions() *Options {
	return a.options
}

func (a fillAction) Run(e *Event) error {
	filler := strings.Split(e.Match[2], " ")

	for i, v := range filler {
		filler[i] = strings.TrimSpace(v)
	}

	var m []string
	var l int

	switch strings.ToLower(e.Match[1]) {
	case "":
		for l < a.limit {
			word := filler[rand.Intn(len(filler))]
			l += len(word) + 1
			m = append(m, word)
		}
	case "o":
		for l < a.limit {
			word := filler[l%len(filler)]
			l += len(word) + 1
			m = append(m, word)
		}
	}

	e.Log.Info().
		Strs("filler", filler).
		Msg("Filling message")
	e.Say(strings.Join(m[:len(m)-1], " "))

	return nil
}
