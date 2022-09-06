package logic

import (
	"context"
	"es4gophers/domain"
	"math/rand"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

func DeleteMovieByDocumentID(rCtx context.Context, ctx context.Context, indexName string, docId string) {

	// movies := ctx.Value(domain.MoviesKey).([]domain.Movie)
	client := ctx.Value(domain.ClientKey).(*elasticsearch.Client)

	rand.Seed(time.Now().UnixNano())
	// documentID := rand.Intn(len(movies) - 1)
	dreq := client.Delete.WithContext(rCtx)
	response, err := client.Delete(indexName, docId, dreq)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	// var getResponse = domain.GetResponse{}
	// err = json.NewDecoder(response.Body).Decode(&getResponse)
	// if err != nil {
	// 	panic(err)
	// }

	// movieTitle := getResponse.Source.Title
	// // fmt.Printf("✅ Movie with the ID %d: %s \n", documentID, movieTitle)
}

// func DeleteIndex(ctx context.Context, indexName string) {
// 	client := ctx.Value(domain.ClientKey).(*elasticsearch.Client)

// 	rand.Seed(time.Now().UnixNano())
// 	// documentID := rand.Intn(len(movies) - 1)
// 	// response, err := client.Bulk(ctx)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer response.Body.Close()
// }
