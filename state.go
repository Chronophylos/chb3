package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultStatePath = "/var/lib/chb3"
	localStatePath   = "."
	stateFilename    = "state.json"
)

type StateClient struct {
	state  *State
	client *mongo.Client
	ctx    context.Context
}

type State struct {
	Channels        map[string]*channelState `json:"channels"`
	Voicemails      map[string][]*Voicemail  `json:"voicemails"`
	Patscher        map[string]*Patscher     `json:"patscher"`
	LastFishFeeding time.Time                `json:"last-fish-feeding"`
	Timeouts        map[string]*Timeout      `json:"timeouts"`
	Filename        string                   `json:"-"`
}

type channelState struct {
	Sleeping bool `json:"sleeping"`
}

type Timeout struct {
	Until time.Time `json:"until"`
}

func (t *Timeout) IsTimedOut(z time.Time) bool {
	return t.Until.After(z)
}

type Voicemail struct {
	Created time.Time `json:"created"`
	Message string    `json:"message"`
	Channel string    `json:"channel"`
	Creator string    `json:"creator"`
}

func (v *Voicemail) String() string {
	return v.Created.Format(time.Stamp) + " " + v.Creator + ": " + v.Message
}

type Patscher struct {
	LastPatsched time.Time `json:"last-patsched"`
	Count        int       `json:"count"`
	Streak       int       `json:"streak"`
}

// NewPatscher creates a new Patscher.
// LastPatsched is set to golangs launch date something far enough
// in the past so it should not interfere with anything.
func NewPatscher() *Patscher {
	return &Patscher{
		// some day in the past, before chb3 was even imagined
		LastPatsched: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
		Count:        0,
		Streak:       0,
	}
}

// HasPatschedLately checks if LastPatsched is not earlier than 48 hours before t.
func (p *Patscher) HasPatschedLately(t time.Time) bool {
	diff := t.Sub(p.LastPatsched)

	return diff.Hours() < 48
}

// HasPatschedToday checks if LastPatsched is on the same day has t.
func (p *Patscher) HasPatschedToday(t time.Time) bool {
	lastPatsched := p.LastPatsched.Truncate(24 * time.Hour)
	t = t.Truncate(24 * time.Hour)

	return lastPatsched.Equal(t)
}

// Patsch sets LastPatsched to t, increases Count by one.
// Streak gets increased by one if it is not broken. Otherwise it resets it to 0.
// t is now.
func (p *Patscher) Patsch(t time.Time) {
	if p.HasPatschedLately(t) {
		// Streak is not broken
		if !p.HasPatschedToday(t) {
			// Don't increase streak multiple times per day
			p.Streak++
		}
	} else {
		// Streak is broken
		p.Streak = 0
	}
	p.LastPatsched = t
	p.Count++
}

func LoadJSONState() *State {
	state := State{
		Channels:   make(map[string]*channelState),
		Voicemails: make(map[string][]*Voicemail),
		Patscher:   make(map[string]*Patscher),
		Timeouts:   make(map[string]*Timeout),
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

func CreateStateClient(uri string) (error, *StateClient) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return err, &StateClient{}
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, &StateClient{}
	}

	return nil, &StateClient{client: client, ctx: ctx}
}

func (sc *StateClient) DoesDatabaseExist() (error, bool) {
	dbNames, err := sc.client.ListDatabaseNames(sc.ctx, nil)
	if err != nil {
		return err, false
	}

	for _, name := range dbNames {
		if name == "chb3" {
			return true
		}
	}

	return false
}

func (sc *StateClient) Migrate() {
	s := LoadJSONState()
}

// Deprecated: not needed since mongo always saves
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
	username = strings.ToLower(username)
	message = strings.TrimSpace(message)

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

	log.Debug().
		Str("username", username).
		Str("channel", channel).
		Str("creator", creator).
		Str("message", message).
		Time("created", created).
		Msg("Added Voicemail")

	s.save()
}

func (s *State) PopVoicemails(username string) []*Voicemail {
	username = strings.ToLower(username)

	voicemails := s.Voicemails[username]

	s.Voicemails[username] = []*Voicemail{}

	s.save()

	return voicemails
}

func (s *State) HasVoicemail(username string) bool {
	username = strings.ToLower(username)

	voicemails, present := s.Voicemails[username]

	if !present {
		return false
	}

	return len(voicemails) > 0
}

func (s *State) GetPatscher(username string) *Patscher {
	_, present := s.Patscher[username]

	if !present {
		s.Patscher[username] = NewPatscher()
	}

	return s.Patscher[username]
}

func (s *State) Patsch(username string, t time.Time) {
	s.GetPatscher(username).Patsch(t)
	s.save()
}

func (s *State) BreakStreak(username string) {
	s.GetPatscher(username).Streak = 0
	s.save()
}

func (s *State) HasPatschedToday(username string, t time.Time) bool {
	return s.GetPatscher(username).HasPatschedToday(t)
}

func (s *State) HasFishBeenFedToday(t time.Time) bool {
	diff := t.Sub(s.LastFishFeeding)
	days := diff.Hours() / 24

	return days < 1
}

func (s *State) FeedFish(t time.Time) {
	s.LastFishFeeding = t
}

func (s *State) GetTimeout(username string) *Timeout {
	_, present := s.Timeouts[username]

	if !present {
		s.Timeouts[username] = &Timeout{}
	}

	return s.Timeouts[username]
}

func (s *State) Timeout(username string, until time.Time) {
	timeout := s.GetTimeout(username)
	timeout.Until = until
	s.Timeouts[username] = timeout
}

func (s *State) IsTimedOut(username string, t time.Time) bool {
	return s.GetTimeout(username).IsTimedOut(t)
}
