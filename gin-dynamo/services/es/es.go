package es

import (
	"context"
	"net/http"
	"sync"
	"user-onboarding/config"

	"github.com/olivere/elastic/v7"
	"go.elastic.co/apm/module/apmelasticsearch"
)

var client *elastic.Client
var esOnce sync.Once

func Init() *elastic.Client {
	esOnce.Do(func() {
		esURL := config.Get().EsURL
		Key := config.Get().Key
		var err error
		client, err = elastic.NewClient(elastic.SetURL(esURL), elastic.SetHttpClient(&http.Client{
			Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport),
		}), elastic.SetSniff(false), elastic.SetBasicAuth("elastic", Key), elastic.SetScheme("https"))
		if err != nil {
			panic(err.Error())
		}
		_, _, err = client.Ping(config.Get().EsURL).Do(context.Background()) //ping elasticsearch to see if connection is stable
		if err != nil {
			panic(err)
		}

	})
	return client
}

func Client() *elastic.Client {
	return client
}
