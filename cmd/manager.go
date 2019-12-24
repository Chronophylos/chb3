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
	botName     string
}

func NewManager(twitch *twitch.Client, state *state.Client, version, botName string) *Manager {
	return &Manager{
		log:         log.With().Logger(),
		twitch:      twitch,
		state:       state,
		chb3Version: version,
		botName:     botName,
	}
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

	for re, action := range actions.GetAll() {
		if match := re.FindStringSubmatch(msg.Message); match != nil {
			opt := action.GetOptions()

			if opt.Name == "" {
				log.Error().Msg("malformed action name: empty string")
				return
			}

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
				BotName:     m.BotName,
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
