package main

import (
	"regexp"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Permission int

const (
	Everyone Permission = iota
	Subscriber
	Regular
	Moderator
	Broadcaster
	Owner
)

type Command struct {
	name       string
	re         []*regexp.Regexp
	permission Permission
	cooldown   time.Duration

	ignoreSleep bool
	reactToBots bool
	disabled    bool

	callback func(c *CommandEvent)

	lastTriggered map[string]time.Time
}

// Trigger is used to trigger a commmand
func (c *Command) Trigger(s *CommandState) bool {
	if c.disabled {
		return false
	}
	if !c.ignoreSleep && s.IsSleeping {
		return false
	}
	if s.IsTimedOut && !s.IsOwner {
		return false
	}
	if c.isCoolingDown(s.Channel, s.Time) {
		return false
	}
	if s.GetPermission() < c.permission {
		return false
	}

	// Check all regexps and stop if a match is found
	var found bool
	var match [][]string
	for _, re := range c.re {
		match = re.FindAllStringSubmatch(s.Message, -1)
		if match != nil {
			found = true
			break
		}
	}
	// quit if no match is found
	if !found {
		return false
	}

	// only get logger if we actually need it
	log := c.getLogger(s)

	log.Debug().
		Interface("match", match).
		Msg("Found matching command")

	r := &CommandEvent{
		CommandState: *s,
		Logger:       log,
		Match:        match,
	}

	// call the callback
	c.callback(r)

	if r.Skipped {
		log.Debug().Msg("Command got skipped")
	} else {
		c.lastTriggered[s.Channel] = s.Time
	}

	return !r.Skipped
}

func (c *Command) isCoolingDown(channel string, t time.Time) bool {
	zeit, present := c.lastTriggered[channel]
	if !present {
		return false
	}
	return t.Sub(zeit) < c.cooldown
}

func (c *Command) getLogger(s *CommandState) zerolog.Logger {
	return log.With().
		Str("command", c.name).
		Str("channel", s.Channel).
		Str("username", s.User.Name).
		Str("msg", s.Message).
		Logger()
}

// CommandState holds information that represent the state of a command. e.g. wheather the invoker is mod what channel and so on
type CommandState struct {
	IsSleeping    bool
	IsMod         bool
	IsSubscriber  bool
	IsBroadcaster bool
	IsOwner       bool
	IsBot         bool
	IsBotChannel  bool
	IsTimedOut    bool

	Channel string
	Message string
	Time    time.Time
	User    *twitch.User

	Raw *twitch.PrivateMessage
}

func (s *CommandState) GetPermission() Permission {
	if s.IsOwner {
		return Owner
	}
	if s.IsBroadcaster {
		return Broadcaster
	}
	if s.IsMod {
		return Moderator
	}
	// TODO: check if regular
	if s.IsSubscriber {
		return Subscriber
	}
	return Everyone
}

type CommandEvent struct {
	CommandState
	Match [][]string

	Logger  zerolog.Logger
	Skipped bool
}

func (c *CommandEvent) Skip() {
	c.Skipped = true
}
