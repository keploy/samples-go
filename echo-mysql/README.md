# echo-mysql
A simple golang based url shortner


# Requirments to run
1. Golang [How to install Golang](https://go.dev/doc/install)
2. Docker [How to install Docker?](https://docs.docker.com/engine/install/)
3. Keploy [Quick Installation (API test generator)](https://github.com/keploy/keploy?tab=readme-ov-file#-quick-installation-api-test-generator)


# Setting up the project
Run the following commands to clone the repository and download the necessary Go modules


``` bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/echo-mysql
go mod download
```


# Running app

## Let's start the MySql Instance

``` bash
docker run --name mysql-container --network keploy-network -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=uss -p 3306:3306 --rm -d mysql:latest
```

If the docker network doesn't exist, create it first.

```bash
docker network create keploy-network
```

## Build the docker image for the application 

``` bash
docker build -t echo-mysql-app . 
```

# Capture the Testcases

``` bash
keploy record \                                                                                 
-c "docker run -p 9090:9090 --name echo-mysql-container --network keploy-network --rm echo-mysql-app" \
--container-name "echo-mysql-container" \
--buildDelay 60
```

To generate testcases we just need to make some API calls. You can use Postman, Hoppscotch, or simply curl


1. Root Endpoint:
```bash
-> curl -X GET http://localhost:9090/
```


2. Health Check:
```bash
-> curl -X GET http://localhost:9090/healthcheck
```


3. Shorten URL:
```bash
-> curl -X POST http://localhost:9090/shorten -H "Content-Type: application/json" -d '{"url": "https://github.com"}'
```


4. Resolve shortened code:

```bash
-> curl -X GET http://localhost:9090/resolve/4KepjkTT
```

Now both these API calls were captured as a testcase and should be visible on the Keploy CLI. You should be seeing an app named keploy folder with the test cases we just captured and data mocks created.

![alt text](https://github.com/Hermione2408/samples-go/blob/app/echo-mysql/img/keploy_record.png?raw=true)

# Run the captured testcases

Now that we have our testcase captured, run the test file.

```bash
keploy test \                                                                                   
-c "docker run -p 9090:9090 --name echo-mysql-container --network keploy-network --rm echo-mysql-app" \
--delay 20
```

So no need to setup dependencies like MySQL, web-go locally or write mocks for your testing. 

keploy test runs the test cases captured in the previous step. It replays the captured API calls against the application to verify its behavior. 

The application thinks it's talking to MySQL 😄

You can verify this by stopping the mysql container and running the tests again.

```bash
docker stop mysql-container
```

We will get output something like this:

![alt text](https://github.com/Hermione2408/samples-go/blob/app/echo-mysql/img/keploy_test.png?raw=true)
