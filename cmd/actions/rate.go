package actions

import (
	"crypto/md5"
	"fmt"
	"math/big"
	"regexp"
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
	// get hash
	hash := md5.Sum([]byte(s))

	// convert to big int
	p := new(big.Int).SetBytes(hash[:])

	// modulus 101
	_, m := p.DivMod(p, big.NewInt(101), new(big.Int))

	// int to float
	q := new(big.Float).SetInt(m)

	// divide by 10
	r, _ := q.Quo(q, big.NewFloat(10)).Float32()

	return r
}
