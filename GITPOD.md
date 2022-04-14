## Quick Gitpod experience intructions
This sample app is a simple url shortner and located in echo-sql. It uses a postgres db. 

## Generate testcases

To genereate testcases we just need to make some API calls.

### Generate shortned url

```
curl --request POST \
  --url http://localhost:8080/url \
  --header 'content-type: application/json' \
  --data '{
  "url": "https://github.com"
}'
```

### Redirect to original url from shortened url

```
curl -L http://localhost:8080/4KepjkTT
```

## Run the testcases
```
go test -coverpkg=./... -covermode=atomic  ./...
```

**You should see around 75-80% coverage without writing any unit tests! You can also shutdown the database because keploy automatically mocks database calls during tests.**

You can also try out some negative cases by changing reponses in `handlers.go`