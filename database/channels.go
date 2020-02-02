package database

type Channels []Channel
type Channel struct {
	Name    string
	Enabled bool
	Paused  bool
}

// TODO: implement
func (c *Client) AddNewChannel() {
	panic("not implemented")
}
