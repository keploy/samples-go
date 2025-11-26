# URL Shortener

A sample url shortener app to test Keploy integration capabilities using [Echo](https://echo.labstack.com/) and PostgreSQL. 

## Installation Setup

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/echo-sql
go mod download
```

## Installation Keploy

Install keploy via one-click:-

```sh
curl --silent -O -L https://keploy.io/install.sh && source install.sh
```

### Start Postgres Instance 

Using the docker-compose file we will start our postgres instance ( use `podman-compose` if you are using RHEL based distribution ):-

```bash
# Start Postgres
docker-compose up -d postgres
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
go build -cover
```

Once we have our binary file ready,this command will start the recording of API calls using ebpf:-

```shell
sudo -E keploy record -c "./echo-psql-url-shortener"
```

Make API Calls using Hoppscotch, Postman or cURL command. Keploy with capture those calls to generate the test-suites containing testcases and data mocks.


#### Generate testcases

To generate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

### Generate shortened url

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
sudo -E keploy test -c "./echo-psql-url-shortener" --delay 10 --goCoverage
```

output should look like

![Testrun](./img/testrun.png?raw=true)

So no need to setup fake database/apis like Postgres or write mocks for them. Keploy automatically mocks them and, **The application thinks it's talking to Postgres 😄**

![keploy coverage](./img/coverage.png?raw=true)
