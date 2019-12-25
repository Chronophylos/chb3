package actions

import "regexp"

type stateSleep struct {
	options *Options
}

func newStateSleep() *stateSleep {
	return &stateSleep{
		options: &Options{
			Name: "state.sleep",
			Re:   regexp.MustCompile(`(?i)^~(shut up|go sleep|sleep|sei ruhig)`),
		},
	}
}

func (a stateSleep) GetOptions() *Options {
	return a.options
}

func (a stateSleep) Run(e *Event) error {
	if !e.HasPermission(Moderator) {
		return &noPermissionError{has: e.perm, needed: Moderator}
	}

	e.Log.Info().Msg("Going to sleep")

	e.State.SetSleeping(e.Msg.Channel, true)

	return nil
}

type stateWake struct {
	options *Options
}

func newStateWake() *stateWake {
	return &stateWake{
		options: &Options{
			Name:      "state.wake",
			Re:        regexp.MustCompile(`(?i)^~(wake up|wach auf)`),
			Sleepless: true,
		},
	}
}
func (a stateWake) GetOptions() *Options {
	return a.options
}

func (a stateWake) Run(e *Event) error {
	if !e.Sleeping {
		return nil
	}

	if !e.HasPermission(Moderator) {
		return &noPermissionError{has: e.perm, needed: Moderator}
	}

	e.Log.Info().Msg("Waking up")

	e.State.SetSleeping(e.Msg.Channel, false)

	return nil
}
