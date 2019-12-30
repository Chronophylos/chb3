package actions

import (
	"errors"
	"regexp"
	"time"
)

type Options struct {
	Name      string
	Re        *regexp.Regexp
	Sleepless bool
	Perm      Permission

	UserCooldown    time.Duration
	ChannelCooldown time.Duration
	GlobalCooldown  time.Duration
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
	newVoicemailAction(),
	newPatscheckAction(),
	newPatschAction(),
	newVanishReplyAction(),
	newCircumflexAction(),
	newPingAction(),
	newRateAction(),
	newWeatherAction1(),
	newWeatherAction2(),
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

	if opt.Perm < Everyone || opt.Perm > Owner {
		return errors.New("field Perm is out of bounds")
	}

	return nil
}
