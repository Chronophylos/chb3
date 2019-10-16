package pkg

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetGlobalLogger(debug bool) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if debug {
		// Pretty logging
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Add file and line number to log
	log.Logger = log.With().Caller().Logger()
}
