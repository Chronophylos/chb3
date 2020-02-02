package actions

import (
	"fmt"
	"regexp"
	"strings"
	"time"
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
	var seconds, minutes, hours, days, weeks, months, years float64
	var seconds_r, minutes_r, hours_r, days_r, weeks_r, months_r, years_r int

	seconds = d.Seconds()
	minutes = seconds / 60
	hours = minutes / 60
	days = hours / 24
	weeks = days / 7
	months = weeks / 4
	years = months / 12

	years_r = int(years)
	months_r = int(months) - years_r*12
	weeks_r = int(weeks) - months_r*4
	hours_r = int(hours) - days_r*24
	minutes_r = int(minutes) - hours_r*60
	seconds_r = int(seconds) - minutes_r*60

	formatToBuilder(&strs, "year", years_r)
	formatToBuilder(&strs, "month", months_r)
	formatToBuilder(&strs, "week", weeks_r)
	formatToBuilder(&strs, "day", days_r)
	formatToBuilder(&strs, "hour", hours_r)
	formatToBuilder(&strs, "minute", minutes_r)
	formatToBuilder(&strs, "second", seconds_r)

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
