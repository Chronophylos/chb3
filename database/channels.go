package database

type Channels []Channel
type Channel struct {
	ID       int    // user and channel id
	Name     string // the channel name
	Enabled  bool   // should the bot join this channel
	Paused   bool   // should the bot react to normal messages?
	ReadOnly bool   // currently unused
}

// PutChannel puts a new channel into the database
func (c *Client) PutChannel(channel *Channel) error {
	_, err := c.db.NamedExec(`
	INSERT INTO channels (id, name, enabled, paused, readonly)
	VALUES (:id, :name, :enabled, :paused, :readonly)
	`, channel)
	return err
}
