# Example URL Shortener

A sample url shortener app to test Keploy integration capabilities using [Echo](https://echo.labstack.com/) and PostgreSQL. 

## Installation

### Start keploy server

```shell
git clone https://github.com/keploy/keploy.git && cd keploy
docker-compose up
```

### Setup URL shortener

```bash
git clone https://github.com/keploy/samples-go && cd samples-go/echo-sql
go mod download
```

### Start PostgreSQL instance
```bash
docker-compose up -d
```

### Run the application

```shell
go run handler.go main.go
```

### Skip above steps with Gitpod

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/Sarthak160/samples-go/tree/echo-sql-gitpod)

## Generate testcases

To genereate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

### Generate shortned url

```bash
curl --request POST \
  --url http://localhost:8080/url \
  --header 'content-type: application/json' \
  --data '{
  "url": "https://github.com"
}'
```

this will return the shortened url. The ts would automatically be ignored during testing because it'll always be different.

```
{
	"ts": 1647802058801841100,
	"url": "http://localhost:8080/GuwHCgoQ"
}
```

### Redirect to original url from shortened url

```bash
curl --request GET \
  --url http://localhost:8080/GuwHCgoQ
```

or by querying through the browser `http://localhost:8080/GuwHCgoQ`

Now both these API calls were captured as a testcase and should be visible on the [Keploy console](http://localhost:8081/testlist).
If you're using Keploy cloud, open [this](https://app.keploy.io/testlist).

You should be seeing an app named `sample-url-shortener` with the test cases we just captured.

![testcases](https://i.imgur.com/7I4TY07.png)

Now, let's see the magic! 🪄💫

## Test mode

Now that we have our testcase captured, run the test file (in the echo-sql directory, not the Keploy directory).

```shell
 go test -coverpkg=./... -covermode=atomic  ./...
```

output should look like

```shell
ok      echo-psql-url-shortener 5.820s  coverage: 74.4% of statements in ./...
```

**We got 74.4% without writing any testcases or mocks for Postgres!**

So no need to setup dependencies like PostgreSQL, web-go locally or write mocks for your testing.

**The application thinks it's talking to
Postgres 😄**

Go to the Keploy Console/testruns to get deeper insights on what testcases ran, what failed.

![testruns](https://i.imgur.com/euROA3X.png)
![testruns](https://user-images.githubusercontent.com/21143531/159177972-8f1b0c92-05ea-4c10-9583-47ddb5e952be.png)
![testrun](https://user-images.githubusercontent.com/21143531/159178008-f7d38738-d841-437a-a3b4-bb7e5b07b808.png)

### Make a code change

Now try changing something like renaming `url` to `urls` in [handler.go](./handler.go) on line 39 and running ` go test -coverpkg=./... -covermode=atomic ./...` again

```shell
starting test execution {"id": "2b2c95e2-bf5e-4a58-a20e-f8d75b453d11", "total tests": 2}
testing 1 of 2  {"testcase id": "5a6695f8-8158-42cc-a860-d6f290145f70"}
testing 2 of 2  {"testcase id": "9833898b-b654-43a6-a581-ca20b1b92f0f"}
result  {"testcase id": "9833898b-b654-43a6-a581-ca20b1b92f0f", "passed": false}
result  {"testcase id": "5a6695f8-8158-42cc-a860-d6f290145f70", "passed": true}
test run completed      {"run id": "2b2c95e2-bf5e-4a58-a20e-f8d75b453d11", "passed overall": false}
--- FAIL: TestKeploy (5.32s)
    keploy.go:43: Keploy test suite failed
FAIL
coverage: 74.4% of statements in ./...
FAIL    echo-psql-url-shortener 5.795s
FAIL
```

To deep dive the problem go to [test runs](http://localhost:8081/testruns)

![recent test runs](https://user-images.githubusercontent.com/21143531/159178101-403e9fab-f92b-4db3-87d7-1abdef0a7a7d.png)

![expected vs actual response](https://user-images.githubusercontent.com/21143531/159178125-9cffa7b5-509d-40ea-be4f-985b7b85d877.png)
