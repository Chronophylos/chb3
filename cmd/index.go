package cmd

import (
	"regexp"
)

type Action interface {
	Run(*Event)
}

type ActionMap map[*regexp.Regexp]Action

var actionMap = ActionMap{
	regexp.MustCompile(`(?i)^~version`): Version{},
}
