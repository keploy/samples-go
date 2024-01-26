# URL Shortener

A sample url shortener app to test Keploy integration capabilities using [Echo](https://echo.labstack.com/) and PostgreSQL. 

## Installation Setup

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/echo-sql
go mod download
```

## Installation Keploy

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

### Start Postgres Instance 

Using the docker-compose file we will start our postgres instance ( use `podman-compose` if you are using RHEL based distribution ):-

```bash
# Start Postgres
docker-compose up -d
```

If there is an error saying keploy-network could not be found. Use the following command to create the docker network

```bash
docker network create keploy-network
```


### Update the Host

> **Since we have setup our sample-app natively, we need to update the Postgres host on line 27, in `main.go`, from `postgresDb` to `localhost`.**

### Capture the testcases

Now, we will create the binary of our application:-

```zsh
go build
```

Once we have our binary file ready,this command will start the recording of API calls using ebpf:-

```shell
sudo -E keploy record -c "./echo-psql-url-shortener"
```

Make API Calls using Hoppscotch, Postman or cURL command. Keploy with capture those calls to generate the test-suites containing testcases and data mocks.


#### Generate testcases

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

Now both these API calls were captured as **editable** testcases and written to `keploy/tests` folder. The keploy directory would also have `mocks` file that contains all the outputs of postgres operations. Here's what the folder structure look like:

![Testcase](./img/testcases.png?raw=true)

Now, let's see the magic! ✨💫

## Run the Testcases

Now that we have our testcase captured, we will add `ts` to noise field in `test-*.yaml` files. 

**1. On line 32 we will add "`body.ts: []`" under the "`header.Date: []`".**

![EliminateNoise](https://github.com/aswinbennyofficial/samples-go/assets/110408942/2b50d994-3418-4f7b-9f95-5bc1acd8ecf9)


Now let's run the test mode (in the echo-sql directory, not the Keploy directory).

```shell
sudo -E keploy test -c "./echo-psql-url-shortener" --delay 10
```

output should look like

![Testrun](./img/testrun.png?raw=true)

So no need to setup fake database/apis like Postgres or write mocks for them. Keploy automatically mocks them and, **The application thinks it's talking to Postgres 😄**

# Using Docker

Keploy can be used on Linux, Windows and MacOS through Docker.

Note: To run Keploy on MacOS through [Docker](https://docs.docker.com/desktop/release-notes/#4252) the version must be ```4.25.2``` or above.

## Create Keploy Alias
To establish a network for your application using Keploy on Docker, follow these steps.

If you're using a docker-compose network, replace keploy-network with your app's `docker_compose_network_name` below.

```shell
alias keploy='sudo docker run --pull always --name keploy-v2 -p 16789:16789 --privileged --pid=host -it -v "$(pwd)":/files -v /sys/fs/cgroup:/sys/fs/cgroup -v /sys/kernel/debug:/sys/kernel/debug -v /sys/fs/bpf:/sys/fs/bpf -v /var/run/docker.sock:/var/run/docker.sock --rm ghcr.io/keploy/keploy'
```
## Let's start the MongoDB Instance
Using the docker-compose file we will start our mongodb instance:-

```zsh
docker-compose up -d
```
> Since we are using docker to run the application, we need to update the `postgres` host on line 28 in `main.go`, update the host to `echo-sql-postgres-1`.

Now, we will create the docker image of our application:-

```zsh
docker build -t echo-app:1.0 .
```

## Capture the Testcases

```zsh
keploy record -c "docker run -p 8082:8082 --name echoSqlApp --network keploy-network echo-app:1.0"
```

![Testcase](./img/testcases.png?raw=true)

### Generate testcases
To genereate testcases we just need to make some API calls. You can use Postman, Hoppscotch, or simply curl

1. Generate shortned url

```zsh
curl --request POST \
  --url http://localhost:8082/url \
  --header 'content-type: application/json' \
  --data '{
  "url": "https://google.com"
}'
```
this will return the shortened url.
```json
{
  "ts": 1645540022,
  "url": "http://localhost:8082/Lhr4BWAi"
}
```
2. Redirect to original url from shòrtened url

```bash
curl --request GET \
  --url http://localhost:8082/Lhr4BWAi
or by querying through the browser http://localhost:8082/Lhr4BWAi 
```
Now, let's see the magic! 🪄💫

Now both these API calls were captured as a testcase and should be visible on the Keploy CLI. You should be seeing an app named keploy folder with the test cases we just captured and data mocks created.

## Run the captured testcases
Now that we have our testcase captured, run the test file.

```zsh
keploy test -c "sudo docker run -p 8082:8082 --net keploy-network --name echoSqlApp echo-app:1.0 echoSqlApp" --delay 10
```
So no need to setup dependencies like mongoDB, web-go locally or write mocks for your testing.

The application thinks it's talking to mongoDB 😄

We will get output something like this:
![Testrun](./img/testrun.png?raw=true)
