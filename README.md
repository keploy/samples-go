## Introduction

A sample user authentication to test Keploy integration capabilities using [Gin](https://gin-gonic.com/) and [Redis](https://redis.io/).

## Setup URL shortener

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/gin-redis
go mod download
```

## Installation

There are two methods to run the sample application using Keploy :-

1. [Using Docker](#running-app-using-docker)
2. [Natively on Ubuntu/Windows(using WSL)](#run-app-natively-on-local-machine)

## Running app using Docker

Keploy can be used on Linux, Windows and MacOS through [Docker](https://docs.docker.com/engine/install/).

Note: To run Keploy on MacOS through [Docker](https://docs.docker.com/desktop/release-notes/#4252) the version must be ```4.25.2``` or above.

### Create Keploy Alias

We need create an alias for Keploy:
```bash
alias keploy='sudo docker run --pull always --name keploy-v2 -p 16789:16789 --privileged --pid=host -it -v $(pwd):$(pwd) -w $(pwd) -v /sys/fs/cgroup:/sys/fs/cgroup -v /sys/kernel/debug:/sys/kernel/debug -v /sys/fs/bpf:/sys/fs/bpf -v /var/run/docker.sock:/var/run/docker.sock -v '"$HOME"'/.keploy-config:/root/.keploy-config -v '"$HOME"'/keploy-config:/root/keploy-config --rm ghcr.io/keploy/keploy'
```

### Update the Redis Connection Settings

> **Important:** When running in Docker, the Redis URL should be `redis:6379` instead of `localhost:6379`. This needs to be updated in `helpers/redis/redisConnect.go` file.

```go
// Change this:
Addr: "localhost:6379"

// To this:
Addr: "redis:6379"
```

This change is necessary because in Docker, `localhost` points to the container itself, not the Redis service. Docker Compose uses an internal network where each service can be accessed via its container name.

### Create a Docker network
```
sudo docker network create <networkName>
```

### Using Docker Compose

Create a `docker-compose.yml` file with the following content:

```yaml
version: '3.7'
services:
  go-app:
    build:
      context: .
    container_name: ginRedisApp
    ports:
      - "3001:3001"
    environment:
      - REDIS_HOST=redis
      - REDIS_PORT=6379
    depends_on:
      - redis
  redis:
    image: redis
    container_name: myredis
    ports:
      - "6379:6379"
```

This setup properly manages the dependency between the Go application and Redis server. 

To start the application using Docker Compose:

```bash
docker-compose up
```

### Using individual Docker containers

Alternatively, you can start Redis manually and build the application:

```bash
sudo docker run -p 6379:6379 -d --network <networkName> --name myredis redis
docker build -t gin-app:1.0 .
```

### Communicating with Dockerized App Natively

If you want to run your application in Docker but interact with it from your host machine or other environments, follow these steps:

#### Option 1: Port Mapping (Already included above)

The Docker Compose and individual container setups both include port mapping (`3001:3001`), which allows you to access the containerized application via `localhost:3001` from your host machine.

#### Option 2: Using Docker's IP Address

Find the Docker container's IP address:

```bash
docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ginRedisApp
```

Then use this IP to communicate with the application:

```bash
curl --location 'http://<container-ip>:3001/api/getVerificationCode?email=something@gmail.com&username=shivamsourav'
```

#### Option 3: Docker Host Network (Linux only)

Run your container using the host network:

```bash
docker run --network host --name ginRedisApp gin-app:1.0
```

This will make the container share the host's network stack, allowing you to access it via `localhost:3001`.

### Capture the Testcases

```shell
keploy record -c "docker run -p 3001:3001 --name RediApp --network <networkName> --name ginRedisApp gin-app:1.0"
```

To genereate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

### 1. Request OTP

```bash
curl --location 'localhost:3001/api/getVerificationCode?email=something@gmail.com&username=shivamsourav'
```

this will return the OTP response. 
```
{
    "status": "true",
    "message": "OTP Generated successfully",
    "otp": "5486"
}
```

**2. Verify OTP**

```bash
curl --location 'localhost:3001/api/verifyCode' \
--header 'Content-Type: application/json' \
--data-raw '{
    "otp":2121,
    "email":"something@gmail.com"
}'
```
this will return the OTP verification response. 
```
{
    "status": "true",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ2YWx1ZSI6ImdtYWlsLmNvbSIsImV4cCI6MTY5ODc1ODIyNn0.eVrNACUY93g-5tu8fxb2BEOs1wn2iCe8wVpUYU6OLSE",
    "username": "shivamsourav",
    "message": "OTP authenticated successfully"
}
```

_Now, let's see the magic! 🪄💫_

Now both these API calls were captured as a testcase and should be visible on the Keploy CLI. You should be seeing an app named `keploy folder` with the test cases we just captured and data mocks created.

### Run the captured testcases

Now that we have our testcase captured, run the test file.

```shell
keploy test -c "sudo docker run -p 3001:3001 --rm --network <networkName> --name ginRedisApp gin-app:1.0" --delay 10
```

So no need to setup dependencies like Redis, web-go locally or write mocks for your testing.

**The application thinks it's talking to Redis 😄**

We will get output something like this:

![TestRun](./img/testRunFail.png)

#### Let's add token to Noisy field:

In `test-2.yml` go to the noisefield and `-body.token` in noise. Now, it's the time to run the test cases again.

```bash
keploy test -c "sudo docker run -p 3001:3001 --rm --network <networkName> --name ginRedisApp gin-app:1.0" --delay 10
```

This time all the test cases will pass.
![testruns](./img/testRunPass.png?raw=true "Recent testruns")

## Run app Natively on local machine

Keploy can be installed on Linux directly and on Windows with the help of WSL. Based on your system archieture, install the keploy latest binary release

**1. AMD Architecture**

```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_linux_amd64.tar.gz" | tar xz -C /tmp

sudo mkdir -p /usr/local/bin && sudo mv /tmp/keploy /usr/local/bin && keploy
```

<details>
<summary> 2. ARM Architecture </summary>

```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_linux_arm64.tar.gz" | tar xz -C /tmp

sudo mkdir -p /usr/local/bin && sudo mv /tmp/keploy /usr/local/bin && keploy
```

</details>

#### Let's start the Redis Instance

Spin up your Redis container using

```shell
sudo docker run -p 6379:6379 -d --name myredis redis
```

### Capture the testcases

Now, we will create the binary of our application:-

```zsh
go build -o gin-redis
```

Once we have our binary file ready,this command will start the recording of API calls using ebpf:-

```shell
sudo -E keploy record -c "./gin-redis"
```

Make API Calls using Hoppscotch, Postman or cURL command. Keploy with capture those calls to generate the test-suites containing testcases and data mocks.

### Generate testcases

To generate testcases we just need to **make some API calls.** You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

**1. Request OTP**

```bash
curl --location 'localhost:3001/api/getVerificationCode?email=something@gmail.com&username=shivamsourav'
```

this will return the OTP response. 

```
{
    "status": "true",
    "message": "OTP Generated successfully",
    "otp": "5486"
}
```

**2. Verify OTP**

```bash
curl --location 'localhost:3001/api/verifyCode' \
--header 'Content-Type: application/json' \
--data-raw '{
    "otp":2121,
    "email":"something@gmail.com"
}'

```

this will return the OTP verification response. 
```
{
    "status": "true",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ2YWx1ZSI6ImdtYWlsLmNvbSIsImV4cCI6MTY5ODc1ODIyNn0.eVrNACUY93g-5tu8fxb2BEOs1wn2iCe8wVpUYU6OLSE",
    "username": "shivamsourav",
    "message": "OTP authenticated successfully"
}
```

You'll be able to see new test file and mock file generated in your project codebase locally.

### Run the Test Mode

Run this command on your terminal to run the testcases and generate the test coverage report:-

```shell
sudo -E keploy test -c "./gin-redis" --delay 10
```

> Note: If delay value is not defined, by default it would be `5`.

So no need to setup dependencies like Redis, web-go locally or write mocks for your testing.

**The application thinks it's talking to Redis 😄**

We will get output something like this:

![TestRun](./img/testRunFail.png)

#### Let's add token to Noisy field:

In `test-2.yml` go to the noisefield and `-body.token` in noise. Now, it's the time to run the test cases again.

```bash
sudo -E keploy test -c "./gin-redis" --delay 10
```

This time all the test cases will pass.

![testruns](./img/testRunPass.png?raw=true "Recent testruns")

## Known Issues and Fixes

### Redis Connectivity Issues in Dockerized Gin Application  

When using Docker, you might face connectivity issues between the Gin application and Redis. Here's how to resolve them:

#### Problem 1: Redis Connection Settings in Go Application  
**Issue:**  
The Go application might be configured to connect to Redis using:  
```go
Addr: "localhost:6379"
```  
This configuration works when running the application locally (without Docker) but fails in a containerized environment.

**Solution:**  
Update the Redis address in the application code to:  
```go
Addr: "redis:6379"
```  
Where `redis` matches the service name defined in `docker-compose.yml`.

#### Problem 2: Docker Compose Configuration  
**Issue:**  
The Docker Compose setup might not properly manage the dependency between the Go application and the Redis server.

**Solution:**  
Improve the `docker-compose.yml` to properly manage service dependencies as shown in the Docker Compose section above.

#### Environment-Variable Based Configuration (Recommended)

For a more flexible setup, modify your Go code to use environment variables:

```go
func RedisConnect() *redis.Client {
    redisHost := os.Getenv("REDIS_HOST")
    if redisHost == "" {
        redisHost = "localhost" // Default fallback
    }
    
    redisPort := os.Getenv("REDIS_PORT")
    if redisPort == "" {
        redisPort = "6379" // Default fallback
    }
    
    rdb := redis.NewClient(&redis.Options{
        Addr: redisHost + ":" + redisPort,
        Password: "",
        DB: 0,
    })
    
    return rdb
}
```

This way, you can easily configure the Redis connection for different environments without changing the code.

**Future Improvements:**  
- Implement retry logic in the Go application to handle situations where Redis is not immediately available.
- Add health checks to the Docker Compose file to ensure Redis is fully initialized before starting the Go application.
