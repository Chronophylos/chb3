package database

type Channels []Channel
type Channel struct {
	ID       int    // user and channel id
	Name     string // the channel name
	Enabled  bool   // should the bot join this channel
	Paused   bool   // should the bot react to normal messages?
	ReadOnly bool   // currently unused
}

func (c Channels) Names() []string {
	r := []string{}
	for _, channel := range c {
		r = append(r, channel.Name)
	}
	return r
}

// PutChannel puts a new channel into the database
func (c *Client) PutChannel(channel *Channel) error {
	_, err := c.DB.NamedExec(`
	INSERT INTO channels (id, name, enabled, paused, readonly)
	VALUES (:id, :name, :enabled, :paused, :readonly)
	`, channel)
	return err
}

func (c *Client) GetChannel(name string) (*Channel, error) {
	var channel Channel

	err := c.DB.Get(&channel, "SELECT * FROM channels WHERE name=$1", name)

	return &channel, err
}

func (c *Client) GetChannels() (Channels, error) {
	var channels Channels

	err := c.DB.Select(&channels, "SELECT * FROM channels WHERE enabled=true")

	return channels, err
}

func (c *Client) PauseChannel(name string) error {
	_, err := c.DB.Exec(`
	UPDATE channels
	SET paused=true
	WHERE name=$1
	`, name)
	return err
}

func (c *Client) ResumeChannel(name string) error {
	_, err := c.DB.Exec(`
	UPDATE channels
	SET paused=false
	WHERE name=$1
	`, name)
	return err
}
