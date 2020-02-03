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

// PopVoicemails returns all voicmails for name and removes them from the database
func (c *Client) PopVoicemails(name string) (Voicemails, error) {
	var voicemails Voicemails

	tx, err := c.db.Beginx()
	if err != nil {
		return voicemails, err
	}

	if err = tx.Select("SELECT * FROM voicemails WHERE recipent=$1", name); err != nil {
		return voicemails, err
	}

	if _, err = tx.Exec("DELETE FROM voicemails WHERE recipent=$1", name); err != nil {
		return voicemails, err
	}

	err = tx.Commit()

	return voicemails, err
}

// HasVoicemails reports true if voicemails for name exist
func (c *Client) HasVoicemails(name string) (bool, error) {
	var exists bool

	err := c.db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM voicemails WHERE name=$1)", name)

	return exists, err
}

// DeleteVoicemail deletes a single voicemails
func (c *Client) DeleteVoicemail(id int) error {
	_, err := c.db.Exec("DELETE FROM voicemails WHERE id=$1", id)

	return err
}

// CreateVoicemail puts a voicemail into the database
func (c *Client) CreateVoicemail(voicemail *Voicemail) error {
	_, err := c.db.Exec(`
	INSERT INTO voicemails (creator, created, recipent, message)
	VALUES (:creator, :created, :recipent, :message)
	`, voicemail)

	return err
}
