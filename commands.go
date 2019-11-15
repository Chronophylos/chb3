package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chronophylos/chb3/state"
	"github.com/rs/zerolog/log"
	"gopkg.in/Knetic/govaluate.v2"
)

func registerCommands(commands []*Command) {
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
}
