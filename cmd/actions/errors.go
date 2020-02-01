package actions

import "fmt"

type notInBotChannelError struct {
	channel string
}

func (e *notInBotChannelError) Error() string {
	return fmt.Sprintf(
		"not in bot channel (actual channel: %s)",
		e.channel,
	)
}
