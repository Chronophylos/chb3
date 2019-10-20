package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	defaultStatePath = "/var/lib/chb3"
	localStatePath   = "."
	stateFilename    = "state.json"
)

type State struct {
	Channels   map[string]*channelState `json:"channels"`
	Voicemails map[string][]*Voicemail  `json:"voicemails"`
	Filename   string                   `json:"-"`
}

type channelState struct {
	Sleeping bool `json:"sleeping"`
}

type Voicemail struct {
	Created time.Time `json:"created"`
	Message string    `json:"message"`
	Channel string    `json:"channel"`
	Creator string    `json:"creator"`
}

func (v *Voicemail) String() string {
	return v.Created.Format(time.StampMilli) + " #" + v.Channel + " " + v.Creator + ": " + v.Message
}

func LoadState() *State {
	state := State{
		Channels:   make(map[string]*channelState),
		Voicemails: make(map[string][]*Voicemail),
	}

	filename := localStatePath

	// check if user is root
	if os.Geteuid() == 0 {
		filename = defaultStatePath

		// make sure the path exists
		err := os.MkdirAll(defaultStatePath, 0644)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("path", defaultStatePath).
				Msg("Error creating path to state file")
		}
	}

	// add filename to path
	filename = filename + "/" + stateFilename
	state.Filename = filename

	// Check if file exists
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		// File seems to exists
		// Read file
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("filename", filename).
				Msg("Could not read state file")
		}

		// Unmarshal file
		err = json.Unmarshal(bytes, &state)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("filename", filename).
				Msg("Could not unmarshal state file")
		}

		log.Info().
			Str("filename", filename).
			Msg("Loaded State")
	} else {
		log.Warn().
			Str("filename", filename).
			Msg("State file does not exist")
	}

	return &state
}

func (s *State) save() {
	bytes, err := json.Marshal(s)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Could not marshal state to json")
	}

	err = ioutil.WriteFile(s.Filename, bytes, 0644)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("filename", s.Filename).
			Msg("Could not write state to file")
	}

	log.Debug().
		Msg("Saved state to disk")
}
func (s *State) IsSleeping(channel string) bool {
	if cState, ok := s.Channels[channel]; ok {
		return cState.Sleeping
	}
	return false
}

func (s *State) SetSleeping(channel string, sleeping bool) {
	if cState, ok := s.Channels[channel]; ok {
		cState.Sleeping = sleeping
	} else {
		s.Channels[channel] = &channelState{Sleeping: sleeping}
	}

	log.Debug().
		Str("channel", channel).
		Bool("sleeping", sleeping).
		Msg("Changed sleeping state")

	s.save()
}

func (s *State) GetChannels() []string {
	channels := make([]string, 0, len(s.Channels))

	for k := range s.Channels {
		channels = append(channels, k)
	}

	return channels
}

func (s *State) AddChannel(channel string) error {
	if s.HasChannel(channel) {
		return fmt.Errorf("Channel %s already exists", channel)
	}

	s.Channels[channel] = &channelState{}
	s.save()

	return nil
}

func (s *State) RemoveChannel(channel string) error {
	if !s.HasChannel(channel) {
		return fmt.Errorf("Channel %s doesn't exists", channel)
	}

	delete(s.Channels, channel)
	s.save()

	return nil
}

func (s *State) HasChannel(channel string) bool {
	_, present := s.Channels[channel]
	return present
}

func (s *State) AddVoicemail(username, channel, creator, message string, created time.Time) {
	voicemail := &Voicemail{
		Created: created,
		Message: message,
		Channel: channel,
		Creator: creator,
	}
	voicemails, present := s.Voicemails[username]
	if !present {
		s.Voicemails[username] = []*Voicemail{voicemail}
	} else {
		s.Voicemails[username] = append(voicemails, voicemail)
	}

	s.save()
}

func (s *State) PopVoicemails(username string) []*Voicemail {
	voicemails := s.Voicemails[username]

	s.Voicemails[username] = []*Voicemail{}

	s.save()

	return voicemails
}

func (s *State) HasVoicemail(username string) bool {
	voicemails, present := s.Voicemails[username]

	if !present {
		return false
	}

	return len(voicemails) > 0
}
