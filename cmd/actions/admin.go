package actions

import (
	"errors"
	"fmt"
	"regexp"
)

type joinChannel struct {
	options *Options
}

func newJoinChannel() *joinChannel {
	return &joinChannel{
		options: &Options{
			Name: "admin.join",
			Re:   regexp.MustCompile(`(?i)^~join( (\w+))?`),
		},
	}
}

func (a joinChannel) GetOptions() *Options {
	return a.options
}

// TODO: check if the bot  already joined the channel
func (a joinChannel) Run(e *Event) error {
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

type leaveChannel struct {
	options *Options
}

func newLeaveChannel() *leaveChannel {
	return &leaveChannel{
		options: &Options{
			Name: "admin.leave",
			Re:   regexp.MustCompile(`(?i)^~leave( (\w+))?`),
		},
	}
}

func (a leaveChannel) GetOptions() *Options {
	return a.options
}

// TODO: check if the bot already left the channel
func (a leaveChannel) Run(e *Event) error {
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

type lurkChannel struct {
	options *Options
}

func newLurkChannel() *lurkChannel {
	return &lurkChannel{
		options: &Options{
			Name: "admin.lurk",
			Re:   regexp.MustCompile(`(?i)^~lurk (\w+)`),
		},
	}
}

func (a lurkChannel) GetOptions() *Options {
	return a.options
}

func (a lurkChannel) Run(e *Event) error {
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

type debug struct {
	options *Options
}

func newDebug() *debug {
	return &debug{
		options: &Options{
			Name: "admin.debug",
			Re:   regexp.MustCompile(`(?i)^~debug (\w+)`),
		},
	}
}

func (a debug) GetOptions() *Options {
	return a.options
}

func (a debug) Run(e *Event) error {
	return errors.New("Not yet implemented")
}
