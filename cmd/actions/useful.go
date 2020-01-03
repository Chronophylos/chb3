package actions

import "regexp"

type vanishReplyAction struct {
	options *Options
}

func newVanishReplyAction() *vanishReplyAction {
	return &vanishReplyAction{
		options: &Options{
			Name: "vanish-reply",
			Re:   regexp.MustCompile(`^!vanish`),
			Perm: Moderator,
		},
	}
}

func (a vanishReplyAction) GetOptions() *Options {
	return a.options
}

func (a vanishReplyAction) Run(e *Event) error {
	if e.Msg.Channel != "moondye7" {
		e.Skip()
		return nil
	}
	if e.IsBot() {
		e.Skip()
		return nil
	}

	e.Log.Info().Msg("unmod the mods")

	e.Say("Try /unmod " + e.Msg.User.Name + " first weSmart")

	return nil
}

type circumflexAction struct {
	options *Options
}

func newCircumflexAction() *circumflexAction {
	return &circumflexAction{
		options: &Options{
			Name: "circumflex",
			Re:   regexp.MustCompile(`^\^`),
		},
	}
}

func (a circumflexAction) GetOptions() *Options {
	return a.options
}

func (a circumflexAction) Run(e *Event) error {
	if e.IsBot() {
		e.Skip()
		return nil
	}
	if e.Msg.Channel == "moondye7" {
		e.Skip()
		return nil
	}

	e.Say("^")

	return nil
}
