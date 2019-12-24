package actions

import (
	"regexp"
)

type Options struct {
	Name      string
	Sleepless bool
}

type Action interface {
	Run(*Event) error
	GetOptions() *Options
}

type ActionMap map[*regexp.Regexp]Action

var actionMap = ActionMap{
	regexp.MustCompile(`(?i)^~version`):                            version{},
	regexp.MustCompile(`(?i)^~(shut up|go sleep|sleep|sei ruhig)`): stateSleep{},
	regexp.MustCompile(`(?i)^~(wake up|wach auf)`):                 stateWake{},
	regexp.MustCompile(`(?i)^~join( (\w+))?`):                      joinChannel{},
	regexp.MustCompile(`(?i)^~leave( (\w+))?`):                     leaveChannel{},
}

func GetAll() ActionMap { return actionMap }
