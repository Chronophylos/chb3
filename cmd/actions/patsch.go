package actions

import (
	"fmt"
	"regexp"

	"github.com/chronophylos/chb3/state"
)

type patscheckAction struct {
	options *Options
}

func newPatscheckAction() *patscheckAction {
	return &patscheckAction{
		options: &Options{
			Name: "patsch.check",
			Re:   regexp.MustCompile(`(?i)^~hihsg\??`),
		},
	}
}

func (a patscheckAction) GetOptions() *Options {
	return a.options
}

func (a patscheckAction) Run(e *Event) error {
	user, err := e.State.GetUserByID(e.Msg.User.ID)
	if err != nil {
		return fmt.Errorf("getting user by id: %v", err)
	}

	if user.PatschCount == 0 {
		e.Say("You have never patted the fish before. You should do that now!")
		return nil
	}

	e.Log.Info().
		Int("streak", user.PatschStreak).
		Int("total", user.PatschCount).
		Msg("Checking Patscher")

	var message string

	message = "You "
	if user.HasPatschedToday(e.Msg.Time) {
		message += "already"
	} else {
		message += "have not yet"
	}
	message += " patted today. "

	if user.PatschStreak == 0 {
		message += "You don't have a streak ongoing"
	} else {
		message += fmt.Sprintf("Your current streak is %d", user.PatschStreak)
	}

	message += " and in total you have patted "
	if user.PatschCount == 1 {
		message += "once."
	} else {
		message += fmt.Sprintf("%d times.", user.PatschCount)
	}

	e.Say(message)

	return nil
}

type patschAction struct {
	options *Options
}

func newPatschAction() *patschAction {
	return &patschAction{
		options: &Options{
			Name: "patsch.patsch",
			Re:   regexp.MustCompile(`fischPatsch|fishPat`),
		},
	}
}

func (a patschAction) GetOptions() *Options {
	return a.options
}

func (a patschAction) Run(e *Event) error {
	if e.Msg.Channel != "furzbart" && !(e.Debug && e.IsInBotChannel()) {
		e.Skip()
	}

	e.Log.Info().Msg("Patsch!")

	if len(e.Match) > 1 {
		e.Say("/timeout " + e.Msg.User.Name + " 1 Wenn du so viel patschst wird das ne Flunder")
		return nil
	}

	if err := e.State.Patsch(e.Msg.User.ID, e.Msg.Time); err != nil {
		if err == state.ErrAlreadyPatsched {
			e.Say("Du hast heute schon gepatscht")
			return nil
		} else if err == state.ErrForgotToPatsch {
			return nil
		}
		return err
	}

	return nil
}
