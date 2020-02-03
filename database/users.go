package database

import (
	"errors"
	"time"

	"database/sql"

	"github.com/gempir/go-twitch-irc/v2"
)

// These errors may occur
var (
	ErrAlreadyPatsched = errors.New("already patsched today")
	ErrForgotToPatsch  = errors.New("fish was not patsched lately")
)

type Users []User
type User struct {
	ID          int
	Name        string
	DisplayName string `db:"display_name"`

	FirstSeen time.Time `db:"first_seen"`
	LastSeen  time.Time `db:"last_seen"`

	Timeout sql.NullTime
	Banned  bool

	LastPatsched sql.NullTime
	PatschStreak int `db:"patsch_streak"`
	PatschCount  int `db:"patsch_count"`

	Birthday sql.NullTime
}

// BumpUser finds a twitch user in the database and updates name, display name
// and last seen. If the user did't exist it creates a new user and
// additionally sets id and first seen.
func (c *Client) BumpUser(u *twitch.User, t time.Time) error {
	_, err := c.db.NamedExec(`
	INSERT INTO users (id, name, display_name, first_seen, last_seen)
	VALUES (:id, :name, :display_name, :time, :time)
	ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
			display_name = EXCLUDED.display_name,
			last_seen = EXCLUDED.time
	`, map[string]interface{}{
		"id":           u.ID,
		"name":         u.Name,
		"display_name": u.DisplayName,
		"time":         t,
	})

	return err
}

// GetUserByID finds a user with id and returns it.
func (c *Client) GetUserByID(id int) (*User, error) {
	var user User
	err := c.db.Get(&user, "SELECT * FROM users WHERE id=$1", id)

	return &user, err
}

func (c *Client) BanUser(userID int) error {
	_, err := c.db.Exec("UPDATE users SET banned=true WHERE id = $1", userID)

	return err
}

func (c *Client) UnbanUser(userID int) error {
	_, err := c.db.Exec("UPDATE users SET banned=false WHERE id = $1", userID)

	return err
}

func (c *Client) TimeoutUser(userID int, until time.Time) error {
	_, err := c.db.Exec("update users set timeout=$1 where id = $2", until, userID)

	return err
}

func (c *Client) UserPatsch(u *User, now time.Time) (error, error) {
	var pErr error
	if u.HasPatschedLately(now) { // check if streak is broken
		if !u.HasPatschedToday(now) { // check if user has patsched today already
			// user has not patsched today -> increase streak
			u.PatschStreak++
		} else {
			// user has patsched today already -> reset streak
			u.PatschStreak = 0
			pErr = ErrAlreadyPatsched
		}
	} else {
		// user forgot to patsch -> reset their streak
		u.PatschStreak = 0
		pErr = ErrForgotToPatsch
	}

	u.PatschCount++

	_, dErr := c.db.NamedExec(`
	UPDATE users
	SET patsch_streak=:patsch_streak,
		patsch_count=:patsch_count,
		last_patsched=:last_patsched
	WHERE id=:id
	`, u)

	return pErr, dErr
}

// IsTimedout reports true if a user is currently timed out.
func (u *User) IsTimedout(now time.Time) bool {
	if u.Timeout.Valid {
		return u.Timeout.Time.After(now)
	}

	return false
}

// HasPatschedLately reports true if a user has patsched in the last 48 hours.
func (u *User) HasPatschedLately(now time.Time) bool {
	if u.LastPatsched.Valid {
		diff := now.Sub(u.LastPatsched.Time)
		return diff.Hours() < 48
	}

	return false
}

// HasPatschedToday reports true if a user has patsched on the same day as now.
func (u *User) HasPatschedToday(now time.Time) bool {
	// only compare days by truncating the time
	if u.LastPatsched.Valid {
		lastPatsched := u.LastPatsched.Time.Truncate(24 * time.Hour)
		now = now.Truncate(24 * time.Hour)

		return lastPatsched.Equal(now)
	}

	return false
}
