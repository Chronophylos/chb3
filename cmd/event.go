package cmd

import (
	"github.com/chronophylos/chb3/state"
	"github.com/chronophylos/chb3/twotsch"
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
	twitch *twotsch.Client
	State  *state.Client

	perm  Permission
	Match []string
}

func NewEvent() *Event {
	return &Event{}
}

func (e *Event) Say(message string) {

}

func (e *Event) HasPermission(perm Permission) bool {
	return e.perm < perm
}

func (e *Event) IsOwner() bool {
	return e.HasPermission(Owner)
}

func (e *Event) IsCoolingDown() bool {

}
