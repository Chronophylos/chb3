package cmd

import (
	"github.com/gempir/go-twitch-irc/v2"

	"github.com/chronophylos/chb3/state"
	"github.com/rs/zerolog"
)

type Permission int

const (
	Everyone Permission = iota
	Subscriber
	Regular
	Moderator
	Broadcaster
	Owner
)

type Event struct {
	Log    zerolog.Logger
	twitch *twitch.Client
	State  *state.Client

	perm    Permission
	Channel string
	Match   []string
	Skipped bool
}

func (e *Event) Say(message string) {
	e.twitch.Say(e.Channel, message)
}

func (e *Event) HasPermission(perm Permission) bool {
	return e.perm < perm
}

func (e *Event) IsOwner() bool {
	return e.HasPermission(Owner)
}

// TODO: implement
func (e *Event) IsCoolingDown() bool {
	return false
}

func (e *Event) Skip() {
	e.Skipped = true
}
