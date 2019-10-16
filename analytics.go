package main

import (
	"bufio"
	"os"

	"github.com/rs/zerolog"
)

func NewAnalytics() (zerolog.Logger, error) {
	file, err := os.OpenFile("analytics.jlog", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return zerolog.Logger{}, err
	}

	writer := bufio.NewWriter(file)

	return zerolog.New(writer).With().Timestamp().Logger(), nil
}
