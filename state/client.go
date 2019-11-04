package state

import (
	"context"
	"fmt"
	"time"

	"github.com/gempir/go-twitch-irc/v2"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type StateClient struct {
	client *mongo.Client
	ctx    context.Context
}

func NewClient(uri string) (error, *StateClient) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err, &StateClient{}
	}

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, &StateClient{}
	}

	return nil, &StateClient{client: client, ctx: ctx}
}

func (sc *StateClient) BumpUser(u twitch.User, t time.Time) error {
	c := sc.client.Database("chb3").Collection("users")

	filter := bson.M{"id", u.ID}
	if err := c.FindOne(sc.ctx, filter).Err(); err != nil {
		// insert new user
		user := User{
			id:          u.ID,
			name:        u.Name,
			displayName: u.displayName,
			firstseen:   t,
			lastseen:    t,
		}
		_, err := c.InsertOne(sc.ctx, user)
		return err
	}
	// update last seen
	update := bson.D{
		{"$set", bson.D{
			{"lastseen", t},
		}},
	}
	result, err := c.UpdateOne(sc.ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount != 1 {
		return fmt.Errorf("not 1 document matched but %d", result.MatchedCount)
	}

	return nil
}

func (sc *StateClient) GetUserByID(id string) (User, error) {
	c := sc.client.Database("chb3").Collection("users")
	var user User

	filter := bson.M{"id": id}
	err := c.FindOne(sc.ctx, filter).Decode(&user)

	return user, err
}

func (sc *StateClient) UpdateUser(user User) error {
	c := sc.client.Database("chb3").Collection("users")

	filter := bson.M{"id": user.id}
	result, err := c.ReplaceOne(sc.ctx, filter, &user)
	if err != nil {
		return err
	}

	if result.MatchedCount != 1 {
		return fmt.Errorf("not 1 document matched but %d", result.MatchedCount)
	}

	return nil
}

func (sc *StateClient) SetSleeping(channelName string, sleeping bool) error {
	c := sc.client.Database("chb3").Collection("channels")

	filter := bson.M{"name": channelName}
	update := bson.D{
		{"$set", bson.D{
			{"name": channelName},
			{"sleeping", sleeping},
		}},
	}
	options := mongo.FindOneAndUpdateOptions{
		Upsert: true,
	}
	result, err := c.FindOneAndUpdate(sc.ctx, filter, update, options)
	if err != nil {
		return err
	}

	if result.MatchedCount != 1 {
		return fmt.Errorf("not 1 document matched but %d", result.MatchedCount)
	}

	return nil
}

func (sc *StateClient) GetJoinedChannels() ([]string, error) {
	c := sc.client.Database("chb3").Collection("channels")
	channels := []string{}

	filter := bson.M{}
	cur, err := c.Find(sc.ctx, filter)
	if err != nil {
		return channels, err
	}
	defer cur.Close(sc.ctx)

	for cur.Next(sc.ctx) {
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

func (sc *StateClient) JoinChannel(channelName string, joined bool) error {
	c := sc.client.Database("chb3").Collection("channels")

	filter := bson.M{"name": channelName}
	update := bson.D{
		{"$set", bson.D{
			{"name": channelName},
			{"joined", joined},
		}},
	}
	options := mongo.FindOneAndUpdateOptions{
		Upsert: true,
	}
	result, err := c.FindOneAndUpdate(sc.ctx, filter, update, options)
	if err != nil {
		return err
	}

	if result.MatchedCount != 1 {
		return fmt.Errorf("not 1 document matched but %d", result.MatchedCount)
	}

	return nil
}
