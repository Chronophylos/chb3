package main

import (
	"errors"
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

	userCD    time.Duration
	channelCD time.Duration

	ignoreSleep bool
	reactToBots bool
	disabled    bool

	callback func(c *CommandEvent)

	lastTriggered struct {
		channels map[string]time.Time
		users    map[string]time.Time
	}
}

func (c *Command) Init() {
	c.lastTriggered.channels = make(map[string]time.Time)
	c.lastTriggered.users = make(map[string]time.Time)
}

// Trigger is used to trigger a commmand
func (c *Command) Trigger(s *CommandState) error {
	if c.disabled {
		return errors.New("command is disabled")
	}
	if !c.ignoreSleep && s.IsSleeping {
		return errors.New("bot is sleeping")
	}
	if s.IsTimedout && !s.IsOwner {
		return errors.New("user is timed out")
	}
	if c.isCoolingDown(s.Channel, s.User.Name, s.Time) {
		return errors.New("command is cooling down")
	}
	if s.GetPermission() < c.permission {
		return errors.New("not enough permissions")
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
		return errors.New("no match found")
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
		return errors.New("command got skipped")
	}

	c.resetCooldown(s.Channel, s.User.Name, s.Time)

	return nil
}

func (c *Command) isCoolingDown(channel, username string, t time.Time) bool {
	return c.isChannelCoolingDown(channel, t) || c.isUserCoolingDown(username, t)
}

func (c *Command) isChannelCoolingDown(channel string, t time.Time) bool {
	zeit, present := c.lastTriggered.channels[channel]
	if !present {
		return false
	}
	diff := t.Sub(zeit)
	return diff < c.channelCD
}

func (c *Command) isUserCoolingDown(username string, t time.Time) bool {
	zeit, present := c.lastTriggered.users[username]
	if !present {
		return false
	}
	diff := t.Sub(zeit)
	return diff < c.userCD
}

// resetCooldown sets lastTriggered
func (c *Command) resetCooldown(channel, username string, t time.Time) {
	c.lastTriggered.channels[channel] = t
	c.lastTriggered.users[username] = t
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
	IsTimedout    bool

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
