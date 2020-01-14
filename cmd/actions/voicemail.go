package actions

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type voicemailAction struct {
	options   *Options
	seperator string
}

func newVoicemailAction() *voicemailAction {
	seperator := " && "
	return &voicemailAction{
		options: &Options{
			Name:         "leave voicmail",
			Re:           regexp.MustCompile(`(?i)^~tell ((\w+)(` + seperator + `(\w+))*) (.*)`),
			UserCooldown: 30 * time.Second,
		},
		seperator: seperator,
	}
}

func (a voicemailAction) GetOptions() *Options {
	return a.options
}

func (a voicemailAction) Run(e *Event) error {
	recipents := []string{}
	message := e.Match[5]

	for _, username := range strings.Split(e.Match[1], a.seperator) {
		username = strings.ToLower(username)
		if username == e.BotName {
			continue
		}
		if username == e.Msg.User.Name {
			continue
		}

		recipents = append(recipents, username)
	}

	if len(recipents) == 0 {
		e.Say("I will not send a message to these recipents")
		return errors.New("no valid username")
	}

	if len(message) >= 400 {
		e.Say("I'm sorry but your message is too long")
		return errors.New("message too long")
	}

	e.Log.Info().
		Strs("recipents", recipents).
		Str("voicemail", message).
		Msg("Leaving a voicmail")

	channel := e.Msg.Channel
	creator := e.Msg.User.Name
	created := e.Msg.Time

	for _, username := range recipents {
		err := e.State.AddVoicemail(username, channel, creator, message, created)
		if err != nil {
			return fmt.Errorf("could not insert voicmail into database: %v", err)
		}
	}

	var recpientString string
	if len(recipents) == 1 {
		recpientString = recipents[0]
	} else {
		n := len(recipents) - 1
		recpientString = strings.Join(recipents[:n], ", ")
		recpientString += " and " + recipents[n]
	}

	e.Say(fmt.Sprintf(
		"I'll forward this message to %s when they type in chat.",
		recpientString,
	))

	return nil
}
