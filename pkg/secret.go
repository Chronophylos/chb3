package pkg

import (
	"encoding/json"
	"io/ioutil"

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
	ClientID string `json:"clientid"`
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

	log.Info().
		Str("filename", filename).
		Msg("Secrets Loaded")

	return secret
}
