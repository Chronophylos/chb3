package main

import (
	"regexp"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Match [][]string

// TriggerFunction is a functions called when a command is triggered that provides all required information.
// The function should return true if the command was successfull and false if not and the next command should be tried
type TriggerFunction func(cmdState *CommandState, log zerolog.Logger, match Match) bool

// Command holds a function that can be triggered with Command#Trigger and a regex
type Command struct {
	Name        string
	re          *regexp.Regexp
	trigger     TriggerFunction
	IgnoreSleep bool
}

// NewCommand creates a new command from a regex string and a TriggerFunction returning the new command
func NewCommand(name, re string, trigger TriggerFunction) *Command {
	return &Command{
		Name:    name,
		re:      regexp.MustCompile(re),
		trigger: trigger,
	}
}

func NewCommandEx(name, re string, trigger TriggerFunction, ignoreSleep bool) *Command {
	cmd := NewCommand(name, re, trigger)
	cmd.IgnoreSleep = ignoreSleep
	return cmd
}

// Trigger is used to trigger a commmand
func (c *Command) Trigger(cmdState *CommandState) bool {
	if c.IgnoreSleep || !cmdState.IsSleeping {
		if !cmdState.IsTimedOut || cmdState.IsOwner {
			match := c.re.FindAllStringSubmatch(cmdState.Message, -1)
			if match == nil {
				return false
			}

			log := c.getLogger(cmdState)

			log.Debug().
				Interface("match", match).
				Str("command", c.Name).
				Msg("Found matching command")

			return c.trigger(cmdState, log, match)
		}
	}

	return false
}

// CommandState holds information that represent the state of a command. e.g. wheather the invoker is mod what channel and so on
type CommandState struct {
	IsSleeping    bool
	IsMod         bool
	IsBroadcaster bool
	IsOwner       bool
	IsBot         bool
	IsTimedOut    bool

	Channel string
	Message string
	Time    time.Time
	User    *twitch.User

	Raw *twitch.PrivateMessage
}

// CommandRegistry is used to register commands and trigger all registered commands
type CommandRegistry struct {
	commands []*Command
}

// NewCommandRegistry creates and returns a new CommandRegistry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{}
}

// Register registers a command in the registry
func (r *CommandRegistry) Register(command *Command) {
	r.commands = append(r.commands, command)
}

// Trigger checks the regex of all commands in order and triggers a command
// if the regex matches. If that command returns true,
// signaling a successfull execution, it returns otherwise tries the command.
func (r *CommandRegistry) Trigger(cmdState *CommandState) {
	for _, command := range r.commands {
		if command.Trigger(cmdState) {
			break
		}
	}
}

func (c *Command) getLogger(cmdState *CommandState) zerolog.Logger {
	return log.With().
		Str("command", c.Name).
		Str("channel", cmdState.Channel).
		Str("username", cmdState.User.Name).
		Str("msg", cmdState.Message).
		Logger()
}
