package cmd

import (
	"fmt"

	"github.com/chronophylos/chb3/cmd/actions"
	"github.com/chronophylos/chb3/database"
	"github.com/chronophylos/chb3/nominatim"
	"github.com/chronophylos/chb3/openweather"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Manager struct {
	Log           zerolog.Logger
	Twitch        *twitch.Client
	DB            *database.Client
	Location      *nominatim.Client
	Weather       *openweather.Client
	CHB3Version   string
	ImgurClientID string
	BotName       string

	actions actions.Actions

	Config struct {
		Debug *bool
	}
}

func NewManager(twitch *twitch.Client, db *database.Client, weather *openweather.Client, location *nominatim.Client, imgurClientID, version, botName string, debug *bool) (*Manager, error) {
	// check actions for errors
	for _, action := range actions.GetAll() {
		if err := actions.Check(action); err != nil {
			return &Manager{}, fmt.Errorf("malformed action %T: %v", action, err)
		}
	}

	m := &Manager{
		Log:           log.With().Logger(),
		Twitch:        twitch,
		DB:            db,
		Weather:       weather,
		Location:      location,
		CHB3Version:   version,
		ImgurClientID: imgurClientID,
		BotName:       botName,
		actions:       actions.GetAll(),
	}
	m.Config.Debug = debug
	return m, nil
}

func (m *Manager) RunActions(msg *twitch.PrivateMessage) {
	log := m.Log.With().
		Str("channel", msg.Channel).
		Logger()

	channel, err := m.DB.GetChannel(msg.Channel)
	if err != nil {
		log.Error().
			Str("channel", msg.Channel).
			Err(err).
			Msg("Could not get channel from database")
		return
	}

	if !channel.Enabled || channel.ReadOnly {
		return
	}

	for _, action := range m.actions {
		opt := action.GetOptions()

		// if sleeping and command is not ignoring sleep
		if channel.Paused && !opt.Sleepless {
			continue
		}

		var channelDisabled bool
		if opt.DisabledChannels != nil {
			_, channelDisabled = opt.DisabledChannels[msg.Channel]
		}

		if opt.Disabled || channelDisabled {
			continue
		}

		if match := opt.Re.FindStringSubmatch(msg.Message); match != nil {
			log := log.With().
				Str("action", opt.Name).
				Str("invoker", msg.User.Name).
				Logger()

			log.Debug().
				Strs("match", match).
				Str("message", msg.Message).
				Msg("Found matching action")

			e := &actions.Event{
				Log:           log,
				Twitch:        m.Twitch,
				DB:            m.DB,
				Weather:       m.Weather,
				Location:      m.Location,
				CHB3Version:   m.CHB3Version,
				ImgurClientID: m.ImgurClientID,
				Match:         match,
				Msg:           msg,
				Channel:       channel,
				BotName:       m.BotName,
			}
			e.Init()

			if !e.HasPermission(opt.Perm) {
				log.Warn().
					Str("has", e.Perm.String()).
					Str("needs", opt.Perm.String()).
					Msg("permission not high enough")
				continue // Skip
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
