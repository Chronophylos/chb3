package state

import "time"

type Voicemail struct {
	Created time.Time
	Message string
	Channel string
	Creator string
}

func NewVoicemail(channel, creator, message string, created time.Time) *Voicemail {
	return &Voicemail{
		Created: created,
		Message: message,
		Channel: channel,
		Creator: creator,
	}
}

func (v *Voicemail) String() string {
	return v.Created.Format(time.Stamp) + " " + v.Creator + ": " + v.Message
}

type User struct {
	ID          string
	Name        string
	DisplayName string

	IsRegular bool

	Firstseen time.Time
	Lastseen  time.Time

	Timeout time.Time

	LastPatsched time.Time
	PatschStreak int
	PatschCount  int

	Voicemails []*Voicemail
}

// IsTimedout checks if a user is currently timed out.
func (u *User) IsTimedout(now time.Time) bool {
	return u.Timeout.After(now)
}

// SetTimeout times out a user until t.
func (u *User) SetTimeout(t time.Time) {
	u.Timeout = t
}

// HasPatschedLately returns true if lastPatsched is no more then 48 hourse before now.
func (u *User) HasPatschedLately(now time.Time) bool {
	diff := now.Sub(u.LastPatsched)
	return diff.Hours() < 48
}

// HasPatschedToday returns true if lastPatsched is on the same day as now.
// It does this by truncating both lastPatsched and now by a day and then checking for equality.
func (u *User) HasPatschedToday(now time.Time) bool {
	lastPatsched := u.LastPatsched.Truncate(24 * time.Hour)
	now = now.Truncate(24 * time.Hour)

	return lastPatsched.Equal(now)
}

// Patsch sets count as well as streak and lastPatsched if applicable.
// A user must patsch every day but not more than once or their streak will be broken.
func (u *User) Patsch(now time.Time) {
	if u.HasPatschedLately(now) { // check if streak is broken
		if !u.HasPatschedToday(now) { // check if user has patsched today already
			// user has not patsched today -> increase streak
			u.PatschStreak++
		} else {
			// user has patsched today already -> reset streak
			u.PatschStreak = 0
		}
	} else {
		// user forgot to patsch reset their streak
		u.PatschStreak = 0
	}

	u.LastPatsched = now
	u.PatschCount++
}

func (u *User) PopVoicemails() []*Voicemail {
	voicemails := u.Voicemails
	u.Voicemails = []*Voicemail{}
	return voicemails
}

func (u *User) HasVoicemails() bool {
	return len(u.Voicemails) > 0
}

func (u *User) AddVoicemail(voicemail *Voicemail) {
	u.Voicemails = append(u.Voicemails, voicemail)
}
