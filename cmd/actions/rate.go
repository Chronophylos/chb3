package actions

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strconv"
)

type rateAction struct {
	options *Options
}

func newRateAction() *rateAction {
	return &rateAction{
		options: &Options{
			Name: "rate",
			Re:   regexp.MustCompile(`(?i)^~rate (.*)`),
		},
	}
}

func (a rateAction) GetOptions() *Options { return a.options }

func (a rateAction) Run(e *Event) error {
	what := e.Match[1]
	rating := rate(what)

	e.Log.Info().
		Str("what", what).
		Float32("rating", rating).
		Msg("rating")

	e.Say(fmt.Sprintf("I rate %s %.1f/10", what, rating))

	return nil
}

func rate(s string) float32 {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(s)))
	p, _ := strconv.ParseInt(hash, 16, 64)
	q := float32(p%101) / 10
	return q
}
