package database

import "time"

type Voicemails []Voicemail
type Voicemail struct {
	ID       int
	Creator  *User
	Created  time.Time
	Recipent string
	Message  string
	Replayed bool
}

func (v *Voicemail) String() string {
	return v.Created.Format(time.Stamp) + " " + v.Creator.DisplayName + ": " + v.Message
}

// PopVoicemails returns all voicmails for name and removes them from the database
func (c *Client) PopVoicemails(name string) (Voicemails, error) {
	var voicemails Voicemails

	tx, err := c.DB.Beginx()
	if err != nil {
		return voicemails, err
	}

	if err = tx.Select("SELECT * FROM voicemails WHERE recipent=$1", name); err != nil {
		return voicemails, err
	}

	if _, err = tx.Exec("UPDATE voicemails SET replayed=true WHERE recipent=$1", name); err != nil {
		return voicemails, err
	}

	err = tx.Commit()

	return voicemails, err
}

// DeleteVoicemail deletes a single voicemails
func (c *Client) DeleteVoicemail(id int) error {
	_, err := c.DB.Exec("DELETE FROM voicemails WHERE id=$1", id)

	return err
}

// PutVoicemail puts a voicemail into the database
func (c *Client) PutVoicemail(voicemail *Voicemail) error {
	_, err := c.DB.Exec(`
	INSERT INTO voicemails (creator, created, recipent, message)
	VALUES (:creator, :created, :recipent, :message)
	`, voicemail)

	return err
}
