package actions

import "fmt"

type joinChannel struct{}

func (a joinChannel) GetOptions() *Options {
	return &Options{
		Name: "admin.join",
	}
}

// TODO: check if the bot  already joined the channel
func (a joinChannel) Run(e *Event) error {
	if !e.IsInBotChannel() {
		return &notInBotChannel{channel: e.Msg.Channel}
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

type leaveChannel struct{}

func (a leaveChannel) GetOptions() *Options {
	return &Options{
		Name: "admin.leave",
	}
}

// TODO: check if the bot already left the channel
func (a leaveChannel) Run(e *Event) error {
	if !e.IsInBotChannel() {
		return &notInBotChannel{channel: e.Msg.Channel}
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
