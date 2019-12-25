package actions

import (
	"errors"
	"regexp"
)

type Options struct {
	Name      string
	Re        *regexp.Regexp
	Sleepless bool
}

type Action interface {
	Run(*Event) error
	GetOptions() *Options
}

type Actions []Action

var actions = Actions{
	newVersionAction(),
	newSleepAction(),
	newWakeAction(),
	newJoinAction(),
	newLeaveAction(),
	newLurkAction(),
	newDebugAction(),
}

func GetAll() Actions { return actions }

func Check(a Action) error {
	opt := a.GetOptions()

	if opt.Name == "" {
		return errors.New("required field Name is empty")
	}

	if opt.Re == nil {
		return errors.New("required field Re is nil")
	}

	return nil
}
