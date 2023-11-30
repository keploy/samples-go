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

Keploy can be used on Linux & Windows through [Docker](https://docs.docker.com/engine/install/), and on MacOS by the help of [Colima](https://docs.keploy/io/server/macos/installation)


### Create Keploy Alias

We need create an alias for Keploy:
```bash
alias keploy='sudo docker run --pull always --name keploy-v2 -p 16789:16789 --privileged --pid=host -it -v "$(pwd)":/files -v /sys/fs/cgroup:/sys/fs/cgroup -v /sys/kernel/debug:/sys/kernel/debug -v /sys/fs/bpf:/sys/fs/bpf -v /var/run/docker.sock:/var/run/docker.sock -v '"$HOME"'/.keploy-config:/root/.keploy-config -v '"$HOME"'/keploy-config:/root/keploy-config --rm ghcr.io/keploy/keploy'
```

> **Since, we are on the docker image the Redis URL will be myredis:6379 instead of localhost:6379. This needs to be updated in `helpers/redis/redisConnect.go` file**

### Create a Docker network
```
sudo docker network create <networkName>
```

### Let's start the Redis Instance
Using the docker-compose file we will start our Redis instance:-
```bash
sudo docker run -p 6379:6379 -d --network <networkName> --name myredis redis
```
```bash
docker build -t gin-app:1.0 .
```

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