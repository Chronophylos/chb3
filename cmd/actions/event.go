package actions

import (
	"github.com/chronophylos/chb3/state"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/rs/zerolog"
)

var botNames = []string{"nightbot", "fossabot", "streamelements"}

// Permission is a int
type Permission int

// Possible values for Permission.
const (
	Everyone Permission = iota
	Subscriber
	Regular
	Moderator
	Broadcaster
	Owner
)

type Event struct {
	Log         zerolog.Logger
	Twitch      *twitch.Client
	State       *state.Client
	CHB3Version string

	Action Action
	Msg    *twitch.PrivateMessage
	Match  []string

	Sleeping bool

	perm Permission

	Skipped bool
}

// Init sets some internal values like the permission of the sender.
func (e *Event) Init() {
	if e.IsOwner() {
		e.perm = Owner
	} else if e.IsBroadcaster() {
		e.perm = Broadcaster
	} else if e.IsModerator() {
		e.perm = Moderator
	} else if e.IsRegular() {
		e.perm = Regular
	} else if e.IsSubscriber() {
		e.perm = Subscriber
	}
	// no need to set e.perm to Everyone since it is the default.
}

// Say sends message to the current channel.
func (e *Event) Say(message string) {
	e.Twitch.Say(e.Msg.Channel, message)
}

// HasPermission compares perm with the permission level of the sender and
// reports wheather the sender has a permission of at least perm.
func (e *Event) HasPermission(perm Permission) bool {
	return e.perm > perm
}

// IsCoolingDown reports wheather the command is available or if it is cooling
// down.
// This could be because of a user, channel or command cooldown.
// TODO: implement
func (e *Event) IsCoolingDown() bool {
	return false
}

// IsRegular reports wheather the sender is a regular.
// TODO: implement
func (e *Event) IsRegular() bool { return false }

// IsSubscriber reports wheather the sender is a subscriber in the current
// channel.
func (e *Event) IsSubscriber() bool {
	return e.Msg.Tags["subscriber"] == "1"
}

// IsModerator reports wheather the sender is a moderator in the current
// channel.
func (e *Event) IsModerator() bool {
	return e.Msg.Tags["mod"] == "1"
}

// IsBroadcaster reports wheather the sender is the owner of the current
// channel.
func (e *Event) IsBroadcaster() bool {
	return e.Msg.User.Name == e.Msg.Channel
}

// IsOwner reports wheather the sender is the bots owner.
func (e *Event) IsOwner() bool {
	return e.Msg.User.ID == "54946241"
}

// IsBot reports wheather the message was sent by a bot.
// Currently it compares the name of the sender with a list of know bots.
// TODO: check in other places eg. badges
func (e *Event) IsBot() bool {
	for _, name := range botNames {
		if e.Msg.User.Name == name {
			return true
		}
	}
	return false
}

// Skip skips this command and allows other commands to run.
func (e *Event) Skip() {
	e.Skipped = true
}
