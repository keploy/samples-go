package logic

import (
	"context"
	"encoding/json"
	"es4gophers/domain"
	"math/rand"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

func QueryMovieByDocumentID(rCtx context.Context, ctx context.Context, indexName string, docId string) string {
	client := ctx.Value(domain.ClientKey).(*elasticsearch.Client)

	rand.Seed(time.Now().UnixNano())
	greq := client.Get.WithContext(rCtx)
	response, err := client.Get(indexName, docId, greq)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	var getResponse = domain.GetResponse{}
	err = json.NewDecoder(response.Body).Decode(&getResponse)
	if err != nil {
		panic(err)
	}

	movieTitle := getResponse.Source.Title
	return movieTitle
}