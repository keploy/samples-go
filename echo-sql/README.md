# URL Shortener

A sample url shortener app to test Keploy integration capabilities using [Echo](https://echo.labstack.com/) and PostgreSQL. 

## Installation Setup

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

### Start URL shortener application 

```bash
git clone https://github.com/keploy/samples-go && cd samples-go/echo-sql

# Start Postgres
docker-compose up -d

# run the sample app in record mode
export KEPLOY_MODE=record && go run handler.go main.go
```

#### Skip above steps with Gitpod

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/from-referrer)

## Generate testcases

To generate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

### Generate shortned url

```bash
curl --request POST \
  --url http://localhost:8082/url \
  --header 'content-type: application/json' \
  --data '{
  "url": "https://github.com"
}'
```

this will return the shortened url. The ts would automatically be ignored during testing because it'll always be different.

```
{
	"ts": 1647802058801841100,
	"url": "http://localhost:8082/GuwHCgoQ"
}
```

### Redirect to original URL from shortened URL 

1. By using Curl Command
```bash
curl --request GET \
  --url http://localhost:8082/GuwHCgoQ
```

2. Or by querying through the browser `http://localhost:8082/GuwHCgoQ`

Now both these API calls were captured as **editable** testcases and written to `keploy-tests` folder. The folder would also have a mocks folder and contain `mocks` of the postgres operations. Here's what the folder structure look like:

```
.
├── README.md
├── docker-compose.yml
├── go.mod
├── go.sum
├── handler.go
├── keploy-tests
│   ├── cb32305f-1bcc-4eab-a6f3-e3e815cb9f48.yaml
│   ├── db7ca851-550b-4c0b-b3e4-00d5809a2f72.yaml
│   └── mocks
│       ├── cb32305f-1bcc-4eab-a6f3-e3e815cb9f48.yaml
│       └── db7ca851-550b-4c0b-b3e4-00d5809a2f72.yaml


```

The test files should look like the sample below and the format is common for both ***http tests*** and ***mocks***. 
```yaml
version: api.keploy.io/v1beta1
kind: Http
name: cb32305f-1bcc-4eab-a6f3-e3e815cb9f48
spec:
    metadata: {}
    req:
        method: POST
        proto_major: 1
        proto_minor: 1
        url: /url
        header:
            Accept: '*/*'
            Content-Length: "33"
            Content-Type: application/json
            User-Agent: curl/7.79.1
        body: |-
            {
              "url": "https://github.com"
            }
    resp:
        status_code: 200
        header:
            Content-Type: application/json; charset=UTF-8
        body: |
            {"ts":1664887399743163000,"url":"http://localhost:8082/4KepjkTT"}
        status_message: ""
        proto_major: 1
        proto_minor: 1
    objects:
        - type: error
          data: ""
    mocks:
        - cb32305f-1bcc-4eab-a6f3-e3e815cb9f48-0
        - cb32305f-1bcc-4eab-a6f3-e3e815cb9f48-1
        - cb32305f-1bcc-4eab-a6f3-e3e815cb9f48-2
        - cb32305f-1bcc-4eab-a6f3-e3e815cb9f48-3
        - cb32305f-1bcc-4eab-a6f3-e3e815cb9f48-4
        - cb32305f-1bcc-4eab-a6f3-e3e815cb9f48-5
    assertions:
        noise:
            - body.ts
    created: 1664887399

```

Now, let's see the magic! ✨💫

## Test mode

Now that we have our testcase captured, run the test file (in the echo-sql directory, not the Keploy directory).

```shell
 go test -coverpkg=./... -covermode=atomic  ./...
```

output should look like

```shell
ok      echo-psql-url-shortener 6.534s  coverage: 51.1% of statements in ./...
```

> **We got 51.1% without writing any e2e testcases or mocks for Postgres!**

So no need to setup fake database/apis like Postgres or write mocks for them. Keploy automatically mocks them and, **The application thinks it's talking to Postgres 😄**

Go to the `Keploy Console` or the `UI` to get deeper insights on what testcases ran, what failed.

<details>
<summary>𝗜𝗻𝘀𝗶𝗴𝗵𝘁𝘀 𝗼𝗻 𝗞𝗲𝗽𝗹𝗼𝘆 𝗖𝗼𝗻𝘀𝗼𝗹𝗲</summary>

```shell
 <=========================================> 
  TESTRUN STARTED with id: "2115e16c-00b9-469a-aad0-e4ff196b02e2"
        For App: "sample-url-shortener"
        Total tests: 3
 <=========================================> 

Testrun passed for testcase with id: "cb32305f-1bcc-4eab-a6f3-e3e815cb9f48"

--------------------------------------------------------------------

Testrun passed for testcase with id: "db7ca851-550b-4c0b-b3e4-00d5809a2f72"

--------------------------------------------------------------------

Testrun passed for testcase with id: "34ba2282-1224-4fde-89de-f337e0a4816c"

--------------------------------------------------------------------


 <=========================================> 
  TESTRUN SUMMARY. For testrun with id: "2115e16c-00b9-469a-aad0-e4ff196b02e2"
        Total tests: 3
        Total test passed: 3
        Total test failed: 0
 <=========================================> 

```

![testruns](https://i.imgur.com/euROA3X.png)
![testruns](https://user-images.githubusercontent.com/21143531/159177972-8f1b0c92-05ea-4c10-9583-47ddb5e952be.png)
![testrun](https://user-images.githubusercontent.com/21143531/159178008-f7d38738-d841-437a-a3b4-bb7e5b07b808.png)

</details>

---

### Make a code change

Now try changing something like renaming `url` to `urls` in [handler.go](./handler.go) on line 39 and running ` go test -coverpkg=./... -covermode=atomic ./...` again

```shell
starting test execution {"id": "3b20b4d9-58d4-485e-97a7-70856591bd7a", "total tests": 3}
testing 1 of 3  {"testcase id": "34ba2282-1224-4fde-89de-f337e0a4816c"}
testing 2 of 3  {"testcase id": "db7ca851-550b-4c0b-b3e4-00d5809a2f72"}
testing 3 of 3  {"testcase id": "cb32305f-1bcc-4eab-a6f3-e3e815cb9f48"}
result  {"testcase id": "cb32305f-1bcc-4eab-a6f3-e3e815cb9f48", "passed": false}
result  {"testcase id": "db7ca851-550b-4c0b-b3e4-00d5809a2f72", "passed": true}
result  {"testcase id": "34ba2282-1224-4fde-89de-f337e0a4816c", "passed": true}
test run completed      {"run id": "3b20b4d9-58d4-485e-97a7-70856591bd7a", "passed overall": false}
--- FAIL: TestKeploy (6.03s)
    keploy.go:77: Keploy test suite failed
FAIL
coverage: 51.1% of statements in ./...
FAIL    echo-psql-url-shortener 6.620s
FAIL

```


To deep dive the problem you can look at the keploy logs or goto [the UI](http://localhost:6789/testruns)
<details>
<summary>𝗞𝗲𝗽𝗹𝗼𝘆 𝗟𝗼𝗴𝘀</summary>

```shell
 <=========================================> 
  TESTRUN STARTED with id: "3b20b4d9-58d4-485e-97a7-70856591bd7a"
        For App: "sample-url-shortener"
        Total tests: 3
 <=========================================> 

Testrun failed for testcase with id: "cb32305f-1bcc-4eab-a6f3-e3e815cb9f48"
Test Result:
        Input Http Request: models.HttpReq{
  Method:     "POST",
  ProtoMajor: 1,
  ProtoMinor: 1,
  URL:        "/url",
  URLParams:  map[string]string{},
  Header:     http.Header{
    "Accept": []string{
      "*/*",
    },
    "Content-Length": []string{
      "33",
    },
    "Content-Type": []string{
      "application/json",
    },
    "User-Agent": []string{
      "curl/7.79.1",
    },
  },
  Body: "{\n  \"url\": \"https://github.com\"\n}",
}

        Expected Response: models.HttpResp{
  StatusCode: 200,
  Header:     http.Header{
    "Content-Type": []string{
      "application/json; charset=UTF-8",
    },
  },
  Body:          "{\"ts\":1664887399743163000,\"url\":\"http://localhost:8082/4KepjkTT\"}\n",
  StatusMessage: "",
  ProtoMajor:    0,
  ProtoMinor:    0,
}

        Actual Response: models.HttpResp{
  StatusCode: 200,
  Header:     http.Header{
    "Content-Type": []string{
      "application/json; charset=UTF-8",
    },
  },
  Body:          "{\"ts\":1664891977696880000,\"urls\":\"http://localhost:8082/4KepjkTT\"}\n",
  StatusMessage: "",
  ProtoMajor:    0,
  ProtoMinor:    0,
}

DIFF: 
        Response body: {
                "url": {
                        Expected value: "http://localhost:8082/4KepjkTT"
                        Actual value: nil
                }
                "urls": {
                        Expected value: nil
                        Actual value: "http://localhost:8082/4KepjkTT"
                }
        }
--------------------------------------------------------------------

Testrun passed for testcase with id: "db7ca851-550b-4c0b-b3e4-00d5809a2f72"

--------------------------------------------------------------------

Testrun passed for testcase with id: "34ba2282-1224-4fde-89de-f337e0a4816c"

--------------------------------------------------------------------


 <=========================================> 
  TESTRUN SUMMARY. For testrun with id: "3b20b4d9-58d4-485e-97a7-70856591bd7a"
        Total tests: 3
        Total test passed: 2
        Total test failed: 1
 <=========================================> 

```

![recent test runs](https://user-images.githubusercontent.com/21143531/159178101-403e9fab-f92b-4db3-87d7-1abdef0a7a7d.png)

![expected vs actual response](https://user-images.githubusercontent.com/21143531/159178125-9cffa7b5-509d-40ea-be4f-985b7b85d877.png)

</details>