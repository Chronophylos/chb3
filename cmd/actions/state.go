package actions

import "regexp"

type sleepAction struct {
	options *Options
}

func newSleepAction() *sleepAction {
	return &sleepAction{
		options: &Options{
			Name: "state.sleep",
			Re:   regexp.MustCompile(`(?i)^~(shut up|go sleep|sleep|sei ruhig)`),
		},
	}
}

func (a sleepAction) GetOptions() *Options {
	return a.options
}

func (a sleepAction) Run(e *Event) error {
	if !e.HasPermission(Moderator) {
		return &noPermissionError{has: e.perm, needed: Moderator}
	}

	e.Log.Info().Msg("Going to sleep")

	e.State.SetSleeping(e.Msg.Channel, true)

	return nil
}

type wakeAction struct {
	options *Options
}

func newWakeAction() *wakeAction {
	return &wakeAction{
		options: &Options{
			Name:      "state.wake",
			Re:        regexp.MustCompile(`(?i)^~(wake up|wach auf)`),
			Sleepless: true,
		},
	}
}
func (a wakeAction) GetOptions() *Options {
	return a.options
}

func (a wakeAction) Run(e *Event) error {
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
