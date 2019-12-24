package actions

import (
	"regexp"
)

type Action interface {
	Run(*Event)
	GetName() string
}

type ActionMap map[*regexp.Regexp]Action

var actionMap = ActionMap{
	regexp.MustCompile(`(?i)^~version`): version{},
}

func GetAll() ActionMap { return actionMap }
