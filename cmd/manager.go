package cmd

import (
	"github.com/gempir/go-twitch-irc/v2"
)

type infoContainer struct {
	twitch *twitch.Client
	state  *state.Client
	chb3   *struct {
		version string
	}
}

type Manager struct {
	infoContainer
}

func (m *Manager) RunActions(msg *twitch.PrivateMessage) {
	for re, action := range actionMap {
		if match := re.FindStringSubmatch(msg.Message); match != nil {
			e := &Event{
				log: m.log.With().
					Str("action", action.GetName()).
					Str("channel", msg.Channel).
					Str("invoker", msg.User.Name).
					Logger(),
				twitch: m.twitch,
				state:  m.state,
				chb3:   m.chb3,
				match:  match,
				action: action,
			}
			e.Init()
			action.Run(e)
			if !e.Skipped {
				return
			}
		}
	}
}
