package actions

import (
	"fmt"
	"regexp"
)

type joinAction struct {
	options *Options
}

func newJoinAction() *joinAction {
	return &joinAction{
		options: &Options{
			Name: "admin.join",
			Re:   regexp.MustCompile(`(?i)^~join( (\w+))?`),
		},
	}
}

func (a joinAction) GetOptions() *Options {
	return a.options
}

// TODO: check if the bot  already joined the channel
func (a joinAction) Run(e *Event) error {
	if !e.IsInBotChannel() {
		return &notInBotChannelError{channel: e.Msg.Channel}
	}

	var channel string
	if e.Match[2] == "" {
		// No channel was specified; join the senders channel.
		channel = e.Msg.User.Name
	} else {
		channel = e.Match[2]
	}

	e.Log.Info().
		Str("new-channel", channel).
		Msg("Joining new channel")

	e.Twitch.Join(channel)
	e.State.JoinChannel(channel, true)

	e.Say(fmt.Sprintf(
		"I joined %s. Type `~leave %s` and I'll leave.",
		channel, channel,
	))

	return nil
}

type leaveAction struct {
	options *Options
}

func newLeaveAction() *leaveAction {
	return &leaveAction{
		options: &Options{
			Name: "admin.leave",
			Re:   regexp.MustCompile(`(?i)^~leave( (\w+))?`),
		},
	}
}

func (a leaveAction) GetOptions() *Options {
	return a.options
}

// TODO: check if the bot already left the channel
func (a leaveAction) Run(e *Event) error {
	if !e.IsInBotChannel() {
		return &notInBotChannelError{channel: e.Msg.Channel}
	}

	var channel string
	if e.Match[2] == "" {
		// No channel was specified; join the senders channel.
		channel = e.Msg.User.Name
	} else {
		channel = e.Match[2]
	}

	e.Log.Info().
		Str("old-channel", channel).
		Msg("Leaving channel")

	e.Twitch.Depart(channel)
	e.State.JoinChannel(channel, false)

	e.Say("ppPoof")

	return nil
}

type lurkAction struct {
	options *Options
}

func newLurkAction() *lurkAction {
	return &lurkAction{
		options: &Options{
			Name: "admin.lurk",
			Re:   regexp.MustCompile(`(?i)^~lurk (\w+)`),
		},
	}
}

func (a lurkAction) GetOptions() *Options {
	return a.options
}

func (a lurkAction) Run(e *Event) error {
	if !e.IsInBotChannel() {
		return &notInBotChannelError{channel: e.Msg.Channel}
	}

	channel := e.Match[2]

	e.Twitch.Join(channel)
	e.State.SetLurking(channel, true)
	e.State.JoinChannel(channel, true)

	e.Log.Info().
		Str("new-channel", channel).
		Msg("Lurking in new channel")

	e.Say(fmt.Sprintf("I'm lurking in %s now.", channel))

	return nil
}
