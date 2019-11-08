package state

import "time"

type Voicemail struct {
	created time.Time
	message string
	channel string
	creator string
}

func NewVoicemail(message, channel, creator string, created time.Time) *Voicemail {
	return &Voicemail{
		created: created,
		message: message,
		channel: channel,
		creator: creator,
	}
}

func (v *Voicemail) String() string {
	return v.created.Format(time.Stamp) + " " + v.creator + ": " + v.message
}

type User struct {
	id          string
	name        string
	displayName string

	isRegular bool

	firstseen time.Time
	lastseen  time.Time

	timeout time.Time

	lastPatsched time.Time
	PatschStreak int
	PatschCount  int

	voicemails []*Voicemail
}

// IsTimedout checks if a user is currently timed out.
func (u *User) IsTimedout(now time.Time) bool {
	return u.timeout.After(now)
}

// Timeout times out a user until t.
func (u *User) Timeout(t time.time) {
	u.timeout = t
}

// HasPatschedLately returns true if lastPatsched is no more then 48 hourse before now.
func (u *User) HasPatschedLately(now time.Time) bool {
	diff := t.Sub(p.lastPatsched)
	return diff.Hours() < 48
}

// HasPatschedToday returns true if lastPatsched is on the same day as now.
// It does this by truncating both lastPatsched and now by a day and then checking for equality.
func (u *User) HasPatschedToday(now time.Time) bool {
	lastPatsched := u.lastPatsched.Truncate(24 * time.Hour)
	now = now.Truncate(24 * time.Hour)

	return lastPatsched.Equal(now)
}

// Patsch sets count as well as streak and lastPatsched if applicable.
// A user must patsch every day but not more than once or their streak will be broken.
func (u *User) Patsch(now time.Time) {
	if u.HasPatschedLately(now) { // check if streak is broken
		if !u.HasPatschedToday(now) { // check if user has patsched today already
			// user has not patsched today -> increase streak
			u.patschStreak++
		} else {
			// user has patsched today already -> reset streak
			u.patschStreak = 0
		}
	} else {
		// user forgot to patsch reset their streak
		u.patschStreak = 0
	}

	u.lastPatsched = now
	u.patschCount++
}

func (u *User) PopVoicemails() []*Voicemail {
	voicemails = s.voicemails
	u.voicemails = []*Voicemail{}
	return voicemails
}

func (u *User) HasVoicemails() bool {
	return len(s.voicemails) > 0
}

func (u *User) AddVoicemail(voicemail *Voicemail) {
	u.voicemails = append(u.voicemails, voicemail)
}
