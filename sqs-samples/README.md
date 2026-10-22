# SQS LocalStack Setup

This guide will walk you through setting up **LocalStack** with **SQS**, **MongoDB**, and interacting with the services using `awslocal`. We'll also walk through building the repository, sending test messages to SQS, and using Keploy to record and test the application.

## Prerequisites

Before proceeding, ensure you have the following installed:

- **Docker** (for running MongoDB)
- **Python 3** (for installing LocalStack using `pipx`)
- **Go** (for building the Go application)
- **Keploy** (for recording and testing)

---

## Step 1: Set up LocalStack

### 1.1 Install LocalStack via `pipx`

First, install **`pipx`** (if not already installed):

```bash
python3 -m pip install --upgrade pipx
python3 -m pipx ensurepath
```
Now, install LocalStack using pipx:
```
pipx install --include-deps localstack
```


After installing LocalStack, install the AWS CLI Local tool (awslocal):
```
pipx inject localstack awscli-local
pipx ensurepath
```

This will allow you to interact with LocalStack using awslocal.
1.2 **Start LocalStack**

Start LocalStack in the background:

```
localstack start -d
```
Wait for LocalStack to initialize. It may take a few seconds to be fully ready.

## Step 2: **Set up MongoDB with Docker**


### 2.1 Run MongoDB
We will now run MongoDB using Docker. Use the following command to start MongoDB in a Docker container:
```
docker run -p 27017:27017 --rm --network keploy-network --name mongoDb mongo
```

## Step 3: Set up SQS Queue in LocalStack
### 3.1 Create SQS Queue
You can create an SQS queue in LocalStack using awslocal. First, create the SQS queue with the following command:

```
awslocal sqs create-queue --queue-name localstack-queue
```
This will create a queue named localstack-queue.

### 3.2 Populate the SQS Queue
To populate the SQS queue with some test messages, use the following commands:

```
awslocal sqs send-message --queue-url http://localhost:4566/000000000000/localstack-queue --message-body "Test message 1"

awslocal sqs send-message --queue-url http://localhost:4566/000000000000/localstack-queue --message-body "Test message 2"
```
This sends two test messages to the localstack-queue in LocalStack.

## Step 4: Build the Repository
## 4.1 Build the Go Application
After setting up LocalStack and MongoDB, it's time to build your Go application. Navigate to your repository directory and run the following:

```
go mod tidy
go build -o ginApp .
```
This will build the Go application and output the executable ginApp.

### 4.2 Builds that speak SQS's Query protocol

The AWS SDKs speak to SQS in one of two wire protocols. The default build
(aws-sdk-go-v2, `service/sqs` v1.38.1) speaks JSON: an `X-Amz-Target:
AmazonSQS.ReceiveMessage` header and a JSON body. SDKs from before SQS's move
to JSON speak the Query protocol: a form body
(`Action=ReceiveMessage&...&Version=2012-11-05`) and XML answers. Two builds of
the same app speak it, one per Go SDK family, each pinned to the family's last
release before JSON:

```
# aws-sdk-go v1.47.5 (v1.47.6 is the first that speaks JSON)
go build -tags sqs_sdk_v1 -o ginApp .

# aws-sdk-go-v2 service/sqs v1.26.0 (v1.27.0 is the first that speaks JSON)
go build -modfile=go.v2query.mod -o ginApp .
```

`go.v2query.mod` is `go.mod` with `service/sqs` and the SDK modules it needs
pinned to their releases of that time. Keploy's CI records and replays all
three builds. `main_test.go` tests the default build and the `go.v2query.mod` one
(`go test -modfile=go.v2query.mod .`).

## Step 5: Run the Application
Start the application (assuming it's a Gin application) using:

```
./ginApp
```
Ensure the application is running before proceeding with the next steps. `GET /healthz` answers `200` once the app's SQS consumer has polled the queue for the first time and processed what that poll delivered (for each message: a read of whether its event was recorded already, the insert into MongoDB, and the delete from the queue, whether they succeeded or not), and `503` before (with the error, if the consumer stopped). A first poll that finds the queue empty makes the app ready at once. Once `200`, it stays `200`. The app pings MongoDB once as it starts, on its one client.

`SQS_S12_FORCED=1` runs the variant keploy's S12 lane forces a consumer's work into a request with: the app is ready after its first poll, before processing it; polls are short (`WaitTimeSeconds: 0`, with a 200 ms pause after an empty answer); and the first `POST /url` it parses, once it has written its URL, waits (at most 25 s) for the consumer's next delete before it answers; every later one answers at once.

## Step 6: Send Requests Using curl
## 6.1 Send POST Request to Shorten URL
Use curl to send a POST request to your application (adjust the URL accordingly if your app runs on a different port):

```
curl --request POST --url http://localhost:8080/url --header 'Content-Type: application/json' --data '{"url": "https://google.com"}'
```
This will send a URL to your application to be shortened.

## 6.2 Send GET Request to Retrieve the URL
After the URL is shortened, you can use curl to send a GET request to retrieve the shortened URL:

```
curl --request GET http://localhost:8080/Lhr4BWAi
```
Make sure to replace Lhr4BWAi with the actual shortened URL ID returned from the POST request.

## Step 7: Keploy - Record and Test
### 7.1 Run Keploy Record
Start Keploy in record mode to capture the interactions between your application and the services (LocalStack, MongoDB):

```
sudo -E env PATH=$PATH <path to your keploy binary> record -c "./ginApp" 
```
Hit the curl commands and populate sqs queue to mimic the working behaiviour

This will run your application in record mode and capture all interactions with LocalStack and MongoDB.

### 7.2 Run Keploy Test
Once the interactions are captured, you can run Keploy in test mode to validate the captured requests:

```
sudo -E env PATH=$PATH <path to your keploy binary> test -c "go run ." --delay 30  &> logs.txt
```
This will execute the recorded tests and output the results to logs.txt.
To start the tests as soon as the app has polled SQS, add `--health-url http://localhost:8080/healthz`.
