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
	"sync"
	"time"

	sw "github.com/JoshuaDoes/gofuckyourself"
	"github.com/Knetic/govaluate"
	"github.com/akamensky/argparse"
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
	Version = "3.4.0"
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

var (
	True  = true
	False = false
)

// Globals
var (
	owClient     *openweather.OpenWeatherClient
	stateClient  *state.Client
	twitchClient *twitch.Client
	swearfilter  *sw.SwearFilter
	osmClient    *nominatim.Client
)

func main() {
	commands := []*Command{}
	True := true
	False := false

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
	aC := func(c Command) {
		c.Init()
		commands = append(commands, &c)
	}

	// State {{{
	aC(Command{
		name:       "go sleep",
		re:         rl(`(?i)^`+prefix+`(shut up|go sleep)`, `(?i)^`+prefix+`sei ruhig`),
		permission: Moderator,
		callback: func(c *CommandEvent) {
			c.Logger.Info().Msg("Going to sleep")

			stateClient.SetSleeping(c.Channel, true)
		},
	})

	aC(Command{
		name:        "wake up",
		re:          rl(`(?i)^` + prefix + `(wake up|wach auf)`),
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
		re:   rl(`(?i)^` + prefix + `join (my channel|\w+)$`),
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
		re:   rl(`(?i)^` + prefix + `leave (\w+)$`),
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

	aC(Command{
		name:       "lurk",
		re:         rl(`(?i)^` + prefix + `lurk in (\w+)$`),
		permission: Moderator,
		callback: func(c *CommandEvent) {
			channel := strings.ToLower(c.Match[0][1])

			if c.IsBotChannel {
				twitchClient.Join(channel)
				stateClient.SetLurking(channel, true)
				stateClient.JoinChannel(channel, true)

				c.Logger.Info().Str("channel", channel).Msg("Lurking in new channel")

				twitchClient.Say(c.Channel, "I'm lurking in "+channel+" now.")
			}
		},
	})

	aC(Command{
		name:       "debug",
		re:         rl(`(?i)^` + prefix + `debug (\w+)`),
		permission: Owner,
		callback: func(c *CommandEvent) {
			action := strings.ToLower(c.Match[0][1])

			switch action {
			case "enable":
				debug = &True
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				c.Logger.Info().Msg("Enabled debugging")
				twitchClient.Say(c.Channel, "Enabled debugging")
			case "disable":
				debug = &False
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
				c.Logger.Info().Msg("Disabled debugging")
				twitchClient.Say(c.Channel, "Disabled debugging")
			}
		},
	})
	// }}}

	// Version Command {{{
	aC(Command{
		name: "version",
		re: rl(
			`(?i)^`+botRe+`\?`,
			`(?i)^`+prefix+`version`,
		),
		callback: func(c *CommandEvent) {
			twitchClient.Say(c.Channel, "I'm a bot written by Chronophylos in Golang. Current Version is "+Version+".")
			c.Logger.Info().Msg("Sending Version")
		},
	})
	// }}}

	// Voicemails {{{
	aC(Command{
		name: "leave voicemail",
		re: rl(
			`(?i)^` + prefix + `tell ((\w+)( && (\w+))*) (.*)`,
		),
		userCD: 30 * time.Second,
		callback: func(c *CommandEvent) {
			match := c.Match[0]
			usernames := []string{}
			message := match[5]

			for _, username := range strings.Split(match[1], " && ") {
				if username == twitchUsername {
					continue
				}

				if username == c.User.Name {
					continue
				}

				usernames = append(usernames, strings.ToLower(username))
			}

			if len(usernames) <= 0 {
				return
			}

			c.Logger.Info().
				Strs("usernames", usernames).
				Str("voicemessage", message).
				Str("creator", c.User.Name).
				Msg("Leaving a voicemail")

			for _, username := range usernames {
				if err := stateClient.AddVoicemail(username, c.Channel, c.User.Name, message, c.Time); err != nil {
					log.Error().
						Err(err).
						Msg("Adding Voicemail")
					return
				}
			}
			twitchClient.Say(c.Channel, "I'll forward this message to "+strings.Join(usernames, ", ")+" when they type something in chat")
		},
	})
	//}}}

	// patscheck {{{
	aC(Command{
		name: "patscheck",
		re:   rl(`(?i)^` + prefix + `hihsg\?`),
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
		channelCD: 10 * time.Second,
		userCD:    30 * time.Second,
		callback: func(c *CommandEvent) {
			if c.IsBot || c.Channel == "moondye7" {
				c.Skip()
				return
			}
			twitchClient.Say(c.Channel, "^")
		},
	})

	aC(Command{
		name: "rate",
		re:   rl(`(?i)^~rate (.*)$`),
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

			if weatherMessage := getWeather(city); weatherMessage != "" {
				twitchClient.Say(c.Channel, weatherMessage)
			}
		},
	})

	aC(Command{
		name: "location",
		re:   rl(`(?i)^wo (ist|liegt) (.*)\?+`),
		callback: func(c *CommandEvent) {
			city := c.Match[0][2]

			place, err := osmClient.GetPlace(city)
			if err != nil {
				c.Logger.Error().Err(err).Msg("Could not get place")
				twitchClient.Say(c.Channel, "Ich kann "+city+" nicht finden")
				return
			}

			twitchClient.Say(c.Channel, place.URL)

			c.Logger.Info().
				Str("city", city).
				Float64("lat", place.Lat).
				Float64("lon", place.Lon).
				Msg("Checking Coordinates")
		},
	})

	aC(Command{
		name: "math",
		re:   rl(`(?i)^` + prefix + `(math|quickmafs) (.*)$`),
		callback: func(c *CommandEvent) {
			exprString := c.Match[0][2]

			defer func() {
				if r := recover(); r != nil {
					c.Logger.Info().
						Str("expression", exprString).
						Msg("failed to do math")

					twitchClient.Say(c.Channel, "I can't calculate that :(")
				}
			}()

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
				Msg("doing math")

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
		name: "robert and wsd",
		re:   rl(`(?i)(wsd|weisserschattendraChe|louis)`),
		callback: func(c *CommandEvent) {
			if c.User.Name == "n0valis" {
				c.Logger.Info().Msg("Confusing robert")

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
			if c.User.ID != idStore["marc_yoyo"] {
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

	aC(Command{
		name: "latertonnennotregal",
		re:   rl(`(?i)\bregal\b`),
		callback: func(c *CommandEvent) {
			if c.User.Name == "nightbot" {
				c.Skip()
				return
			}

			c.Logger.Info().Msg("latertonnennotregal")
			twitchClient.Say(c.Channel, "lager")
		},
	})

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

		sleeping, err := stateClient.IsSleeping(message.Channel)
		if err != nil {
			log.Error().
				Err(err).
				Str("channel", message.Channel).
				Msg("Checking if channel is sleeping")
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
const weatherText = "Das aktuelle Wetter für %s, %s: %s bei %.1f°C. Der Wind kommt aus %s mit %.1fm/s bei einer Luftfeuchtigkeit von %d%%. Die Wettervorhersagen für morgen: %s bei %.1f°C bis %.1f°C."

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
	tomorrow := time.Date(year, month, day+1, 12, 0, 0, 0, time.UTC)
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
		currentWeather.City.Country,
		currentCondition,
		currentWeather.Temperature.Current,
		currentWeather.Wind.Direction,
		currentWeather.Wind.Speed,
		currentWeather.Humidity,
		tomorrowsConditions,
		tomorrowsWeather.Temperature.Min,
		tomorrowsWeather.Temperature.Max,
	)
}

func getLocation(city string) string {
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

	return fmt.Sprintf(
		"https://www.google.com/maps/@%f,%f,10z",
		currentWeather.Position.Latitude,
		currentWeather.Position.Longitude,
	)
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
