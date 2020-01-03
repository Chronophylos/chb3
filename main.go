package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	botRe  = "@?chronophylosbot,?"
	prefix = "~"
)

var idStore = map[string]string{
	"marc_yoyo":    "89131006",
	"chronophylos": "54946241",
}

// Build Infos
var (
	Version = "3.5.0"
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

/*
var (
	True  = true
	False = false
)
*/

// Globals
var (
	owClient     *openweather.OpenWeatherClient
	stateClient  *state.Client
	twitchClient *twitch.Client
	swearfilter  *sw.SwearFilter
	osmClient    *nominatim.Client
)

func main() {
	/*
		commands := []*Command{}
		True := true
		False := false
	*/

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
		owClient = openweather.NewOpenWeatherClient(openweatherAppID)
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

	// Commands {{{
	manager, err := cmd.NewManager(twitchClient, stateClient, owClient, osmClient, Version, twitchUsername, debug)
	if err != nil {
		log.Error().
			Err(err).
			Msg("could not create command manager")
		return
	}

	/* Old Style Commands are DISABLED.
	// Arguably Useful Commands {{{
	aC(Command{
		name: "scambot",
		re:   rl(`(?i)\bscambot\b`),
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msg("not a scambot")
			twitchClient.Say(c.Channel, "FeelsNotsureMan")
		},
	})
	// }}}

	// Hardly Useful Commands {{{
	aC(Command{
		name: "reupload",
		re: rl(
			`((https?:\/\/)?(damn-community.com)|(screenshots.relentless.wtf)\/.*\.(png|jpe?g))`,
			`((https?:\/\/)?(puddelgaming.de\/upload)\/.*\.(png|jpe?g))`,
		),
		callback: func(c *CommandEvent) {
			link := c.Match[0][1]

			// Fix links
			if !strings.HasPrefix(link, "https://") {
				if !strings.HasPrefix(link, "https://") {
					link = strings.TrimPrefix(link, "http://")
				}
				link = "https://" + link
			}

			c.Logger.Info().
				Str("link", link).
				Msg("Reuploading a link to imgur")

			newURL := reupload(link)
			if newURL != "" {
				twitchClient.Say(c.Channel, "Did you mean "+newURL+" ?")
			}
		},
	})
	// }}}
	*/
	// }}}

	// Twitch Client Event Handling {{{
	twitchClient.OnPrivateMessage(func(message twitch.PrivateMessage) {
		// Don't listen to messages sent by the bot
		if message.User.Name == twitchUsername {
			return
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

		/* Old Style Command Invokation is DISABLED
		s := &CommandState{
			IsSleeping:    sleeping,
			IsMod:         message.Tags["mod"] == "1",
			IsSubscriber:  message.Tags["subscriber"] != "0",
			IsBroadcaster: message.User.Name == message.Channel,
			IsOwner:       message.User.ID == idStore["chronophylos"],
			IsBot:         checkIfBotname(message.User.Name),
			IsBotChannel:  message.Channel == twitchUsername,
			IsTimedout:    user.IsTimedout(message.Time),

			Channel: message.Channel,
			Message: message.Message,
			Time:    message.Time,
			User:    &message.User,

			Raw: &message,
		}

		for _, c := range commands {
			if err := c.Trigger(s); err != nil {
				switch err.Error() {
				case "not enough permissions":
					fallthrough
				case "no match found":
					continue
				}

				log.Debug().
					Err(err).
					Str("command", c.name).
					Msg("Command did not get executed")
			}
		}

		if !s.IsSleeping && !s.IsTimedout {
			checkForVoicemails(message.User.Name, message.Channel)
		}
		*/
	})

	twitchClient.OnConnect(func() {
		log.Info().Msg("Connected to irc.twitch.tv")
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

	log.Info().Msg("Connecting to irc.twitch.tv")
	err = twitchClient.Connect()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to connect to irc.twitch.tv")
	}

	log.Error().Msg("Twitch Client Terminated")
}

// Command Functions {{{
// jumble {{{
// jumble uses the Fisher-Yates shuffle to shuffle a string in plaCe
func jumble(name string) string {
	a := strings.Split(name, "")
	l := len(a)

	for i := l - 2; i > 1; i-- {
		j := int32(math.Floor(rand.Float64()*float64(i+1)) + 1)
		a[i], a[j] = a[j], a[i]
	}

	return strings.Join(a, "")
}

// }}}
// Imgur Reupload {{{
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

// }}}

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
			if len(messages[0])+len(message) > 400 {
				truncate(message, 400-len(messages[0]))
			}
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
func rl(re ...string) []*regexp.Regexp {
	res := []*regexp.Regexp{}

	for _, r := range re {
		res = append(res, regexp.MustCompile(r))
	}

	return res
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

// Credit: https://stackoverflow.com/users/130095/geoff
func truncate(s string, i int) string {
	runes := []rune(s)
	if len(runes) > i {
		return string(runes[:i])
	}
	return s
}

// }}}

// vim: set foldmarker={{{,}}} foldmethod=marker:
