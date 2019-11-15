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
	"os/signal"
	dbg "runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/akamensky/argparse"
	"github.com/chronophylos/chb3/openweather"
	"github.com/chronophylos/chb3/state"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	chronophylosID = "54946241"
	botRe          = "@?chronophylosbot,?"
)

// Build Infos
var (
	Version = "3.3.0"
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

	openweatherAppID string
)

// Globals
var (
	owClient     *openweather.OpenWeatherClient
	stateClient  *state.Client
	twitchClient *twitch.Client
)

func main() {
	commands := []*Command{}

	// Commandline Flags {{{
	// Create new parser
	parser := argparse.NewParser("chb3", "ChronophylosBot but version 3")

	debug = parser.Flag("", "debug",
		&argparse.Options{Help: "Enable debugging. Sets --level=DEBUG."})

	logLevel = parser.Selector("", "level",
		[]string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL", "PANIC"},
		&argparse.Options{Default: "INFO", Help: "Set Log Level."})

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

	// Panics {{{
	defer func() {
		if err := recover(); err != nil {
			log.Error().
				Interface("error", err).
				Msg("Panic!")
			dbg.PrintStack()
		}
	}()
	// }}}

	log.Info().Msg("Creating State Client")
	stateClient, err = state.NewClient("mongodb://localhost:27017")
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Could not create State Client")
	}

	log.Info().
		Str("appid", censor(openweatherAppID)).
		Msg("Creating new OpenWeather Client")
	owClient = openweather.NewOpenWeatherClient(openweatherAppID)

	log.Info().
		Str("username", twitchUsername).
		Str("token", censor(twitchToken)).
		Msg("Creating new Twitch Client")
	twitchClient = twitch.NewClient(twitchUsername, twitchToken)

	registerCommands(commands)

	// Twich Client Event Handling {{{
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

		sleeping, err := stateClient.IsSleeping(message.Channel)
		if err != nil {
			log.Error().
				Err(err).
				Str("channel", message.Channel).
				Msg("Checking if channel is sleeping")
			return
		}

		s := &CommandState{
			IsSleeping:    sleeping,
			IsMod:         message.Tags["mod"] == "1",
			IsSubscriber:  message.Tags["subscriber"] != "0",
			IsBroadcaster: message.User.Name == message.Channel,
			IsOwner:       message.User.ID == chronophylosID,
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
// rate {{{
func rate(key string) string {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(key)))
	p, _ := strconv.ParseInt(hash, 16, 64)
	q := float32(p%101) / 10
	return fmt.Sprintf("%.1f", q)
}

// }}}
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
// weather {{{
const weatherText = "Das aktuelle Wetter für %s: %s bei %.1f°C. Der Wind kommt aus %s mit %.1fm/s. Die Wettervorhersagen für morgen: %s bei %.1f°C bis %.1f°C."

func getWeather(city string) string {
	currentWeather, err := owClient.GetCurrentWeatherByName(city)
	if err != nil {
		if err.Error() == "OpenWeather API returned an error with code 404: city not found" {
			log.Warn().
				Err(err).
				Str("city", city).
				Msg("City not found")
			return fmt.Sprintf("Ich kann %s nicht finden", city)
		}
		log.Error().
			Err(err).
			Str("city", city).
			Msg("Error getting current weather")
		return ""
	}

	conditions := []string{}
	for _, condition := range currentWeather.Conditions {
		conditions = append(conditions, condition.Description)
	}

	currentCondition := strings.Join(conditions, " und ")

	weatherForecast, err := owClient.GetWeatherForecastByName(city)
	if err != nil {
		log.Error().
			Err(err).
			Str("city", city).
			Msg("Error getting weather forecast")
	}

	var tomorrowsWeather *openweather.Weather
	year, month, day := time.Now().Date()
	tomorrow := time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
	for _, weather := range weatherForecast {
		if weather.Time == tomorrow {
			tomorrowsWeather = weather
			break
		}
	}

	conditions = []string{}
	for _, condition := range tomorrowsWeather.Conditions {
		conditions = append(conditions, condition.Description)
	}

	tomorrowsConditions := strings.Join(conditions, " und ")

	return fmt.Sprintf(weatherText,
		currentWeather.City.Name,
		currentCondition,
		currentWeather.Temperature.Current,
		currentWeather.Wind.Direction,
		currentWeather.Wind.Speed,
		tomorrowsConditions,
		tomorrowsWeather.Temperature.Min,
		tomorrowsWeather.Temperature.Max,
	)
}

// }}}
// }}}

// Helper Functions {{{
func setGlobalLogger() {
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

	if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
		// Pretty logging
		output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.StampMilli}
		log.Logger = log.Output(output)
	}
}

// }}}

// vim: set foldmarker={{{,}}} foldmethod=marker:
