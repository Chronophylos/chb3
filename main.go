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
	"regexp"
	dbg "runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
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

	// Commands {{{
	aC := func(c Command) {
		c.Init()
		commands = append(commands, &c)
	}

	// State {{{
	aC(Command{
		name:       "go sleep",
		re:         rl(`(?i)^(shut up|go sleep) `+botRe, `(?i)^`+botRe+` sei ruhig`),
		permission: Moderator,
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msg("Going to sleep")

			stateClient.SetSleeping(c.Channel, true)
		},
	})

	aC(Command{
		name:        "wake up",
		re:          rl(`(?i)^(wake up|wach auf) ` + botRe),
		ignoreSleep: true,
		permission:  Moderator,
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msg("Waking up")

			stateClient.SetSleeping(c.Channel, false)
		},
	})
	// }}}

	// Admin Commands {{{
	aC(Command{
		name: "join",
		re:   rl(`(?i)^join (my channel|\w+) pls$`),
		callback: func(c *CommandEvent) {
			joinChannel := strings.ToLower(c.Match[0][1])

			if c.IsBotChannel {
				if joinChannel == "my channel" {
					if joined, err := stateClient.IsChannelJoined(c.User.Name); err != nil && joined {
						twitchClient.Say(c.Channel, "I'm already in your channel.")
					} else {
						join(c.Logger, c.User.Name)
						twitchClient.Say(c.Channel, "I joined your channel. Type `@chronophylosbot leave this channel pls` in your channel and I'll leave again.")
					}
				} else if c.IsOwner {
					if joined, err := stateClient.IsChannelJoined(joinChannel); err != nil && joined {
						twitchClient.Say(c.Channel, "I'm already in that channel.")
					} else {
						join(c.Logger, joinChannel)
						twitchClient.Say(c.Channel, "I joined "+joinChannel+". Type `leave "+joinChannel+" pls` in this channel and I'll leave again.")
					}
				}
			} else {
				c.Skip()
			}
		},
	})

	aC(Command{
		name:        "leave",
		re:          rl(`(?i)^@?chronophylosbot leave this channel pls$`),
		permission:  Moderator,
		ignoreSleep: true,
		callback: func(c *CommandEvent) {
			twitchClient.Say(c.Channel, "ppPoof")

			part(c.Logger, c.Channel)
		},
	})

	aC(Command{
		name: "leave",
		re:   rl(`(?i)^leave (\w+) pls$`),
		callback: func(c *CommandEvent) {
			partChannel := strings.ToLower(c.Match[0][1])

			if c.IsBotChannel {
				if c.IsOwner || c.User.Name == partChannel {
					part(c.Logger, partChannel)
					twitchClient.Say(c.Channel, "I left "+partChannel+".")
				}
			} else {
				c.Skip()
			}
		},
	})
	// }}}

	// Version Command {{{
	aC(Command{
		name: "version",
		re:   rl(`(?i)^` + botRe + `\?`),
		callback: func(c *CommandEvent) {
			twitchClient.Say(c.Channel, "I'm a bot by Chronophylos. Version: "+Version)
			c.Logger.Info().Msg("Sending Version")
		},
	})
	// }}}

	// Voicemails {{{
	aC(Command{
		name:   "leave voicemail",
		re:     rl(`(?i)` + botRe + ` tell (\w+) (.*)`),
		userCD: 30 * time.Second,
		callback: func(c *CommandEvent) {
			username := strings.ToLower(c.Match[0][1])
			message := c.Match[0][2]

			if username == twitchUsername {
				c.Skip()
				return
			}

			if username == c.User.Name {
				return
			}

			c.Logger.Info().
				Str("username", username).
				Str("voicemessage", message).
				Str("creator", c.User.Name).
				Msg("Leaving a voicemail")

			if err := stateClient.AddVoicemail(username, c.Channel, c.User.Name, message, c.Time); err != nil {
				log.Error().
					Err(err).
					Msg("Adding Voicemail")
				return
			}
			twitchClient.Say(c.Channel, "I'll forward this message to "+username+" when they type something in chat")
		},
	})
	//}}}

	// patscheck {{{
	aC(Command{
		name: "patscheck",
		re:   rl(`(?i)habe ich heute schon gepatscht\?`, `(?i)hihsg\?`),
		callback: func(c *CommandEvent) {
			user, err := stateClient.GetUserByID(c.User.ID)
			if err != nil {
				log.Error().
					Err(err).
					Str("id", c.User.ID).
					Msg("Could not get user")
				return
			}

			c.Logger.Info().Msg("Checking Patscher")

			if user.PatschCount == 0 {
				twitchClient.Say(c.Channel, "You've never patted the fish before. You should do that now.")
				return
			}

			streak := "Your current streak is " + strconv.Itoa(user.PatschStreak) + "."
			if user.PatschStreak == 0 {
				streak = "You don't have a streak ongoing."
			}

			total := " In total you patted " + strconv.Itoa(user.PatschCount) + " times."
			if user.PatschCount == 0 {
				total = ""
			}

			if user.HasPatschedToday(c.Time) {
				twitchClient.Say(c.Channel, "You already patted today. "+streak+total)
			} else {
				twitchClient.Say(c.Channel, "You have not yet patted today. "+streak+total)
			}
		},
	})

	aC(Command{
		name: "patsch",
		re:   rl(`fischPatsch|fishPat`),
		callback: func(c *CommandEvent) {
			if c.Channel != "furzbart" && !(*debug && c.IsBotChannel) {
				c.Skip()
				return
			}

			if len(c.Match) > 1 {
				twitchClient.Say(c.Channel, "/timeout "+c.User.Name+" 1 Wenn du so viel patschst wird das ne Flunder.")
				return
			}

			if err := stateClient.Patsch(c.User.ID, c.Time); err != nil {
				if err == state.ErrAlreadyPatsched {
					twitchClient.Say(c.Channel, "Du hast heute schon gepatscht.")
					return
				} else if err == state.ErrForgotToPatsch {
					// did not patsch
				}
			}

			c.Logger.Info().Msg("Patsch!")
		},
	})
	// }}}

	// Useful Commands {{{
	aC(Command{
		name:       "vanish reply",
		re:         rl(`^!vanish`),
		permission: Moderator,
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msgf("Telling %s how to use !vanish", c.User.Name)
			twitchClient.Say(c.Channel, "Try /unmod "+c.User.Name+" first weSmart")
		},
	})

	aC(Command{
		name:      "^",
		re:        rl(`^\^`),
		channelCD: 1 * time.Second,
		userCD:    5 * time.Second,
		callback: func(c *CommandEvent) {
			if c.IsBot {
				c.Skip()
				return
			}
			twitchClient.Say(c.Channel, "^")
		},
	})

	aC(Command{
		name: "rate",
		re:   rl(`(?i)^rate (.*) pls$`),
		callback: func(c *CommandEvent) {
			key := c.Match[0][1]
			rating := rate(key)

			c.Logger.Info().
				Str("key", key).
				Str("rating", rating).
				Msg("Rating something")

			twitchClient.Say(c.Channel, "I rate "+key+" "+rating+"/10")
		},
	})

	aC(Command{
		name: "weather",
		//disabled: true,
		re: rl(`(?i)^wie ist das wetter in (.*)\?`),
		callback: func(c *CommandEvent) {
			city := c.Match[0][1]

			c.Logger.Info().
				Str("city", city).
				Msg("Checking weather")

			weatherMessage := getWeather(city)
			if weatherMessage != "" {
				twitchClient.Say(c.Channel, weatherMessage)
			}
		},
	})

	aC(Command{
		name: "math",
		re:   rl(`(?i)^!math (.*)$`),
		callback: func(c *CommandEvent) {
			exprString := c.Match[0][1]

			expr, err := govaluate.NewEvaluableExpression(exprString)
			if err != nil {
				c.Logger.Error().
					Err(err).
					Str("expression", exprString).
					Msg("Error parsing expression")
				twitchClient.Say(c.Channel, fmt.Sprintf("Error: %v", err))
				return
			}

			result, err := expr.Evaluate(nil)
			if err != nil {
				c.Logger.Error().
					Err(err).
					Str("expression", exprString).
					Msg("Error evaluating expression")
				return
			}

			c.Logger.Info().
				Str("expression", exprString).
				Interface("result", result).
				Msg("Evaluated Math Expression")

			twitchClient.Say(c.Channel, fmt.Sprintf("%v", result))
		},
	})
	// }}}

	// Arguably Useful Commands {{{
	aC(Command{
		name:        "er dr",
		re:          rl(`er dr`),
		reactToBots: true,
		callback: func(c *CommandEvent) {
			if c.User.Name == "nightbot" {
				log.Info().Msg("Robert pressed two keys.")
				twitchClient.Say(c.Channel, "ückt voll oft zwei tasten LuL")
			} else {
				c.Skip()
			}
		},
	})

	aC(Command{
		name: "hello user",
		re:   rl(`(?i)(hey|hi|h[ea]llo) @?chronop(phylos(bot)?)?`),
		callback: func(c *CommandEvent) {

			log.Info().Msgf("Greeting %s.", c.User.DisplayName)
			twitchClient.Say(c.Channel, "Hello "+c.User.DisplayName+"👋")
		},
	})

	aC(Command{
		name:        "hello stirnbot",
		re:          rl(`^I'm here FeelsGoodMan$`),
		reactToBots: true,
		callback: func(c *CommandEvent) {
			if c.User.Name == "stirnbot" {
				c.Logger.Info().Msg("Greeting StirnBot")

				twitchClient.Say(c.Channel, "StirnBot MrDestructoid /")
			} else {
				c.Skip()
			}
		},
	})

	aC(Command{
		name: "***REMOVED*** and wsd",
		re:   rl(`(?i)(wsd|weisserschattendraChe|louis)`),
		callback: func(c *CommandEvent) {
			if c.User.Name == "n0valis" {
				c.Logger.Info().Msg("Confusing ***REMOVED***")

				twitchClient.Say(c.Channel, "did you mean me?")
			} else {
				c.Skip()
			}
		},
	})

	aC(Command{
		name: "the age of marc",
		re:   rl(`(?i)(\bmarc alter\b)|(\balter marc\b)`),
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msg("Gratulating marc for his birthday")

			twitchClient.Say(c.Channel, "marc ist heute 16 geworden FeelsBirthdayMan Clap")
		},
	})

	aC(Command{
		name: "kleiwe",
		re:   rl(`(?i)\bkleiwe\b`),
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msgf("Missspelling %s", c.User.DisplayName)

			twitchClient.Say(c.Channel, jumble(c.User.DisplayName))
		},
	})

	aC(Command{
		name: "time",
		re:   rl(`(?i)what time is it\?`),
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msg("Checking the time")

			twitchClient.Say(c.Channel, "The current time is: "+time.Now().Format(time.RFC3339))
		},
	})

	aC(Command{
		name: "marc likes u-bahnen",
		re:   rl(`(?i)md7H /`),
		callback: func(c *CommandEvent) {
			if c.User.Name != "marc_yoyo" {
				c.Skip()
				return
			}

			c.Logger.Info().Msg("greeting marcs u-bahn")
			twitchClient.Say(c.Channel, "marc U-Bahn /")
		},
	})

	aC(Command{
		name: "nymnCREB",
		re:   rl(`(?i)nymnCREB (\w+) IS GONE nymnCREB`),
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msg("nymnCREB")
			twitchClient.Say(c.Channel, "nymnCREB "+c.Match[0][1]+" IS GONE nymnCREB")
		},
	})
	// }}}

	// Hardly Useful Commands {{{
	aC(Command{
		name: "reupload",
		re:   rl(`((https?:\/\/)?(damn-community.com)|(screenshots.relentless.wtf)\/.*\.(png|jpe?g))`),
		callback: func(c *CommandEvent) {
			link := c.Match[0][1]

			// Fix links
			if !strings.HasPrefix("https://", link) {
				if !strings.HasPrefix("http://", link) {
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
	// }}}

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
