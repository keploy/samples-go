## Introduction

A sample url shortener app to test Keploy integration capabilities using [Gin](https://gin-gonic.com/) and [mongoDB](https://www.mongodb.com/).

## Setup URL shortener

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/gin-mongo
go mod download
```

## Installation

```bash
curl --silent -O -L https://keploy.io/install.sh && source install.sh
```

## Running app using Docker

Keploy can be used on Linux, Windows and MacOS through [Docker](https://docs.docker.com/engine/install/).

> Note: To run Keploy on MacOS through [Docker](https://docs.docker.com/desktop/release-notes/#4252) the version must be ```4.25.2``` or above.


### Let's start the MongoDB Instance
Using the docker-compose file we will start our mongodb instance:-

```bash
sudo docker run -p 27017:27017 -d --network keploy-network --name mongoDb mongo
```

Now, we will create the docker image of our application:-


```bash
docker build -t gin-app:1.0 .
```

### Capture the Testcases

```shell
keploy record -c "docker run -p 8080:8080 --name MongoApp --network keploy-network gin-app:1.0"
```

To generate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

### 1. Generate shortened url

```bash
curl --request POST \
  --url http://localhost:8080/url \
  --header 'content-type: application/json' \
  --data '{
  "url": "https://google.com"
}'
```

this will return the shortened url. 

```json
{
  "ts": 1645540022,
  "url": "http://localhost:8080/Lhr4BWAi"
}
```

**2. Redirect to original url from shortened url**

```bash
curl --request GET \
  --url http://localhost:8080/Lhr4BWAi
```

or by querying through the browser `http://localhost:8080/Lhr4BWAi` _Now, let's see the magic! 🪄💫_

Now both these API calls were captured as a testcase and should be visible on the Keploy CLI. You should be seeing an app named `keploy folder` with the test cases we just captured and data mocks created.

### Run the captured testcases

Now that we have our testcase captured, run the test file.

```shell
keploy test -c "sudo docker run -p 8080:8080 --net keploy-network --name ginMongoApp gin-app:1.0" --delay 10
```

So no need to setup dependencies like mongoDB, web-go locally or write mocks for your testing.

**The application thinks it's talking to mongoDB 😄**

We will get output something like this:

![TestRun](./img/testrun-fail-1.png)

Go to the Keploy log to get deeper insights on what testcases ran, what failed. The `ts` is causing be failure during testing because it'll always be different.

#### Let's add Timestamp to Noisy field:

In `test-1.yml` and `test-2.yml`, go the noisefield and under `-header.Data` add the `-body.ts` on line number _37_. Now, it's the time to run the test cases again.

```bash
keploy test -c "sudo docker run -p 8080:8080 --net keploy-network --name ginMongoApp gin-app:1.0" --delay 10
```

This time all the test cases will pass.

![testruns](./img/testrun.png?raw=true "Recent testruns")

## Run app Natively on local machine

#### Let's start the MongoDB Instance

Spin up your mongo container using

```shell
sudo docker run --rm -p27017:27017 -d --network keploy-network --name mongoDb mongo
```

### Update the Host

> **Since, we are on the local machine the MongoDB URL will be `localhost:27017`. This needs to be updated on the on line 21 in `main.go` file**

### Capture the testcases

Now, we will create the binary of our application:-

```zsh
go build -cover
```

Once we have our binary file ready,this command will start the recording of API calls using ebpf:-

```shell
sudo -E keploy record -c "./test-app-url-shortener"
```

Make API Calls using Hoppscotch, Postman or cURL command. Keploy with capture those calls to generate the test-suites containing testcases and data mocks.

### Generate testcases

To generate testcases we just need to **make some API calls.** You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

**1. Generate shortened url**

```bash
curl --request POST \
  --url http://localhost:8080/url \
  --header 'content-type: application/json' \
  --data '{
  "url": "https://google.com"
}'
```

this will return the shortened url. The ts would automatically be ignored during testing because it'll always be different.

```json
{
  "ts": 1645540022,
  "url": "http://localhost:8080/Lhr4BWAi"
}
```

**2. Redirect to original url from shortened url**

```bash
curl --request GET \
  --url http://localhost:8080/Lhr4BWAi
```

or by querying through the browser `http://localhost:8080/Lhr4BWAi`

You'll be able to see new test file and mock file generated in your project codebase locally.

### Run the Test Mode

Run this command on your terminal to run the testcases and generate the test coverage report:-

```shell
sudo -E keploy test -c "./test-app-url-shortener" --delay 10
```

> Note: If delay value is not defined, by default it would be `5`.

So no need to setup dependencies like mongoDB, web-go locally or write mocks for your testing.

**The application thinks it's talking to mongoDB 😄**

We will get output something like this:

![TestRun](./img/testrun-fail-2.png)

Go to the Keploy log to get deeper insights on what testcases ran, what failed. The `ts` is causing be failure during testing because it'll always be different.

#### Let's add Timestamp to Noisy field:

In `test-1.yml` and `test-2.yml`, go the noisefield and under `-header.Data` add the `-body.ts` on line number _37_. Now, it's the time to run the test cases again.

```shell
sudo -E keploy test -c "./test-app-url-shortener" --goCoverage --delay 10
```

This time all the test cases will pass.

![testruns](./img/testrun-coverage.png?raw=true "Recent testruns")
