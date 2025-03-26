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

## Step 5: Run the Application
Start the application (assuming it's a Gin application) using:

```
./ginApp
```
Ensure the application is running before proceeding with the next steps.

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
sudo -E env PATH=$PATH <path to your keploy binary> test -c "go run main.go" --delay 30  &> logs.txt
```
This will execute the recorded tests and output the results to test_logs.txt.
