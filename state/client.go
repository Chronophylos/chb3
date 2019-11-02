package state

import (
	"context"
	"fmt"
	"time"

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
	result, err := c.UpdateOne(sc.ctx, filter, &user)
	if err != nil {
		return err
	}

	if result.MatchedCount != 1 {
		return fmt.Errorf("not 1 document matched but %d", result.MatchedCount)
	}

	return nil
}
