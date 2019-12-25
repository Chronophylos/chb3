package cmd

import (
	"fmt"

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
	botName     string
	actions     actions.Actions
}

func NewManager(twitch *twitch.Client, state *state.Client, version, botName string) (*Manager, error) {
	// check actions for errors
	for _, action := range actions.GetAll() {
		if err := actions.Check(action); err != nil {
			return &Manager{}, fmt.Errorf("malformed action %T: %v", action, err)
		}
	}

	return &Manager{
		log:         log.With().Logger(),
		twitch:      twitch,
		state:       state,
		chb3Version: version,
		botName:     botName,
		actions:     actions.GetAll(),
	}, nil
}

func (m *Manager) RunActions(msg *twitch.PrivateMessage, user *state.User) {
	log := m.log.With().
		Str("channel", msg.Channel).
		Logger()

	sleeping, err := m.state.IsSleeping(msg.Channel)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Checking if channel is sleeping")
		return
	}

	for _, action := range m.actions {
		opt := action.GetOptions()

		if match := opt.Re.FindStringSubmatch(msg.Message); match != nil {

			log := log.With().
				Str("action", opt.Name).
				Str("invoker", msg.User.Name).
				Logger()

			// sleeping: nothing to do
			if sleeping && !opt.Sleepless {
				return
			}

			log.Debug().
				Strs("match", match).
				Str("message", msg.Message).
				Msg("Found matching action")

			e := &actions.Event{
				Log:         log,
				Twitch:      m.twitch,
				State:       m.state,
				CHB3Version: m.chb3Version,
				Match:       match,
				Action:      action,
				Msg:         msg,
				Sleeping:    sleeping,
				BotName:     m.botName,
			}
			e.Init()

			if err := action.Run(e); err != nil {
				log.Error().Err(err).Msg("action failed")
				return
			}

			if !e.Skipped {
				return
			}
		}
	}
}
