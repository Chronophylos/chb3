package twotsch

import (
	"github.com/gempir/go-twitch-irc/v2"
)

type Client struct {
	client *twitch.Client
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Say(channel, message string) {

}

func (c *Client) Raw() *twitch.Client {
	return c.client
}
