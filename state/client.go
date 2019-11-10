// Package state binds a mongo database and exports functions for interacting
// with said database.
package state

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gempir/go-twitch-irc/v2"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client provides functions to interact with the databse.
type Client struct {
	mongo  *mongo.Client
	upsert *bool
}

// NewClient connects to the mongo database located at uri and pings it.
func NewClient(uri string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return &Client{}, err
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return &Client{}, errors.New("ping timed out")
	}

	upsert := true

	return &Client{mongo: client, upsert: &upsert}, nil
}

// BumpUser makes sure the twitch user u exists in the database and creates it
// if needed. Either way it sets lastseen to t.
func (c *Client) BumpUser(u twitch.User, t time.Time) error {
	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"id": u.ID}
	if err := col.FindOne(ctx, filter).Err(); err != nil {
		// insert new user
		user := User{
			ID:          u.ID,
			Name:        u.Name,
			DisplayName: u.DisplayName,
			firstseen:   t,
			lastseen:    t,
		}
		_, err := col.InsertOne(ctx, user)
		return err
	}
	// update last seen
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "lastseen", Value: t},
		}},
	}
	result, err := col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount != 1 {
		return fmt.Errorf("not 1 document matched but %d", result.MatchedCount)
	}

	return nil
}

// GetUserByID gets the user with id id.
func (c *Client) GetUserByID(id string) (User, error) {
	var user User

	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"id": id}
	err := col.FindOne(ctx, filter).Decode(&user)

	return user, err
}

// GetUserByName gets the user with name name.
func (c *Client) GetUserByName(name string) (User, error) {
	var user User

	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"name": name}
	err := col.FindOne(ctx, filter).Decode(&user)

	return user, err
}

// UpdateUser updates the user user in the mongo database.
func (c *Client) UpdateUser(user User) error {
	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"id": user.ID}
	result, err := col.ReplaceOne(ctx, filter, &user)
	if err != nil {
		return err
	}

	if result.MatchedCount != 1 {
		return fmt.Errorf("not 1 document matched but %d", result.MatchedCount)
	}

	return nil
}

// IsTimedout checks if a user is timed out.
func (c *Client) IsTimedout(id string, now time.Time) (bool, error) {
	user, err := c.GetUserByID(id)
	if err != nil {
		return false, err
	}
	return user.IsTimedout(now), nil
}

// SetSleeping sets sleeping.
func (c *Client) SetSleeping(channelName string, sleeping bool) error {
	col := c.mongo.Database("chb3").Collection("channels")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"name": channelName}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "name", Value: channelName},
			{Key: "sleeping", Value: sleeping},
		}},
	}
	opts := options.FindOneAndUpdate()
	opts.Upsert = c.upsert
	col.FindOneAndUpdate(ctx, filter, update, opts)

	return nil
}

// IsSleeping checks if a channels is sleeping.
func (c *Client) IsSleeping(channelName string) (bool, error) {
	var channel Channel

	col := c.mongo.Database("chb3").Collection("channels")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"name": channelName}
	err := col.FindOne(ctx, filter).Decode(&channel)
	if err != nil {
		return false, err
	}

	return channel.Sleeping, nil
}

// GetJoinedChannels returns all currentyl joined channels.
func (c *Client) GetJoinedChannels() ([]string, error) {
	channels := []string{}

	col := c.mongo.Database("chb3").Collection("channels")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{}
	cur, err := col.Find(ctx, filter)
	if err != nil {
		return channels, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		channel := Channel{}
		if err := cur.Decode(channel); err != nil {
			return channels, err
		}

		if channel.Joined {
			channels = append(channels, channel.Name)
		}
	}

	if err := cur.Err(); err != nil {
		return channels, err
	}

	return channels, nil
}

// JoinChannel sets joined.
func (c *Client) JoinChannel(channelName string, joined bool) error {
	col := c.mongo.Database("chb3").Collection("channels")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"name": channelName}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "name", Value: channelName},
			{Key: "joined", Value: joined},
		}},
	}
	opts := options.FindOneAndUpdate()
	opts.Upsert = c.upsert
	col.FindOneAndUpdate(ctx, filter, update, opts)

	return nil
}

// IsChannelJoined check if a channel is joined.
func (c *Client) IsChannelJoined(channelName string) (bool, error) {
	var channel Channel

	col := c.mongo.Database("chb3").Collection("channels")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"name": channelName}
	err := col.FindOne(ctx, filter).Decode(&channel)
	if err != nil {
		return false, err
	}

	return channel.Joined, nil
}

// AddVoicemail adds a voicemail to a user.
func (c *Client) AddVoicemail(username, channel, creator, message string, created time.Time) error {
	var user User

	user, err := c.GetUserByName(username)
	if err != nil {
		return err
	}

	voicemail := NewVoicemail(channel, creator, message, created)

	user.AddVoicemail(voicemail)

	if err = c.UpdateUser(user); err != nil {
		return err
	}

	return nil
}
