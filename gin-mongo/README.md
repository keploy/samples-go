# URL Shortener
A sample url shortener app to test Keploy integration capabilities

## Installation
### Start keploy server
> Note that Testcases are exported as files in the repo by default

<details>
<summary>Mac</summary>

```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_darwin_all.tar.gz" | tar xz -C /tmp

sudo mv /tmp/keploy /usr/local/bin

# start keploy with default settings
keploy
```

</details>

<details>
<summary>Linux</summary>

```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_linux_amd64.tar.gz" | tar xz -C /tmp

sudo mv /tmp/keploy /usr/local/bin 

# start keploy with default settings
keploy
```

</details>


### Setup URL shortener
```bash
git clone https://github.com/keploy/example-url-shortener && cd gin-mongo

go mod download
```

### Run the application
```shell
# Start mongo server on localhost:27017
docker-compose up -d

# run the sample app
go run handler.go main.go

# run the sample app in record mode
export KEPLOY_MODE=record && go run handler.go main.go

```

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

### Redirect to original url from shortened url
1. By using Curl Command
```bash
curl --request GET \
  --url http://localhost:8080/Lhr4BWAi
```

2. By querying through the browser `http://localhost:8080/Lhr4BWAi`

Now both these API calls were captured as editable testcases and written to keploy/tests folder. The keploy directory would also have mocks folder that contains all the outputs of postgres operations. Here's what the folder structure look like:

```
.
├── README.md
├── docker-compose.yml
├── go.mod
├── go.sum
├── handler.go
├── keploy
│   ├── tests
│       ├── test-1.yaml
│       ├── test-2.yaml
│       ├── test-3.yaml
│   └── mocks
│       ├── mock-1.yaml
│       └── mock-2.yaml
│       └── mock-3.yaml

```

![testcases](https://imgur.com/bcEvNED)


Now, let's see the magic! 🪄💫

## Generate Test Runs

To generate Test Runs, close the application and run the below command:
```
export KEPLOY_MODE="test"
go test -v -coverpkg=./... -covermode=atomic  ./...
```


Once done, you can see the Test Runs on the Keploy server, like this:
![test-runs](https://imgur.com/77bd1Oi)

### Make a code change
Now try changing something like renaming `url` to `urls` in [handlers.go](./handler.go) on line 96 and running ` go test -coverpkg=./... -covermode=atomic  ./...` again
```shell
{"msg":"result","testcase id": "test-3", "passed": false}
{"msg":"result","testcase id": "test-2", "passed": false}
{"msg":"result","testcase id": "test-1", "passed": false}
{"msg":"test run completed","run id": "97d0870c-737d-414d-80af-c56802f974e8", "passed overall": false}
--- FAIL: TestKeploy (5.77s)
FAIL
coverage: 64.8% of statements in ./...
FAIL    test-app-url-shortener  6.787s
FAIL
```