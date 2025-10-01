<h1 align="center"> Keploy Go Samples Apps </h1>
<p align="center">
## Introduction
<a href="CODE_OF_CONDUCT.md" alt="Contributions welcome">
    <img src="https://img.shields.io/badge/Contributions-Welcome-brightgreen?logo=github" /></a>

  <a href="https://join.slack.com/t/keploy/shared_invite/zt-12rfbvc01-o54cOG0X1G6eVJTuI_orSA" alt="Slack">
    <img src=".github/slack.svg" /></a>

  <a href="https://opensource.org/licenses/Apache-2.0" alt="License">
    <img src=".github/License-Apache_2.0-blue.svg" /></a>
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
**Note** :- Issue Creation is disabled on this Repository, please visit [here](https://github.com/keploy/keploy/issues/new/choose) to submit Issue.

So no need to setup dependencies like Redis, web-go locally or write mocks for your testing.

# Keploy Go Samples
This repo contains the sample for [Keploy's](https://keploy.io)🐰 with Golang. Please feel free to contribute if you'd like submit a sample for another use-case or library.

**The application thinks it's talking to Redis 😄**

We will get output something like this:

## Go Sample Apps with Keploy
![TestRun](./img/testRunFail.png)

1. [Echo-MySQL](https://github.com/keploy/samples-go/tree/main/echo-mysql)
2. [Echo-SQL](https://github.com/keploy/samples-go/tree/main/echo-sql)
3. [Fasthttp-Postgress](https://github.com/keploy/samples-go/tree/main/fasthttp-postgres)
4. [Gin-Mongo](https://github.com/keploy/samples-go/tree/main/gin-mongo)
5. [Gin-Redis](https://github.com/keploy/samples-go/tree/main/gin-redis)
6. [Go-Grpc](https://github.com/keploy/samples-go/tree/main/go-grpc)
7. [Go-Jwt](https://github.com/keploy/samples-go/tree/main/go-jwt)
8. [Go-Twilio](https://github.com/keploy/samples-go/tree/main/go-twilio)
9. [GraphQL-SQL](https://github.com/keploy/samples-go/tree/main/graphql-sql)
10. [Mux-SQL](https://github.com/keploy/samples-go/tree/main/mux-sql)
11. [Mux-Elasticsearch](https://github.com/keploy/samples-go/tree/main/mux-elasticsearch)
12. [Mux-MySQL](https://github.com/keploy/samples-go/tree/main/mux-mysql)
13. [S3-Keploy](https://github.com/keploy/samples-go/tree/main/S3-Keploy)
14. [SSE-Svelte](https://github.com/keploy/samples-go/tree/main/sse-svelte)
15. [Users-Profile](https://github.com/keploy/samples-go/tree/main/users-profile)
16. [HTTP-PokeAPI](https://github.com/keploy/samples-go/tree/main/http-pokeapi)
17. [book-store-inventory (`gin + sqlite`) ](https://github.com/keploy/samples-go/tree/main/book-store-inventory)

#### Let's add token to Noisy field:

In `test-2.yml` go to the noisefield and `-body.token` in noise. Now, it's the time to run the test cases again.

## Community Support ❤️
```bash
keploy test -c "sudo docker run -p 3001:3001 --rm --network <networkName> --name ginRedisApp gin-app:1.0" --delay 10
```
### 🤔 Questions?
Reach out to us. We're here to help!
This time all the test cases will pass.
![testruns](./img/testRunPass.png?raw=true "Recent testruns")

[![Slack](https://img.shields.io/badge/Slack-4A154B?style=for-the-badge&logo=slack&logoColor=white)](https://join.slack.com/t/keploy/shared_invite/zt-12rfbvc01-o54cOG0X1G6eVJTuI_orSA)
[![LinkedIn](https://img.shields.io/badge/linkedin-%230077B5.svg?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/company/keploy/)
[![YouTube](https://img.shields.io/badge/YouTube-%23FF0000.svg?style=for-the-badge&logo=YouTube&logoColor=white)](https://www.youtube.com/channel/UC6OTg7F4o0WkmNtSoob34lg)
[![Twitter](https://img.shields.io/badge/Twitter-%231DA1F2.svg?style=for-the-badge&logo=Twitter&logoColor=white)](https://twitter.com/Keployio)

## Run app Natively on local machine

### 💖 Let's Build Together!
Whether you're a newbie coder or a wizard 🧙‍♀️, your perspective is golden. Take a peek at our:
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
📜 [Contribution Guidelines](https://github.com/keploy/keploy/blob/main/CONTRIBUTING.md)
This way, you can easily configure the Redis connection for different environments without changing the code.


❤️ [Code of Conduct](https://github.com/keploy/keploy/blob/main/CODE_OF_CONDUCT.md)
**Future Improvements:**  
- Implement retry logic in the Go application to handle situations where Redis is not immediately available.
- Add health checks to the Docker Compose file to ensure Redis is fully initialized before starting the Go application.