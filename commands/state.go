package commands

import (
	chb3 "github.com/chronophylos/chb3/pkg"
	"github.com/gempir/go-twitch-irc/v2"
)

func InitStateCommands(r *chb3.CommandRegistry, state *chb3.State) {
	r.Register(chb3.NewCommandEx(`(?i)^@?chronophylosbot leave this channel pls$`, func(client *twitch.Client, cmdState *chb3.CommandState, match [][]string) bool {
		log := chb3.GetLogger(cmdState)

		if !(cmdState.IsMod || cmdState.IsOwner) {
			return false
		}

		log.Info().Msg("Leaving Channel")

		client.Say(cmdState.Channel, "ppPoof")
		client.Depart(cmdState.Channel)

		return true
	}, true))

	r.Register(chb3.NewCommand(`(?i)^shut up @?chronophylosbot`, func(client *twitch.Client, cmdState *chb3.CommandState, match [][]string) bool {
		log := chb3.GetLogger(cmdState)

		if !(cmdState.IsMod || cmdState.IsOwner) {
			return false
		}

		log.Info().Msg("Going to sleep")

		state.SetSleeping(cmdState.Channel, true)

		return true
	}))

	r.Register(chb3.NewCommandEx(`(?i)^wake up @?chronophylosbot`, func(client *twitch.Client, cmdState *chb3.CommandState, match [][]string) bool {
		log := chb3.GetLogger(cmdState)

		if !(cmdState.IsMod || cmdState.IsOwner) {
			return false
		}

		log.Info().Msg("Waking up")

		state.SetSleeping(cmdState.Channel, false)

		return true
	}, true))
}
