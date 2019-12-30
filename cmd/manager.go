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
	Log         zerolog.Logger
	Twitch      *twitch.Client
	State       *state.Client
	CHB3Version string
	BotName     string

	actions actions.Actions

	Config struct {
		Debug *bool
	}
}

func NewManager(twitch *twitch.Client, state *state.Client, version, botName string, debug *bool) (*Manager, error) {
	// check actions for errors
	for _, action := range actions.GetAll() {
		if err := actions.Check(action); err != nil {
			return &Manager{}, fmt.Errorf("malformed action %T: %v", action, err)
		}
	}

	m := &Manager{
		Log:         log.With().Logger(),
		Twitch:      twitch,
		State:       state,
		CHB3Version: version,
		BotName:     botName,
		actions:     actions.GetAll(),
	}
	m.Config.Debug = debug
	return m, nil
}

func (m *Manager) RunActions(msg *twitch.PrivateMessage, user *state.User) {
	log := m.Log.With().
		Str("channel", msg.Channel).
		Logger()

	sleeping, err := m.State.IsSleeping(msg.Channel)
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
				Twitch:      m.Twitch,
				State:       m.State,
				CHB3Version: m.CHB3Version,
				Match:       match,
				Msg:         msg,
				Sleeping:    sleeping,
				BotName:     m.BotName,
			}
			e.Init()

			if !e.HasPermission(opt.Perm) {
				return // Skip
			}

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
