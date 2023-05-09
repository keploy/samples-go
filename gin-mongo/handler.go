package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/itchyny/base58-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type url struct {
	ID      string    `json:"id" bson:"_id"`
	Created time.Time `json:"created" bson:"created"`
	Updated time.Time `json:"updated" bson:"updated"`
	URL     string    `json:"URL" bson:"url"`
}

func Get(ctx context.Context, id string) (*url, error) {
	filter := bson.M{"_id": id}
	var u url
	clientOptions := options.Client()

	clientOptions.ApplyURI("mongodb://" + "localhost:27017" + "/" + "keploy" + "?retryWrites=true&w=majority")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientOptions = clientOptions.SetHeartbeatInterval(40 * time.Second)
	client, err := mongo.Connect(ctx, clientOptions)
	// defer client.Disconnect(ctx)
	if err != nil {
		log.Fatal("failed to create mgo db client", zap.Error(err))
	}
	dbName, collection := "keploy", "url-shortener"
	db := client.Database(dbName)

	// integrate keploy with mongo
	// col = kmongo.NewCollection(db.Collection(collection))
	col := db.Collection(collection)
	err = col.FindOne(ctx, filter).Decode(&u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func Upsert(ctx context.Context, u url) error {
	upsert := true
	opt := &options.UpdateOptions{
		Upsert: &upsert,
	}
	filter := bson.M{"_id": u.ID}
	update := bson.D{{"$set", u}}

	clientOptions := options.Client()

	clientOptions.ApplyURI("mongodb://" + "localhost:27017" + "/" + "keploy" + "?retryWrites=true&w=majority")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	clientOptions = clientOptions.SetHeartbeatInterval(40 * time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	// defer client.Disconnect(ctx)
	if err != nil {
		log.Fatal("failed to create mgo db client", zap.Error(err))
	}
	dbName, collection := "keploy", "url-shortener"
	db := client.Database(dbName)

	// integrate keploy with mongo
	// col = kmongo.NewCollection(db.Collection(collection))
	col := db.Collection(collection)

	_, err = col.UpdateOne(ctx, filter, update, opt)
	if err != nil {
		return err
	}
	return nil
}

func get(c *gin.Context) {
	resp, err := http.Get("http://localhost:8082/ritik")
	if err != nil {
		log.Println("failed to make http call from handler. error: ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": `failed to make http call from handler. error: ` + err.Error()})
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read http response. error: ", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": `failed to read http response. error: ` + err.Error()})
	}

	log.Println("the response body: ", string(respBody))

	// Get(c.Request.Context(), "ritik")

	c.JSON(http.StatusOK, gin.H{
		"ts":  time.Now().UnixNano(),
		"url": "http://localhost:8080/",
	})
}

func getURL(c *gin.Context) {
	hash := c.Param("param")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "please append url hash"})
		return
	}
	log.Printf("recieved param: %v\n", hash)

	u, err := Get(c.Request.Context(), hash)
	if err != nil {
		logger.Error("failed to find url in the database", zap.Error(err), zap.String("hash", hash))
		c.JSON(http.StatusNotFound, gin.H{"error": "url not found"})
		return
	}
	c.Redirect(http.StatusSeeOther, u.URL)
	return
}

func putURL(c *gin.Context) {
	var m map[string]string

	err := c.ShouldBindJSON(&m)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decode req"})
		return
	}
	u := m["url"]

	if u == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing url param"})
		return
	}

	t := time.Now()
	id := GenerateShortLink(u)
	err = Upsert(c.Request.Context(), url{
		ID:      id,
		Created: t,
		Updated: t,
		URL:     u,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ts":  time.Now().UnixNano(),
		"url": "http://localhost:8080/" + id,
	})
}

func New(host, db string) (*mongo.Client, error) {
	clientOptions := options.Client()

	clientOptions = clientOptions.ApplyURI("mongodb://" + host + "/" + db + "?retryWrites=true&w=majority")

	clientOptions = clientOptions.SetHeartbeatInterval(4 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return mongo.Connect(ctx, clientOptions)
}

func GenerateShortLink(initialLink string) string {
	urlHashBytes := sha256Of(initialLink)
	generatedNumber := new(big.Int).SetBytes(urlHashBytes).Uint64()
	finalString := base58Encoded([]byte(fmt.Sprintf("%d", generatedNumber)))
	return finalString[:8]
}

func sha256Of(input string) []byte {
	algorithm := sha256.New()
	algorithm.Write([]byte(input))
	return algorithm.Sum(nil)
}

func base58Encoded(bytes []byte) string {
	encoding := base58.BitcoinEncoding
	encoded, _ := encoding.Encode(bytes)
	return string(encoded)
}
