package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/akamensky/argparse"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	chronophylosID = "54946241"
)

// Build Infos
var (
	Version = "local"
)

// Flags
var (
	showSecrets *bool
	debug       *bool
	logLevel    *string
)

// Config
var (
	twitchUsername string
	twitchToken    string

	imgurClientID string
)

func init() {
	// Commandline Flags {{{
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
		os.Exit(1)
	}
	// }}}

	// Setup logger
	setGlobalLogger(false)

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

	if !viper.IsSet("imgur.clientid") {
		log.Fatal().Msg("Imgur ClientID is not set.")
	}
	imgurClientID = viper.GetString("imgur.clientid")
	// }}}
}

func main() {
	log.Info().Msgf("Starting CHB3 %s", Version)

	state := NewState()

	analytics, err := NewAnalytics()
	if err != nil {
		log.Fatal().Msg("Error creating analytics logger.")
	}

	log.Info().
		Str("username", twitchUsername).
		Str("token", censor(twitchToken)).
		Msg("Creating new Twitch Client.")
	client := twitch.NewClient(twitchUsername, twitchToken)

	log.Info().Msg("Creating Command Registry")
	commandRegistry := NewCommandRegistry()

	log.Info().Msg("Registering State Commands")

	// Commands {{{
	// State {{{
	commandRegistry.Register(NewCommand("go sleep", `(?i)^(shut up|go sleep) @?chronophylosbot`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		if !(cmdState.IsMod || cmdState.IsOwner) {
			return false
		}

		log.Info().Msg("Going to sleep")

		state.SetSleeping(cmdState.Channel, true)

		return true
	}))

	commandRegistry.Register(NewCommandEx("wake up", `(?i)^wake up @?chronophylosbot`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		if !(cmdState.IsMod || cmdState.IsOwner) {
			return false
		}

		log.Info().Msg("Waking up")

		state.SetSleeping(cmdState.Channel, false)

		return true
	}, true))
	// }}}

	// Admin Commands {{{
	commandRegistry.Register(NewCommand("join", `(?i)^join (my channel|\w+) pls$`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		joinChannel := match[0][1]

		if cmdState.Channel == twitchUsername {
			if joinChannel == "my channel" {
				if state.HasChannel(cmdState.User.Name) {
					client.Say(cmdState.Channel, "I'm already in your channel.")
				} else {
					join(client, state, log, cmdState.User.Name)
					client.Say(cmdState.Channel, "I joined your channel. Type `@chronophylosbot leave this channel pls` in your channel and I'll leave again.")
				}
			} else if cmdState.IsOwner {
				if state.HasChannel(joinChannel) {
					client.Say(cmdState.Channel, "I'm already in that channel.")
				} else {
					join(client, state, log, joinChannel)
					client.Say(cmdState.Channel, "I joined "+joinChannel+". Type `leave "+cmdState.User.Name+" pls` in this channel and I'll leave again.")
				}
			}
			return true
		}
		return false
	}))

	commandRegistry.Register(NewCommandEx("leave", `(?i)^@?chronophylosbot leave this channel pls$`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		if !(cmdState.IsMod || cmdState.IsOwner) {
			return false
		}

		client.Say(cmdState.Channel, "ppPoof")
		part(client, state, log, cmdState.Channel)

		return true
	}, true))

	commandRegistry.Register(NewCommand("leave", `(?i)^leave (\w+) pls$`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		partChannel := match[0][1]
		if cmdState.Channel == twitchUsername {
			part(client, state, log, partChannel)
			client.Say(cmdState.Channel, "I left "+partChannel+".")

			return true
		}

		return false
	}))
	// }}}

	// Version Command {{{
	commandRegistry.Register(NewCommand("version", `(?i)^chronophylosbot\?`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		client.Say(cmdState.Channel, "I'm a bot by Chronophylos. Version: "+Version)
		log.Info().Msg("Sending Version")
		return true
	}))
	// }}}

	// Useful Commands {{{
	commandRegistry.Register(NewCommand("vanish reply", `^!vanish`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		if cmdState.IsMod {
			log.Info().Msgf("Telling %s how to use !vanish", cmdState.User.Name)
			client.Say(cmdState.Channel, "Try /unmod"+cmdState.User.Name+" first weSmart")
			return true
		}
		return false
	}))

	commandRegistry.Register(NewCommand("^", `^\^`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		if !cmdState.IsBot {
			client.Say(cmdState.Channel, "^")
			return true
		}
		return false
	}))

	commandRegistry.Register(NewCommand("rate", `(?i)^rate (.*) pls$`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
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
	commandRegistry.Register(NewCommand("er dr", `er dr`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		if cmdState.User.Name == "nightbot" {
			log.Info().Msg("Robert pressed two keys.")
			client.Say(cmdState.Channel, "ückt voll oft zwei tasten LuL")
			return true
		}
		return false
	}))
	commandRegistry.Register(NewCommand("hello user", `(?i)(hey|hi|h[ea]llo) @?chronop(phylos(bot)?)?`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		log.Info().Msgf("Greeting %s.", cmdState.User.DisplayName)
		client.Say(cmdState.Channel, "Hello "+cmdState.User.DisplayName+"👋")
		return true
	}))
	commandRegistry.Register(NewCommand("hello stirnbot", `^I'm here FeelsGoodMan$`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		if cmdState.User.Name == "stirnbot" {
			log.Info().Msg("Greeting StirnBot.")
			client.Say(cmdState.Channel, "StirnBot MrDestructoid /")
			return true
		}
		return false
	}))
	commandRegistry.Register(NewCommand("robert and wsd", `(?i)(wsd|weisserschattendrache|louis)`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		if cmdState.User.Name == "n0valis" {
			log.Info().Msg("Confusing robert.")
			client.Say(cmdState.Channel, "did you mean me?")
			return true
		}
		return false
	}))
	commandRegistry.Register(NewCommand("the age of marc", `(?i)(\bmarc alter\b)|(\balter marc\b)`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		log.Info().Msg("Gratulating marc for his birthday.")
		client.Say(cmdState.Channel, "marc ist heute 16 geworden FeelsBirthdayMan Clap")
		return true
	}))
	commandRegistry.Register(NewCommand("kleiwe", `(?i)\bkleiwe\b`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		log.Info().Msgf("Missspelling %s.", cmdState.User.DisplayName)
		client.Say(cmdState.Channel, jumble(cmdState.User.DisplayName))
		return true
	}))
	// }}}

	// Hardly Useful Commands {{{
	commandRegistry.Register(NewCommand("reupload", `((https?:\/\/)?(damn-community.com)|(screenshots.relentless.wtf)\/.*\.(png|jpe?g))`, func(cmdState *CommandState, log zerolog.Logger, match Match) bool {
		link := match[0][1]

		// Fix links
		if !strings.HasPrefix("https://", link) {
			if !strings.HasPrefix("http://", link) {
				link = strings.TrimPrefix(link, "http://")
			}
			link = "https://" + link
		}

		log.Info().
			Str("link", link).
			Msg("Reuploading a link to imgur")

		newURL := reupload(link)
		if newURL != "" {
			client.Say(cmdState.Channel, "Did you mean "+newURL+" ?")
			return true
		}
		return false
	}))
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

		// Don't listen to messages sent by the bot
		if message.User.Name == twitchUsername {
			return
		}

		cmdState := &CommandState{
			IsSleeping:    state.IsSleeping(message.Channel),
			IsMod:         false,
			IsBroadcaster: message.User.Name == message.Channel,
			IsOwner:       message.User.ID == chronophylosID,
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
	// Make sure the bot is always in it's own channel
	client.Join(twitchUsername)
	state.AddChannel(twitchUsername)
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

func join(client *twitch.Client, state *State, log zerolog.Logger, channel string) {
	client.Join(channel)
	state.AddChannel(channel)
	log.Info().Str("channel", channel).Msg("Joined new channel")
}

func part(client *twitch.Client, state *State, log zerolog.Logger, channel string) {
	client.Depart(channel)
	state.RemoveChannel(channel)
	log.Info().Str("channel", channel).Msg("Parted from channel")
}

func setGlobalLogger(json bool) {
	zerolog.TimeFieldFormat = time.StampMilli
	level := zerolog.InfoLevel

	if *debug {
		// Force log level to debug
		*logLevel = "DEBUG"
		// Add file and line number to log
		log.Logger = log.With().Caller().Logger()
	}

	// Get zerolog.Level from stringimgurClientID
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
		output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.StampMilli}
		log.Logger = log.Output(output)
	}
}

func checkIfBotname(name string) bool {
	switch name {
	case "nightbot":
		fallthrough
	case "fossabot":
		fallthrough
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
		j := int32(math.Floor(rand.Float64()*float64(i+1)) + 1)
		a[i], a[j] = a[j], a[i]
	}

	return strings.Join(a, "")
}

type imgurBody struct {
	Data    imgurBodyData `json:"data"`
	Success bool          `json:"success"`
}

type imgurBodyData struct {
	Link  string `json:"link"`
	Error string `json:"error"`
}

func reupload(link string) string {
	client := &http.Client{}

	form := url.Values{}
	form.Add("image", link)

	req, err := http.NewRequest("POST", "https://api.imgur.com/3/upload", strings.NewReader(form.Encode()))
	if err != nil {
		log.Error().
			Err(err).
			Str("link", link).
			Msg("Error creating POST request for https://api.imgur.com/3/upload")
		return ""
	}

	req.Header.Add("Authorization", "Client-ID "+imgurClientID)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	log.Debug().
		Str("link", link).
		Str("client-id", censor(imgurClientID)).
		Msg("Posting url to imgur")

	resp, err := client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("link", link).
			Str("client-id", censor(imgurClientID)).
			Msg("Error posting url to imgur")
		return ""
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().
			Err(err).
			Str("link", link).
			Msg("Error reading bytes from request body")
		return ""
	}

	log.Debug().
		Str("body", string(bytes[:])).
		Msg("Read bytes from body")

	var body imgurBody
	err = json.Unmarshal(bytes, &body)
	if err != nil {
		log.Error().
			Err(err).
			Str("link", link).
			Msg("Error unmarshalling response from imgur")
		return ""
	}

	log.Debug().
		Bool("success", body.Success).
		Str("old-link", link).
		Str("new-link", body.Data.Link).
		Msg("Got an answer from imgur")

	if !body.Success {
		log.Error().
			Str("error", body.Data.Error).
			Msg("Imgur API returned an error")
		return ""
	}

	return body.Data.Link
}

func censor(text string) string {
	if *showSecrets {
		return text
	}
	return "[REDACTED]"
}

// vim: set foldmarker={{{,}}} foldmethod=marker:
