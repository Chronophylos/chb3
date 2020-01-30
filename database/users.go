package database

import (
	"time"

	"github.com/gempir/go-twitch-irc/v2"
)

type Users []User
type User struct {
	ID          int
	Name        string
	DisplayName string `db:"display_name"`

	FirstSeen time.Time `db:"first_seen"`
	LastSeen  time.Time `db:"last_seen"`

	Timeout time.Time
	Banned  bool

	LastPatsched time.Time
	PatschStream int
	PatschCount  int

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
	SET display_name = EXCLUDED.display_name,
	SET last_seen = EXCLUDED.time;
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

func (c *Client) UpdateUserBan(userID int, banned bool) error {
	_, err := c.db.Exec(`
UPDATE users
SET banned=$1
WHERE id = $2
	`, banned, userID)

	return err
}
