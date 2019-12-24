package actions

import (
	"regexp"
)

type ActionOptions struct {
	Name      string
	Sleepless bool
}

type Action interface {
	Run(*Event) error
	GetOptions() *ActionOptions
}

type ActionMap map[*regexp.Regexp]Action

var actionMap = ActionMap{
	regexp.MustCompile(`(?i)^~version`):                            version{},
	regexp.MustCompile(`(?i)^~(shut up|go sleep|sleep|sei ruhig)`): stateSleep{},
	regexp.MustCompile(`(?i)^~(wake up|wach auf)`):                 stateWake{},
}

func GetAll() ActionMap { return actionMap }
