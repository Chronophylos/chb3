package cmd

import (
	"github.com/chronophylos/chb3/cmd/actions"
	"github.com/chronophylos/chb3/state"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Manager struct {
	log         zerolog.Logger
	twitch      *twitch.Client
	state       *state.Client
	chb3Version string
}

func NewManager(twitch *twitch.Client, state *state.Client, version string) *Manager {
	return &Manager{
		log:         log.With().Logger(),
		twitch:      twitch,
		state:       state,
		chb3Version: version,
	}
}

func (m *Manager) RunActions(msg *twitch.PrivateMessage, user *state.User) {
	for re, action := range actions.GetAll() {
		if match := re.FindStringSubmatch(msg.Message); match != nil {
			e := &actions.Event{
				Log: m.log.With().
					Str("action", action.GetName()).
					Str("channel", msg.Channel).
					Str("invoker", msg.User.Name).
					Logger(),
				Twitch:      m.twitch,
				State:       m.state,
				CHB3Version: m.chb3Version,
				Match:       match,
				Action:      action,
				Msg:         msg,
			}
			e.Init()
			action.Run(e)
			if !e.Skipped {
				return
			}
		}
	}
}
