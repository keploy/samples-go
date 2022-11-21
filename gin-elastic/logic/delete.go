package logic

import (
	"context"
	"es4gophers/domain"
	"math/rand"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

func DeleteMovieByDocumentID(rCtx context.Context, ctx context.Context, indexName string, docId string) {
	client := ctx.Value(domain.ClientKey).(*elasticsearch.Client)

	rand.Seed(time.Now().UnixNano())
	dreq := client.Delete.WithContext(rCtx)
	response, err := client.Delete(indexName, docId, dreq)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
}