package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rs/zerolog/log"
)

type State struct {
	channels map[string]*channelState
	filename string
}

type channelState struct {
	Sleeping bool `json:"sleeping"`
}

func NewState(filename string) *State {
	var channels map[string]*channelState

	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal().
				Err(err).
				Msg("Could not read state file")
		}

		err = json.Unmarshal(bytes, &channels)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("filename", filename).
				Msg("Could not unmarshal state file")
		}

		log.Info().
			Str("filename", filename).
			Msg("Loaded State")

		return &State{
			channels: channels,
			filename: filename,
		}
	}

	log.Warn().
		Str("filename", filename).
		Msg("State file does not exist")

	channels = make(map[string]*channelState)

	return &State{
		channels: channels,
		filename: filename,
	}

}

func (s *State) save() {
	bytes, err := json.Marshal(s.channels)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Could not marshal state to json")
	}

	err = ioutil.WriteFile(s.filename, bytes, 0644)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("filename", s.filename).
			Msg("Could not write state to file")
	}

	log.Debug().
		Msg("Saved state to disk")
}
func (s *State) IsSleeping(channel string) bool {
	if cState, ok := s.channels[channel]; ok {
		return cState.Sleeping
	}
	return false
}

func (s *State) SetSleeping(channel string, sleeping bool) {
	if cState, ok := s.channels[channel]; ok {
		cState.Sleeping = sleeping
	} else {
		s.channels[channel] = &channelState{Sleeping: sleeping}
	}

	log.Debug().
		Str("channel", channel).
		Bool("sleeping", sleeping).
		Msg("Changed sleeping state")

	s.save()
}

func (s *State) GetChannels() []string {
	channels := make([]string, 0, len(s.channels))

	for k := range s.channels {
		channels = append(channels, k)
	}

	return channels
}

func (s *State) AddChannel(channel string) error {
	if s.HasChannel(channel) {
		return fmt.Errorf("Channel %s already exists", channel)
	}

	s.channels[channel] = &channelState{}
	s.save()

	return nil
}

func (s *State) RemoveChannel(channel string) error {
	if !s.HasChannel(channel) {
		return fmt.Errorf("Channel %s doesn't exists", channel)
	}

	delete(s.channels, channel)
	s.save()

	return nil
}

func (s *State) HasChannel(channel string) bool {
	_, present := s.channels[channel]
	return present
}
