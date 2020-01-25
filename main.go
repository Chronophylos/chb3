package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	sw "github.com/JoshuaDoes/gofuckyourself"
	"github.com/akamensky/argparse"
	"github.com/chronophylos/chb3/cmd"
	"github.com/chronophylos/chb3/nominatim"
	"github.com/chronophylos/chb3/openweather"
	"github.com/chronophylos/chb3/state"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/nicklaw5/helix"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Build Infos
var (
	Version = "3.6.1"
)

// Flags
var (
	showSecrets *bool
	debug       *bool
)

// Config
var (
	twitchUsername string
	twitchToken    string

	imgurClientID string

	openweatherAppID string

	swears []string
)

var userBlacklist = []string{
	"86621952", // ritzenspalt
	"38286541", // klotz795
}

// Globals
var (
	owClient     *openweather.Client
	stateClient  *state.Client
	twitchClient *twitch.Client
	swearfilter  *sw.SwearFilter
	osmClient    *nominatim.Client
	helixClient  *helix.Client
)

func main() {
	// Commandline Flags {{{
	// Create new parser
	parser := argparse.NewParser("chb3", "ChronophylosBot but version 3")

	debug = parser.Flag("", "debug",
		&argparse.Options{Help: "Enable debugging. Sets --level=DEBUG."})

	showSecrets = parser.Flag("", "show-secrets",
		&argparse.Options{Help: "Show secrets in log (eg. your twitch token)."})

	// Parse Flags
	err := parser.Parse(os.Args)
	if err != nil {
		// Print usage for err
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}
	// }}}

	// Setup logger
	setGlobalLogger()

	// Viper {{{
	viper.SetConfigType("toml") // toml is nice
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/chb3") // config location
	viper.AddConfigPath(".")         // also look in the working directory

	// Not sure what to use this for yet.
	viper.SetEnvPrefix("CHB3")

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			log.Fatal().
				Err(err).
				Msg("Error config not found.")
		}
		log.Fatal().
			Err(err).
			Msg("Error reading config.")
	}
	// }}}

	// Required Settings {{{
	if !viper.IsSet("twitch.username") {
		log.Fatal().Msg("Twitch Username is not set.")
	}
	twitchUsername = viper.GetString("twitch.username")

	if !viper.IsSet("twitch.token") {
		log.Fatal().Msg("Twitch Token is not set.")
	}
	twitchToken = viper.GetString("twitch.token")

	if !viper.IsSet("twitch.clientid") {
		log.Fatal().Msg("Twitch ClientID is not set.")
	}

	if !viper.IsSet("imgur.clientid") {
		log.Fatal().Msg("Imgur ClientID is not set.")
	}
	imgurClientID = viper.GetString("imgur.clientid")

	if !viper.IsSet("openweather.appid") {
		log.Fatal().Msg("OpenWeather AppID is not set.")
	}
	openweatherAppID = viper.GetString("openweather.appid")

	swears = viper.GetStringSlice("chb3.swears")
	// }}}

	log.Info().Msgf("Starting CHB3 %s", Version)

	// Signals {{{
	// setup signal catching
	sigs := make(chan os.Signal, 1)

	// catch all term signals
	signal.Notify(sigs, os.Interrupt)

	// method invoked upon seeing signal
	go func() {
		s := <-sigs
		log.Info().Msgf("Received %s. Quitting.", s)
		twitchClient.Disconnect()
		os.Exit(1)
	}()
	// }}}

	wg := sync.WaitGroup{}

	wg.Add(5)

	go func() {
		stateClient, err = state.NewClient("mongodb://localhost:27017")
		if err != nil {
			log.Fatal().
				Err(err).
				Msg("Could not create State Client")
		}
		wg.Done()
		log.Info().Msg("Created State Client")
	}()

	go func() {
		owClient = openweather.NewClient(openweatherAppID, Version)
		wg.Done()
		log.Info().
			Str("appid", censor(openweatherAppID)).
			Msg("Created new OpenWeather Client")
	}()

	go func() {
		twitchClient = twitch.NewClient(twitchUsername, twitchToken)
		wg.Done()
		log.Info().
			Str("username", twitchUsername).
			Str("token", censor(twitchToken)).
			Msg("Created new Twitch Client")
	}()

	// why did i make this concurrent?
	go func() {
		swearfilter = &sw.SwearFilter{BlacklistedWords: swears}
		wg.Done()
		log.Info().Strs("swears", swears).Msg("Loaded Swearfilter")
	}()

	go func() {
		osmClient = &nominatim.Client{UserAgent: "ChronophylosBot/" + Version}
		wg.Done()
		log.Info().Msg("Created OpenStreetMaps Client")
	}()

	wg.Wait()

	helixClient, err := helix.NewClient(&helix.Options{
		ClientID:  viper.GetString("twitch.clientid"),
		UserAgent: "ChronophylosBot/" + Version,
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Could not create helix client")
	}

	manager, err := cmd.NewManager(twitchClient, stateClient, owClient, osmClient, imgurClientID, Version, twitchUsername, debug)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("could not create command manager")
	}

	// Twitch Client Event Handling {{{
	twitchClient.OnReconnectMessage(func(message twitch.ReconnectMessage) {
		log.Info().Msg("Reconnected to chat")
	})

	twitchClient.OnPrivateMessage(func(message twitch.PrivateMessage) {
		// Don't listen to messages sent by the bot
		if message.User.Name == twitchUsername {
			return
		}

		for _, userID := range userBlacklist {
			if message.User.ID == userID {
				return
			}
		}

		message.Message = strings.ReplaceAll(message.Message, "\U000e0000", "")
		message.Message = strings.TrimSpace(message.Message)

		user, err := stateClient.BumpUser(message.User, message.Time)
		if err != nil {
			log.Error().
				Err(err).
				Str("username", message.User.Name).
				Msg("Bumping user")
			return
		}

		isLurking, err := stateClient.IsLurking(message.Channel)
		if err != nil {
			log.Error().
				Err(err).
				Str("channel", message.Channel).
				Msg("Checking if bot is lurking")
			return
		}

		if isLurking {
			return
		}

		foundASwear, swearsFound, err := swearfilter.Check(message.Message)
		if err != nil {
			log.Error().
				Str("message", message.Message).
				Msg("Checking message for swears")
			return
		}
		if foundASwear {
			log.Info().
				Strs("swears", swearsFound).
				Msg("Found forbidden words")
			return
		}

		manager.RunActions(&message, user)

		checkForVoicemails(message.User.Name, message.Channel)
	})

	twitchClient.OnConnect(func() {
		log.Info().Msg("Connected to chat")
		twitchClient.Say(twitchUsername, "Connected FeelsGoodMan")
	})
	// }}}

	joinedChannels, err := stateClient.GetJoinedChannels()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Getting currently joined channels")
	}
	log.Info().
		Str("own-channel", twitchUsername).
		Interface("channels", joinedChannels).
		Msg("Joining Channels")
	// Make sure the bot is always in it's own channel
	twitchClient.Join(twitchUsername)
	stateClient.JoinChannel(twitchUsername, true)
	twitchClient.Join(joinedChannels...)

	for {
		log.Info().Msg("Connecting to chat")

		if twitchClient.Connect(); err != nil {
			log.Fatal().
				Err(err).
				Msg("Failed to connect to chat")
		}

		log.Info().Msg("Disconnected from chat")

		time.Sleep(10 * time.Second)
	}
}

// check for voicemails {{{
func checkForVoicemails(username, channel string) {

	voicemails, err := stateClient.CheckForVoicemails(username)
	if err != nil {
		log.Error().
			Err(err).
			Str("username", username).
			Msg("Failed to get Voicemails")
		return
	}

	if len(voicemails) > 0 {
		log.Info().
			Int("count", len(voicemails)).
			Str("username", username).
			Msg("Replaying Voicemails")

		messages := []string{"@" + username + ", " + pluralize("message", int64(len(voicemails))) + " for you: "}
		i := 0
		noDelimiter := true
		var delimiter string

		for _, voicemail := range voicemails {
			message := voicemail.String()
			if len(messages[i])+len(message) > 400 {
				i++
				messages = append(messages, message)
			} else {
				delimiter = " — "
				if noDelimiter {
					noDelimiter = false
					delimiter = ""
				}
				messages[i] += delimiter + message
			}
		}

		for _, message := range messages {
			twitchClient.Say(channel, message)
		}
	}
}

// }}}

// Helper Functions {{{
func setGlobalLogger() {
	level := zerolog.InfoLevel

	if *debug {
		// Force log level to debug
		level = zerolog.DebugLevel
		// Add file and line number to log
		log.Logger = log.With().Caller().Logger()
	}

	// Set Log Level
	zerolog.SetGlobalLevel(level)

	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}
	log.Logger = log.Output(output)
}
func censor(text string) string {
	if *showSecrets {
		return text
	}
	return "[REDACTED]"
}

func pluralize(text string, times int64) string {
	if times > 1 {
		return strconv.FormatInt(times, 10) + " " + text + "s"
	}
	return "one " + text
}

func join(log zerolog.Logger, channel string) {
	twitchClient.Join(channel)
	stateClient.JoinChannel(channel, true)
	log.Info().Str("channel", channel).Msg("Joined new channel")
}

func part(log zerolog.Logger, channel string) {
	twitchClient.Depart(channel)
	stateClient.JoinChannel(channel, false)
	log.Info().Str("channel", channel).Msg("Parted from channel")
}

// }}}

// vim: set foldmarker={{{,}}} foldmethod=marker:
