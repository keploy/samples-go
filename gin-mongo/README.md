# Example URL Shortener
A sample url shortener app to test Keploy integration capabilities

## Installation
### Start keploy server
```shell
git clone https://github.com/keploy/keploy.git && cd keploy
docker-compose up
```

### Setup URL shortener
```bash
git clone https://github.com/keploy/example-url-shortener && cd example-url-shortener
go mod download
```

### Run the application
```shell
go run handler.go main.go

```
### Skip above steps with Gitpod

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/Sarthak160/samples-go/tree/gin-mongo-gitpod)

## Generate testcases

To genereate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

###1. Generate shortned url

```bash
curl --request POST \
  --url http://localhost:8080/url \
  --header 'content-type: application/json' \
  --data '{
  "url": "https://google.com"
}'
```
this will return the shortened url. The ts would automatically be ignored during testing because it'll always be different. 
```
{
  "ts": 1645540022,
  "url": "http://localhost:8080/Lhr4BWAi"
}
```

###2. Redirect to original url from shortened url
```bash
curl --request GET \
  --url http://localhost:8080/Lhr4BWAi
```

or by querying through the browser `http://localhost:8080/Lhr4BWAi`


Now both these API calls were captured as a testcase and should be visible on the [Keploy console](http://localhost:8081/testlist).
If you're using Keploy cloud, open [this](https://app.keploy.io/testlist).

You should be seeing an app named `sample-url-shortener` with the test cases we just captured.

![testcases](testcases.png?raw=true "Web console testcases")


Now, let's see the magic! 🪄💫


## Test mode

Now that we have our testcase captured, run the test file.
```shell
 go test -coverpkg=./... -covermode=atomic  ./...
```
output should look like
```shell
ok      test-app-url-shortener  6.268s  coverage: 80.3% of statements in ./...
```

**We got 80.3% without writing any testcases or mocks for mongo db!!**

So no need to setup dependencies like mongoDB, web-go locally or write mocks for your testing.

**The application thinks it's talking to
mongoDB 😄**

Go to the Keploy Console/testruns to get deeper insights on what testcases ran, what failed.

![testruns](testrun1.png?raw=true "Recent testruns")
![testruns](testrun2.png?raw=true "Summary")
![testruns](testrun3.png?raw=true "Detail")

### Make a code change
Now try changing something like renaming `url` to `urls` in [handlers.go](./handler.go) on line 96 and running ` go test -coverpkg=./... -covermode=atomic  ./...` again
```shell
{"msg":"result","testcase id":"05a576e1-c03a-4c25-a469-4bea0307cd08","passed":false}
{"msg":"result","testcase id":"cad6d926-b531-477c-935c-dd7314c4357a","passed":true}
{"msg":"test run completed","run id":"19d4cba1-b77c-4301-884a-5b3f08dc6248","passed overall":false}
--- FAIL: TestKeploy (5.72s)
    keploy.go:42: Keploy test suite failed
FAIL
coverage: 80.3% of statements in ./...
FAIL    test-app-url-shortener  6.213s
FAIL
```

To deep dive the problem go to [test runs](http://localhost:8081/testruns)

![testruns](testrun4.png?raw=true "Recent testruns")
![testruns](testrun5.png?raw=true "Detail")