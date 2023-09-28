// To connect to the MongoDB database with our application
package configs

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectDB function first configures the client to use the correct URI and check for errors.
// Secondly, we defined a timeout of 10 seconds we wanted to use when trying to connect.
// Thirdly, check if there is an error while connecting to the database and cancel the connection if the connecting period exceeds 10 seconds.
// Finally, we pinged the database to test our connection and returned the client instance.
func ConnectDB() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(EnvMongoURI()))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	defer cancel()

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB")
	return client
}

// Creating a DB variable instance of the ConnectDB.
// This will come in handy when creating collections.
var DB *mongo.Client = ConnectDB()

// GetCollection function to retrieve and create collections on the database.
func GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	database := client.Database("users-profile")
	collection := database.Collection(collectionName)
	return collection
}
