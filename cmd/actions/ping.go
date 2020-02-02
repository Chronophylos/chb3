package actions

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chronophylos/chb3/util"
)

type pingAction struct {
	options *Options
	created time.Time
}

func newPingAction() *pingAction {
	return &pingAction{
		options: &Options{
			Name: "ping",
			Re:   regexp.MustCompile(`(?i)^~ping`),
		},
		created: time.Now(),
	}
}

func (a pingAction) GetOptions() *Options {
	return a.options
}

func (a pingAction) Run(e *Event) error {
	e.Say(fmt.Sprintf("I've been running for %s. It took %dms to receive your message.",
		formatDuration(e.Msg.Time.Sub(a.created)),
		time.Since(e.Msg.Time).Milliseconds(),
	))

	return nil
}

func formatDuration(d time.Duration) string {
	var strs strings.Builder
	var seconds, minutes, hours, days, weeks, months, years int
	var temp float64

	temp, seconds = util.Divmod(d.Seconds(), 60)
	temp, minutes = util.Divmod(temp, 60)
	temp, hours = util.Divmod(temp, 24)
	temp, days = util.Divmod(temp, 7)
	temp, weeks = util.Divmod(temp, 4)
	temp, months = util.Divmod(temp, 12)

	years = int(temp)

	formatToBuilder(&strs, "year", years)
	formatToBuilder(&strs, "month", months)
	formatToBuilder(&strs, "week", weeks)
	formatToBuilder(&strs, "day", days)
	formatToBuilder(&strs, "hour", hours)
	formatToBuilder(&strs, "minute", minutes)
	formatToBuilder(&strs, "second", seconds)

	return strings.TrimSpace(strs.String())
}

func formatToBuilder(b *strings.Builder, s string, c int) {
	if c > 1 {
		b.WriteString(pluralize(s, c))
	}
}

func pluralize(s string, c int) string {
	if c >= 2 {
		s = s + "s"
	}
	return fmt.Sprintf("%d %s ", c, s)
}
