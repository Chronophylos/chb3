package actions

type stateSleep struct{}

func (a stateSleep) GetOptions() *Options {
	return &Options{
		Name: "state.sleep",
	}
}

func (a stateSleep) Run(e *Event) error {
	if !e.HasPermission(Moderator) {
		return &noPermissionError{has: e.perm, needed: Moderator}
	}

	e.Log.Info().Msg("Going to sleep")

	e.State.SetSleeping(e.Msg.Channel, true)

	return nil
}

type stateWake struct{}

func (a stateWake) GetOptions() *Options {
	return &Options{
		Name:      "state.wake",
		Sleepless: true,
	}
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
