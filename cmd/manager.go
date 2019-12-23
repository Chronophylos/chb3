package cmd

import (
	"github.com/gempir/go-twitch-irc/v2"
)

type Manager struct {
	twitch *twitch.Client
}

func (m *Manager) RunActions(msg *twitch.PrivateMessage) {
	e := &Event{
		twitch: m.twitch,
	}

	for re, action := range actionMap {
		if match := re.FindStringSubmatch(msg.Message); match != nil {
			e.Match = match
			action.Run(e)
			if !e.Skipped() {
				return
			}
		}
	}
}
