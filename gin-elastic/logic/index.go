package logic

import (
	"context"
	"es4gophers/domain"
	"fmt"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

func IndexMoviesAsDocuments(rCtx context.Context, ctx context.Context, indexName string) {

	movies := ctx.Value(domain.MoviesKey).([]domain.Movie)
	client := ctx.Value(domain.ClientKey).(*elasticsearch.Client)

	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:      indexName,
		Client:     client,
		NumWorkers: 200,
	})
	if err != nil {
		panic(err)
	}

	for documentID, document := range movies {
		err = bulkIndexer.Add(
			rCtx,
			esutil.BulkIndexerItem{
				Action:     "index",
				DocumentID: strconv.Itoa(documentID),
				Body:       esutil.NewJSONReader(document),
			},
		)
		if err != nil {
			panic(err)
		}
	}

	bulkIndexer.Close(rCtx)
	biStats := bulkIndexer.Stats()
	fmt.Printf("✅ Movies indexed on Elasticsearch: %d \n", biStats.NumIndexed)

}
