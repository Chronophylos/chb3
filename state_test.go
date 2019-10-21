package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPatscher(t *testing.T) {
	assert := assert.New(t)

	date := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	p := NewPatscher()

	assert.Equal(date, p.LastPatsched, "last patsched")
	assert.Equal(0, p.Count, "count should be 0")
	assert.Equal(0, p.Streak, "streak should be 0")

}

func TestPatscherPatsch(t *testing.T) {
	assert := assert.New(t)

	day1 := time.Date(2010, time.November, 10, 23, 0, 0, 0, time.UTC)
	day2 := time.Date(2010, time.November, 11, 23, 0, 0, 0, time.UTC)

	p := NewPatscher()

	assert.True(p.LastPatsched.Before(day1))

	p.Patsch(day1)

	assert.Equal(day1, p.LastPatsched, "last patsched")
	assert.Equal(1, p.Count, "count should be 1")
	assert.Equal(0, p.Streak, "streak should be 0")

	p.Patsch(day2)

	assert.Equal(day2, p.LastPatsched, "last patsched")
	assert.Equal(2, p.Count, "count should be 4")
	assert.Equal(1, p.Streak, "streak should be 1")
}

func TestPatscherHasPatsched(t *testing.T) {
	assert := assert.New(t)

	now := time.Date(2010, time.November, 10, 23, 0, 0, 0, time.UTC)
	yesterday := time.Date(2010, time.November, 9, 23, 0, 0, 0, time.UTC)
	lastWeek := time.Date(2010, time.November, 3, 23, 0, 0, 0, time.UTC)

	p := NewPatscher()

	assert.False(p.HasPatschedLately(now))
	assert.False(p.HasPatschedToday(now))

	p.Patsch(lastWeek)

	assert.False(p.HasPatschedLately(now))
	assert.False(p.HasPatschedToday(now))

	p.Patsch(yesterday)

	assert.True(p.HasPatschedLately(now))
	assert.False(p.HasPatschedToday(now))
}
