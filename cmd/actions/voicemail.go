package actions

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chronophylos/chb3/database"
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

	id, err := strconv.ParseInt(e.Msg.User.ID, 10, 64)
	if err != nil {
		return err
	}

	user, err := e.DB.GetUserByID(id)
	if err != nil {
		return err
	}

	for _, username := range recipents {
		err := e.DB.PutVoicemail(&database.Voicemail{
			Creator:  user,
			Created:  e.Msg.Time,
			Recipent: username,
			Message:  message,
		})
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
