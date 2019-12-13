// Package state binds a mongo database and exports functions for interacting
// with said database.
package state

import (
	"context"
	"fmt"
	"time"

	"github.com/gempir/go-twitch-irc/v2"

	"github.com/rs/zerolog/log"
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
		return &Client{}, fmt.Errorf("ping timed out: %v", err)
	}

	upsert := true

	return &Client{mongo: client, upsert: &upsert}, nil
}

// BumpUser makes sure the twitch user u exists in the database and creates it
// if needed. Either way it sets lastseen to t.
func (c *Client) BumpUser(u twitch.User, t time.Time) (*User, error) {
	var user *User

	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "id", Value: u.ID}},
			bson.D{{Key: "name", Value: u.Name}},
		}},
	}
	if err := col.FindOne(ctx, filter).Err(); err != nil {
		log.Info().
			Str("id", u.ID).
			Str("username", u.Name).
			Msg("Inserting new User to database")
		// insert new user
		user = &User{
			ID:          u.ID,
			Name:        u.Name,
			DisplayName: u.DisplayName,
			Firstseen:   t,
			Lastseen:    t,
			Voicemails:  []*Voicemail{},
		}
		_, err := col.InsertOne(ctx, user)
		return user, err
	}
	// update
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "lastseen", Value: t},
			{Key: "id", Value: u.ID},
			{Key: "name", Value: u.Name},
			{Key: "displayname", Value: u.DisplayName},
		}},
	}
	err := col.FindOneAndUpdate(ctx, filter, update).Decode(&user)
	if err != nil {
		return user, err
	}

	return user, nil
}

// GetUserByID gets the user with id id.
func (c *Client) GetUserByID(id string) (User, error) {
	var user User

	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{{Key: "id", Value: id}}
	err := col.FindOne(ctx, filter).Decode(&user)

	return user, err
}

// GetUserByName gets the user with name name.
func (c *Client) GetUserByName(name string) (User, error) {
	var user User

	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{{Key: "name", Value: name}}
	err := col.FindOne(ctx, filter).Decode(&user)

	return user, err
}

// UpdateUser updates the user user in the mongo database.
func (c *Client) UpdateUser(user User) error {
	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{{Key: "id", Value: user.ID}}
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

	filter := bson.D{{Key: "name", Value: channelName}}
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

	filter := bson.D{{Key: "name", Value: channelName}}
	err := col.FindOne(ctx, filter).Decode(&channel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return channel.Sleeping, nil
}

// IsLurking return true if the bot is just lurking in a channel.
func (c *Client) IsLurking(channelName string) (bool, error) {
	var channel Channel

	col := c.mongo.Database("chb3").Collection("channels")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{{Key: "name", Value: channelName}}
	err := col.FindOne(ctx, filter).Decode(&channel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return channel.Lurking, nil
}

// SetLurking sets lurking.
func (c *Client) SetLurking(channelName string, lurking bool) error {
	col := c.mongo.Database("chb3").Collection("channels")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{{Key: "name", Value: channelName}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "name", Value: channelName},
			{Key: "lurking", Value: lurking},
		}},
	}
	opts := options.FindOneAndUpdate()
	opts.Upsert = c.upsert
	col.FindOneAndUpdate(ctx, filter, update, opts)

	return nil
}

// GetJoinedChannels returns all currentyl joined channels.
func (c *Client) GetJoinedChannels() ([]string, error) {
	channels := []string{}

	col := c.mongo.Database("chb3").Collection("channels")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{{Key: "joined", Value: true}}
	cur, err := col.Find(ctx, filter)
	if err != nil {
		return channels, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var channel Channel

		if err := cur.Decode(&channel); err != nil {
			return channels, err
		}

		log.Debug().
			Interface("channel", channel).
			Send()

		channels = append(channels, channel.Name)
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
	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	voicemail := NewVoicemail(channel, creator, message, created)

	log.Debug().
		Str("username", username).
		Interface("voicemail", voicemail).
		Msg("Adding Voicemail")

	filter := bson.D{{Key: "name", Value: username}}
	update := bson.D{
		{Key: "$push", Value: bson.D{
			{Key: "voicemails", Value: voicemail},
		}},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true)
	err := col.FindOneAndUpdate(ctx, filter, update, opts).Err()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		return err
	}

	return nil
}

// CheckForVoicemails pops all voicemails a user has
func (c *Client) CheckForVoicemails(name string) ([]*Voicemail, error) {
	var voicemails []*Voicemail
	var user User

	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{{Key: "name", Value: name}}
	update := bson.D{
		{Key: "$pull", Value: bson.D{
			{Key: "voicemails", Value: bson.D{}},
		}},
	}
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.Before)
	if err := col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return voicemails, nil
		}
		return voicemails, err
	}

	return user.PopVoicemails(), nil
}

func (c *Client) Patsch(id string, now time.Time) error {
	col := c.mongo.Database("chb3").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := c.GetUserByID(id)
	if err != nil {
		return err
	}

	result := user.Patsch(now)

	filter := bson.D{{Key: "id", Value: id}}
	update := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "patschcount", Value: 1}}},
		{Key: "$set", Value: bson.D{
			{Key: "patschstreak", Value: user.PatschStreak},
			{Key: "lastpatsched", Value: now},
		}},
	}
	if err := col.FindOneAndUpdate(ctx, filter, update).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return result
		}
		return err
	}

	return result
}
