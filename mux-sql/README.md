# Product Catelog
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


### Setup Application
```bash
git clone https://github.com/keploy/samples-go && cd mux-sql

go mod download
```

### Run the application
```shell
# Start mongo server on localhost:27017
docker-compose up -d

# run the sample app in record mode
export KEPLOY_MODE=record && go run .

```

## Generate testcases

To genereate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

###1. Generate shortned url

```bash
curl --request POST \
  --url http://localhost:8010/url \
  --header 'content-type: application/json' \
  --data '{
    "name":"Bubbles", 
    "price": 123
}'
```
this will return the response. 
```
{
    "id": 1,
    "name": "Bubbles",
    "price": 123
}
```

### Redirect to original url from shortened url
1. By using Curl Command
```bash
curl --request GET \
  --url http://localhost:8010/products
```

2. By querying through the browser `http://localhost:8010/products`

Now both these API calls were captured as editable testcases and written to ``keploy/tests folder``. The keploy directory would also have mocks folder that contains all the outputs of postgres operations. Here's what the folder structure look like:

```
.
├── README.md
├── docker-compose.yml
├── go.mod
├── go.sum
├── keploy
│   ├── tests
│       ├── test-1.yaml
│       ├── test-2.yaml
│   └── mocks
│       ├── mock-1.yaml
│       └── mock-2.yaml

```

<img width="929" alt="testcases" src="https://user-images.githubusercontent.com/53110238/217449242-d2b24d72-426a-4da5-8196-6929a652cad4.png">

Now, let's see the magic! 🪄💫

## Generate Test Runs

To generate Test Runs, close the application and run the below command:
```
export KEPLOY_MODE="test"
go test -v -coverpkg=./... -covermode=atomic  ./...
```

Once done, you can see the Test Runs on the Keploy server, like this:

<img width="657" alt="testrun" src="https://user-images.githubusercontent.com/53110238/217449300-3a41c3e8-9d93-488c-a47f-0c6928236419.png">


