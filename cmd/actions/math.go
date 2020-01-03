package actions

import (
	"fmt"
	"regexp"

	"github.com/Knetic/govaluate"
)

type mathAction struct {
	options *Options
}

func newMathAction() *mathAction {
	return &mathAction{
		options: &Options{
			Name: "math",
			Re:   regexp.MustCompile(`(?i)^~(math|quickmafs) (.*)`),
		},
	}
}

func (a mathAction) GetOptions() *Options {
	return a.options
}

func (a mathAction) Run(e *Event) error {
	exprString := e.Match[2]

	defer func() {
		if r := recover(); r != nil {
			e.Log.Info().
				Str("expression", exprString).
				Msg("failed to do math")

			e.Say("I can't calculate that :(")
		}
	}()

	expr, err := govaluate.NewEvaluableExpression(exprString)
	if err != nil {
		e.Say(fmt.Sprintf("Error: %v", err))
		return err
	}

	result, err := expr.Evaluate(nil)
	if err != nil {
		return err
	}

	e.Log.Info().
		Str("expression", exprString).
		Interface("result", result).
		Msg("doing math")

	e.Say(fmt.Sprintf("%v", result))

	return nil
}
