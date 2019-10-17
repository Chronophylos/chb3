package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Secret struct {
	Twitch TwitchSecret `json:"twitch"`
	Imgur  ImgurSecret  `json:"imgur"`
}

type TwitchSecret struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

type ImgurSecret struct {
	ClientID string `json:"id"`
}

func NewSecret(filename string) Secret {
	var secret Secret

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal().
			Str("filename", filename).
			Err(err).
			Msg("Error reading secret file")
	}

	err = json.Unmarshal(bytes, &secret)
	if err != nil {
		log.Fatal().
			Str("filename", filename).
			Err(err).
			Msg("Error unmarshalling secret")
	}

	if log.Logger.GetLevel() > zerolog.DebugLevel && *showSecrets {
		log.Debug().
			Str("filename", filename).
			Interface("secrets", secret).
			Msg("Secrets Loaded")
	} else {
		log.Info().
			Str("filename", filename).
			Msg("Secrets Loaded")
	}

	return secret
}

func CencorSecrets(secret string) string {
	if *showSecrets {
		return secret
	}
	return "[REDACTED]"
}
