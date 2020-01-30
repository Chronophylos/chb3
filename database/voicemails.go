package database

import "time"

type Voicemails []Voicemail
type Voicemail struct {
	ID       int
	Creator  *User
	Created  time.Time
	Recipent string
	Message  string
}

func (c *Client) PopVoicemails(name string) (Voicemails, error) {
	var voicemails Voicemails

	tx, err := c.db.Begin()
	if err != nil {
		return voicemails, err
	}

	if err = tx.Select("SELECT * FROM voicemails WHERE recipent=$1", name); err != nil {
		return voicemails, err
	}

	if err = tx.Exec("DELETE FROM voicemails WHERE recipent=$1", name); err != nil {
		return voicemails, err
	}

	err = tx.Commit()

	return voicemails, nil
}

func (c *Client) HasVoicemails(name string) (bool, error) {
	var exists bool
	err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM voicemails WHERE name=$1)", name)
	return exists, err
}

func (c *Client) DeleteVoicemail(id int) error {
	_, err := c.db.Exec("DELTE FROM voicemails WHERE id=$1", id)

	return err
}
