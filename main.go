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
	"strconv"
	"strings"
	"time"

	"github.com/akamensky/argparse"
	"github.com/chronophylos/chb3/openweather"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gopkg.in/Knetic/govaluate.v2"
)

const (
	chronophylosID = "54946241"
)

// Build Infos
var (
	Version = "3.2.0"
)

// Flags
var (
	showSecrets *bool
	debug       *bool
	logLevel    *string
	daemon      *bool
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
	openweatherClient *openweather.OpenWeatherClient
	state             *State
	client            *twitch.Client
)

func main() {
	commands := []*Command{}

	// Commandline Flags {{{
	// Create new parser
	parser := argparse.NewParser("chb3", "ChronophylosBot but version 3")

	debug = parser.Flag("", "debug",
		&argparse.Options{Help: "Enable debugging. Sets --level=DEBUG."})

	daemon = parser.Flag("", "daemon",
		&argparse.Options{Help: "Run as a daemon."})

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
			log.Panic().
				Interface("error", err).
				Msg("Panic!")
		}
	}()
	// }}}

	state = LoadState()

	log.Info().
		Str("appid", censor(openweatherAppID)).
		Msg("Creating new OpenWeather Client.")
	openweatherClient = openweather.NewOpenWeatherClient(openweatherAppID)

	analytics, err := NewAnalytics()
	if err != nil {
		log.Fatal().Msg("Error creating analytics logger.")
	}

	log.Info().
		Str("username", twitchUsername).
		Str("token", censor(twitchToken)).
		Msg("Creating new Twitch Client.")
	client = twitch.NewClient(twitchUsername, twitchToken)

	// Commands {{{
	aC := func(c Command) {
		c.Init()
		commands = append(commands, &c)
	}

	// State {{{
	aC(Command{
		name:       "go sleep",
		re:         rl(`(?i)^(shut up|go sleep) @?chronophylosbot`, `(?i)^@?chronophylosbot sei ruhig`),
		permission: Moderator,
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msg("Going to sleep")

			state.SetSleeping(c.Channel, true)
		},
	})

	aC(Command{
		name:        "wake up",
		re:          rl(`(?i)^(wake up|wach auf) @?chronophylosbot`),
		ignoreSleep: true,
		permission:  Moderator,
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msg("Waking up")

			state.SetSleeping(c.Channel, false)
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
					if state.HasChannel(c.User.Name) {
						client.Say(c.Channel, "I'm already in your channel.")
					} else {
						join(client, state, c.Logger, c.User.Name)
						client.Say(c.Channel, "I joined your channel. Type `@chronophylosbot leave this channel pls` in your channel and I'll leave again.")
					}
				} else if c.IsOwner {
					if state.HasChannel(joinChannel) {
						client.Say(c.Channel, "I'm already in that channel.")
					} else {
						join(client, state, c.Logger, joinChannel)
						client.Say(c.Channel, "I joined "+joinChannel+". Type `leave "+joinChannel+" pls` in this channel and I'll leave again.")
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
			client.Say(c.Channel, "ppPoof")

			part(client, state, c.Logger, c.Channel)
		},
	})

	aC(Command{
		name: "leave",
		re:   rl(`(?i)^leave (\w+) pls$`),
		callback: func(c *CommandEvent) {
			partChannel := strings.ToLower(c.Match[0][1])

			if c.IsBotChannel {
				if c.IsOwner || c.User.Name == partChannel {
					part(client, state, c.Logger, partChannel)
					client.Say(c.Channel, "I left "+partChannel+".")
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
		re:   rl(`(?i)^chronophylosbot\?`),
		callback: func(c *CommandEvent) {
			client.Say(c.Channel, "I'm a bot by Chronophylos. Version: "+Version)
			c.Logger.Info().Msg("Sending Version")
		},
	})
	// }}}

	// Voicemails {{{
	aC(Command{
		name:   "leave voicemail",
		re:     rl(`(?i)@?chronophylosbot tell (\w+) (.*)`),
		userCD: 30 * time.Second,
		callback: func(c *CommandEvent) {
			username := strings.ToLower(c.Match[0][1])
			message := c.Match[0][2]

			if username == twitchUsername {
				c.Skip()
				return
			}

			c.Logger.Info().
				Str("username", username).
				Str("voicemessage", message).
				Str("creator", c.User.Name).
				Msg("Leaving a voicemail")

			state.AddVoicemail(username, c.Channel, c.User.Name, message, c.Time)
			client.Say(c.Channel, "I'll forward this message to "+username+" when they type something in chat")
		},
	})
	//}}}

	// patscheck {{{
	aC(Command{
		name: "patscheck",
		re:   rl(`(?i)habe ich heute schon gepatscht\?`, `(?i)hihsg\?`),
		callback: func(c *CommandEvent) {
			patscher := state.GetPatscher(c.User.Name)

			c.Logger.Info().
				Interface("patscher", patscher).
				Msg("Checking Patscher")

			if patscher.Count == 0 {
				client.Say(c.Channel, "You've never patted the fish before. You should do that now.")
				return
			}

			streak := "Your current streak is " + strconv.Itoa(patscher.Streak) + "."
			if patscher.Streak == 0 {
				streak = "You don't have a streak ongoing."
			}

			total := " In total you patted " + strconv.Itoa(patscher.Count) + " times."
			if patscher.Count == 0 {
				total = ""
			}

			if patscher.HasPatschedToday(c.Time) {
				client.Say(c.Channel, "You already patted today. "+streak+total)
			} else {
				client.Say(c.Channel, "You have not yet patted today. "+streak+total)
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
				client.Say(c.Channel, "/timeout "+c.User.Name+" 1 Wenn du so viel patschst wird das ne Flunder.")
				return
			}

			if state.HasPatschedToday(c.User.Name, c.Time) {
				client.Say(c.Channel, "Du hast heute schon gepatscht.")
				return
			}

			state.Patsch(c.User.Name, c.Time)
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
			client.Say(c.Channel, "Try /unmod"+c.User.Name+" first weSmart")
		},
	})

	aC(Command{
		name:      "^",
		re:        rl(`^\^`),
		channelCD: 1 * time.Second,
		userCD:    5 * time.Second,
		callback: func(c *CommandEvent) {
			client.Say(c.Channel, "^")
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

			client.Say(c.Channel, "I rate "+key+" "+rating+"/10")
		},
	})

	aC(Command{
		name: "weather",
		//disabled: true,
		re: rl(`(?i)^wie ist das wetter in (\w+)\??`),
		callback: func(c *CommandEvent) {
			city := c.Match[0][1]

			c.Logger.Info().
				Str("city", city).
				Msg("Checking weather")

			weatherMessage := getWeather(city)
			if weatherMessage != "" {
				client.Say(c.Channel, weatherMessage)
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
				client.Say(c.Channel, fmt.Sprintf("Error: %v", err))
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

			client.Say(c.Channel, fmt.Sprintf("%v", result))
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
				client.Say(c.Channel, "ückt voll oft zwei tasten LuL")
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
			client.Say(c.Channel, "Hello "+c.User.DisplayName+"👋")
		},
	})

	aC(Command{
		name:        "hello stirnbot",
		re:          rl(`^I'm here FeelsGoodMan$`),
		reactToBots: true,
		callback: func(c *CommandEvent) {
			if c.User.Name == "stirnbot" {
				c.Logger.Info().Msg("Greeting StirnBot")

				client.Say(c.Channel, "StirnBot MrDestructoid /")
			} else {
				c.Skip()
			}
		},
	})

	aC(Command{
		name: "robert and wsd",
		re:   rl(`(?i)(wsd|weisserschattendraChe|louis)`),
		callback: func(c *CommandEvent) {
			if c.User.Name == "n0valis" {
				c.Logger.Info().Msg("Confusing robert")

				client.Say(c.Channel, "did you mean me?")
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

			client.Say(c.Channel, "marc ist heute 16 geworden FeelsBirthdayMan Clap")
		},
	})

	aC(Command{
		name: "kleiwe",
		re:   rl(`(?i)\bkleiwe\b`),
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msgf("Missspelling %s", c.User.DisplayName)

			client.Say(c.Channel, jumble(c.User.DisplayName))
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
				client.Say(c.Channel, "Did you mean "+newURL+" ?")
			}
		},
	})
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
			Interface("tags", message.Tags).
			Str("username", message.User.Name).
			Str("msg-id", message.ID).
			Str("msg", message.Message).
			Interface("tags", message.Tags).
			Msg("PRIVMSG")

		// Don't listen to messages sent by the bot
		if message.User.Name == twitchUsername {
			return
		}

		message.Message = strings.ReplaceAll(message.Message, "\U000e0000", "")
		message.Message = strings.TrimSpace(message.Message)

		s := &CommandState{
			IsSleeping:    state.IsSleeping(message.Channel),
			IsMod:         message.Tags["mod"] == "1",
			IsSubscriber:  message.Tags["subscriber"] != "0",
			IsBroadcaster: message.User.Name == message.Channel,
			IsOwner:       message.User.ID == chronophylosID,
			IsBot:         checkIfBotname(message.User.Name),
			IsBotChannel:  message.Channel == twitchUsername,
			IsTimedOut:    state.IsTimedOut(message.User.Name, message.Time),

			Channel: message.Channel,
			Message: message.Message,
			Time:    message.Time,
			User:    &message.User,

			Raw: &message,
		}

		for _, c := range commands {
			if err := c.Trigger(s); err != nil {
				if err.Error() == "no match found" {
					continue
				}

				log.Debug().
					Err(err).
					Str("command", c.name).
					Msg("Command did not get executed")
			}
		}

		if !s.IsSleeping && !s.IsTimedOut {
			checkForVoicemails(client, state, message.User.Name, message.Channel)
		}
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

	client.OnUserJoinMessage(func(message twitch.UserJoinMessage) {
		analytics.Log().
			Str("channel", message.Channel).
			Str("username", message.User).
			Msg("USERJOIN")

		//checkForVoicemails(client, state, message.User, message.Channel)
	})

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

	client.OnUserPartMessage(func(message twitch.UserPartMessage) {
		analytics.Log().
			Str("channel", message.Channel).
			Str("username", message.User).
			Msg("USERPARt")
	})

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

	log.Info().
		Str("own-channel", twitchUsername).
		Interface("channels", state.GetChannels()).
		Msg("Joining Channels")
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
func checkForVoicemails(client *twitch.Client, state *State, username, channel string) {
	if state.HasVoicemail(username) {

		voicemails := state.PopVoicemails(username)

		log.Info().
			Int("count", len(voicemails)).
			Str("username", username).
			Msg("Replaying voicemails")

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
			client.Say(channel, message)
		}

	}
}

// }}}
// weather {{{
const weatherText = "Das aktuelle Wetter für %s: %s bei %.1f°C. Der Wind kommt aus %s mit %.1fm/s. Die Wettervorhersagen für morgen: %s bei %.1f°C bis %.1f°C."

func getWeather(city string) string {
	currentWeather, err := openweatherClient.GetCurrentWeatherByName(city)
	if err != nil {
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

	weatherForecast, err := openweatherClient.GetWeatherForecastByName(city)
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
func rl(re ...string) []*regexp.Regexp {
	res := []*regexp.Regexp{}

	for _, r := range re {
		res = append(res, regexp.MustCompile(r))
	}

	return res
}

func censor(text string) string {
	if *showSecrets && !*daemon {
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

func setGlobalLogger() {
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

	if !*daemon {
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
