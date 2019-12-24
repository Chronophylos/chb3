package actions

import "fmt"

type noPermissionError struct {
	has    Permission
	needed Permission
}

func (e *noPermissionError) Error() string {
	return fmt.Sprintf(
		"needed permission is not high enough (has: %d, needed: %d)",
		e.has, e.needed,
	)
}

type notInBotChannel struct {
	channel string
}

func (e *notInBotChannel) Error() string {
	return fmt.Sprintf(
		"not in bot channel (actual channel: %s)",
		e.channel,
	)
}
