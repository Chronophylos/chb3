package main

import (
	"crypto/md5"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/akamensky/argparse"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	secretFile = ".bot_secret"
)

// Build Infos
var (
	Version = "debug"
)

// Flags
var (
	showSecrets *bool
	debug       *bool
	logLevel    *string
)

func init() {
	// Create new parser
	parser := argparse.NewParser("chb3", "ChronophylosBot but version 3")

	debug = parser.Flag("", "debug",
		&argparse.Options{Help: "Enable debugging. Shorthand for --level=DEBUG"})

	logLevel = parser.Selector("", "level",
		[]string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL", "PANIC"},
		&argparse.Options{Default: "INFO", Help: "Set Log Level"})

	showSecrets = parser.Flag("", "show-secrets",
		&argparse.Options{Help: "Show secrets in log (eg. your twitch token)"})

	// Parse Flags
	err := parser.Parse(os.Args)
	if err != nil {
		// Print usage for err
		fmt.Print(parser.Usage(err))
	}
}

func main() {
	setGlobalLogger(false)

	log.Info().Msgf("Starting CHB3 %s", Version)

	secret := NewSecret(secretFile)
	state := NewState(".bot_state")

	analytics, err := NewAnalytics()
	if err != nil {
		log.Fatal().Msg("Error creating analytics logger")
	}

	token := "[REDACTED]"
	if *showSecrets {
		token = secret.Twitch.Token
	}

	log.Info().
		Str("username", secret.Twitch.Username).
		Str("token", token).
		Msg("Creating new Twitch Client")
	client := twitch.NewClient(secret.Twitch.Username, secret.Twitch.Token)

	log.Info().Msg("Creating Command Registry")
	commandRegistry := NewCommandRegistry()

	log.Info().Msg("Registering State Commands")

	// Commands {{{
	// State {{{
	commandRegistry.Register(NewCommandEx(`(?i)^@?chronophylosbot leave this channel pls$`, func(cmdState *CommandState, match Match) bool {
		log := GetLogger(cmdState)

		if !(cmdState.IsMod || cmdState.IsOwner) {
			return false
		}

		log.Info().Msg("Leaving Channel")

		client.Say(cmdState.Channel, "ppPoof")
		client.Depart(cmdState.Channel)

		return true
	}, true))

	commandRegistry.Register(NewCommand(`(?i)^shut up @?chronophylosbot`, func(cmdState *CommandState, match Match) bool {
		log := GetLogger(cmdState)

		if !(cmdState.IsMod || cmdState.IsOwner) {
			return false
		}

		log.Info().Msg("Going to sleep")

		state.SetSleeping(cmdState.Channel, true)

		return true
	}))

	commandRegistry.Register(NewCommandEx(`(?i)^wake up @?chronophylosbot`, func(cmdState *CommandState, match Match) bool {
		log := GetLogger(cmdState)

		if !(cmdState.IsMod || cmdState.IsOwner) {
			return false
		}

		log.Info().Msg("Waking up")

		state.SetSleeping(cmdState.Channel, false)

		return true
	}, true))
	// }}}

	// Version Command {{{
	commandRegistry.Register(NewCommand(`(?i)^chronophylosbot\?`, func(cmdState *CommandState, match Match) bool {
		client.Say(cmdState.Channel, "I'm a bot by Chronophylos. Version: "+Version)
		return true
	}))
	// }}}

	// Merlins Commands aka Spell Checker {{{
	commandRegistry.Register(NewCommand(`.*`, func(cmdState *CommandState, match Match) bool {
		// TODO: Implement
		return false
	}))
	// }}}

	// Useful Commands {{{
	commandRegistry.Register(NewCommand(`^!vanish`, func(cmdState *CommandState, match Match) bool {
		if cmdState.IsMod {
			GetLogger(cmdState).Info().
				Msgf("Telling %s how to use !vanish", cmdState.User.Name)
			client.Say(cmdState.Channel, "Try /unmod"+cmdState.User.Name+" first weSmart")
			return true
		}
		return false
	}))

	commandRegistry.Register(NewCommand(`^\^`, func(cmdState *CommandState, match Match) bool {
		if !cmdState.IsBot {
			client.Say(cmdState.Channel, "^")
			return true
		}
		return false
	}))

	commandRegistry.Register(NewCommand(`(?i)^rate (.*) pls$`, func(cmdState *CommandState, match Match) bool {
		log := GetLogger(cmdState)
		key := match[0][1]
		rating := rate(key)
		log.Info().
			Str("key", key).
			Str("rating", rating).
			Msg("Rating something")
		client.Say(cmdState.Channel, "I rate "+key+" "+rating+"/10")
		return true
	}))
	/// weather
	// }}}

	// Arguably Useful Commands {{{
	commandRegistry.Register(NewCommand(`er dr`, func(cmdState *CommandState, match Match) bool {
		if cmdState.User.Name == "nightbot" {
			client.Say(cmdState.Channel, "ückt voll oft zwei tasten LuL")
			return true
		}
		return false
	}))
	commandRegistry.Register(NewCommand(`(?i)(hey|hi|h[ea]llo) @?chronop(phylos(bot)?)?`, func(cmdState *CommandState, match Match) bool {
		client.Say(cmdState.Channel, "Hello "+cmdState.User.DisplayName+"👋")
		return true
	}))
	commandRegistry.Register(NewCommand(`^I'm here FeelsGoodMan$`, func(cmdState *CommandState, match Match) bool {
		if cmdState.User.Name == "stirnbot" {
			log.Info().Msg("Greeting StrinBot")
			client.Say(cmdState.Channel, "StirnBot MrDestructoid /")
			return true
		}
		return false
	}))
	commandRegistry.Register(NewCommand(`(?i)(wsd|weisserschattendrache|louis)`, func(cmdState *CommandState, match Match) bool {
		if cmdState.User.Name == "n0valis" {
			client.Say(cmdState.Channel, "did you mean me?")
			return true
		}
		return false
	}))
	commandRegistry.Register(NewCommand(`(?i)(\bmarc alter\b)|(\balter marc\b)`, func(cmdState *CommandState, match Match) bool {
		client.Say(cmdState.Channel, "marc ist heute 16 geworden FeelsBirthdayMan Clap")
		return true
	}))
	commandRegistry.Register(NewCommand(`(?i)\bkleiwe\b`, func(cmdState *CommandState, match Match) bool {
		client.Say(cmdState.Channel, jumble(cmdState.User.DisplayName))
		return true
	}))
	// }}}

	// Hardly Useful Commands {{{
	/// reupload
	// }}}
	// }}}

	// Twich Client Event Handling {{{
	client.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		analytics.Log().
			Time("sent", message.Time).
			Str("channel", message.Channel).
			Int("duration", message.BanDuration).
			Str("target", message.TargetUsername).
			Msg("CLEARCHAT")
	})

	client.OnClearMessage(func(message twitch.ClearMessage) {
		analytics.Log().
			Str("channel", message.Channel).
			Str("invoker", message.Login).
			Str("msg", message.Message).
			Str("target-message-id", message.TargetMsgID).
			Msg("CLEARMSG")
	})

	client.OnNamesMessage(func(message twitch.NamesMessage) {
		analytics.Log().
			Str("channel", message.Channel).
			Interface("users", message.Users)
	})

	client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		analytics.Log().
			Str("channel", message.Channel).
			Str("msg", message.Message).
			Str("msg-id", message.MsgID).
			Msg("NOTICE")
	})

	client.OnPingMessage(func(message twitch.PingMessage) {})
	client.OnPongMessage(func(message twitch.PongMessage) {})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		analytics.Log().
			Time("sent", message.Time).
			Str("channel", message.Channel).
			Str("username", message.User.Name).
			Str("msg-id", message.ID).
			Str("msg", message.Message).
			Interface("tags", message.Tags).
			Msg("PRIVMSG")

		cmdState := &CommandState{
			IsSleeping:    state.IsSleeping(message.Channel),
			IsMod:         false,
			IsBroadcaster: message.User.Name == message.Channel,
			IsOwner:       message.User.ID == "54946241",
			IsBot:         checkIfBotname(message.User.Name),

			Channel: message.Channel,
			Message: message.Message,
			Time:    message.Time,
			User:    &message.User,

			Raw: &message,
		}

		commandRegistry.Trigger(cmdState)
	})

	client.OnReconnectMessage(func(message twitch.ReconnectMessage) {
		analytics.Log().Msg("RECONNECT")
	})

	client.OnRoomStateMessage(func(message twitch.RoomStateMessage) {
		analytics.Log().
			Str("channel", message.Channel).
			Str("msg", message.Message).
			Interface("state", message.State).
			Msg("ROOMSTATE")
	})

	// not logged for now
	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {})

	client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		analytics.Log().
			Time("sent", message.Time).
			Str("channel", message.Channel).
			Str("msg", message.Message).
			Str("msg-id", message.MsgID).
			Interface("msg-params", message.MsgParams).
			Str("system-msg", message.SystemMsg).
			Msg("USERNOTICE")
	})

	// not logged for now
	client.OnUserPartMessage(func(message twitch.UserPartMessage) {})

	client.OnUserStateMessage(func(message twitch.UserStateMessage) {
		analytics.Log().
			Str("username", message.User.Name).
			Str("channel", message.Channel).
			Str("msg", message.Message).
			Interface("emotesets", message.EmoteSets).
			Msg("USERSTATE")
	})

	client.OnWhisperMessage(func(message twitch.WhisperMessage) {
		analytics.Log().
			Str("username", message.User.Name).
			Str("msg", message.Message).
			Str("msg-id", message.MessageID).
			Str("thread-id", message.ThreadID).
			Msg("WHISPER")
	})

	client.OnConnect(func() {
		analytics.Log().Msg("CONNECTED")
		log.Info().Msg("Connected to irc.twitch.tv")
	})
	// }}}

	log.Info().Msg("Joining Channels")
	client.Join("chronophylos")
	client.Join(state.GetChannels()...)

	log.Info().Msg("Connecting to irc.twitch.tv")
	err = client.Connect()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to connect to irc.twitch.tv")
	}

	log.Error().Msg("Twitch Client Terminated")
}

func setGlobalLogger(json bool) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level := zerolog.InfoLevel

	// Force log level to debug
	if *debug {
		*logLevel = "DEBUG"
	}

	// Get zerolog.Level from string
	switch *logLevel {
	case "DEBUG":
		level = zerolog.DebugLevel
		break
	case "INFO":
		level = zerolog.InfoLevel
		break
	case "WARN":
		level = zerolog.WarnLevel
		break
	case "ERROR":
		level = zerolog.ErrorLevel
		break
	case "PANIC":
		level = zerolog.PanicLevel
		break
	}

	// Set Log Level
	zerolog.SetGlobalLevel(level)

	if !json {
		// Pretty logging
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Add file and line number to log
	log.Logger = log.With().Caller().Logger()
}

func checkIfBotname(name string) bool {
	switch name {
	case "nightbot":
	case "fossabot":
	case "streamelements":
		return true
	}
	return false
}

func rate(key string) string {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(key)))
	p, _ := strconv.ParseInt(hash, 16, 64)
	q := float32(p%101) / 10
	return fmt.Sprintf("%.1f", q)
}

// jumble uses the Fisher-Yates shuffle to shuffle a string in place
func jumble(name string) string {
	a := strings.Split(name, "")
	l := len(a)

	for i := l - 2; i > 1; i-- {
		j := math.Floor(rand.Float64()*(i+1)) + 1
		a[i], a[j] = a[j], a[i]
	}

	return strings.Join(a, "")
}

// vim: set foldmarker={{{,}}} foldmethod=marker:
