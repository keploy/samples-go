# Example URL Shortener
A sample url shortener app to test Keploy integration capabilities

## Installation Using Docker

### Setup URL shortener

Clone the repository and navigate to the `Gin-Mongo` folder.
```bash
git clone https://github.com/keploy/samples-go && cd gin-mongo

go mod download
```

### Create Keploy Alias

```bash
alias keploy='sudo docker run --name keploy-v2 -p 16789:16789 --privileged --pid=host -it -v "$(pwd)":/files -v /sys/fs/cgroup:/sys/fs/cgroup -v /sys/kernel/debug:/sys/kernel/debug -v /sys/fs/bpf:/sys/fs/bpf -v /var/run/docker.sock:/var/run/docker.sock --rm ghcr.io/keploy/keploy'
```

### Let's start the MongoDB Instance
Using the docker-compose file we will start our mongodb instance:-
```bash
docker compose up -d
```

Now, we will create the docker image of our application:-

```bash
docker build -t gin-app:1.0 .
```
### Capture the Testcases

```shell
keploy record -c "docker run -p 8080:8080 --name MongoApp --network keploy-network gin-app:1.0" --containerName "MongoApp" --delay 10
```

## Generate testcases

To genereate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

### 1. Generate shortned url

```bash
curl --request POST \
  --url http://localhost:8080/url \
  --header 'content-type: application/json' \
  --data '{
  "url": "https://google.com"
}'
```
this will return the shortened url. 
```
{
  "ts": 1645540022,
  "url": "http://localhost:8080/Lhr4BWAi"
}
```

###2. Redirect to original url from shortened url
```bash
curl --request GET \
  --url http://localhost:8080/Lhr4BWAi
```

or by querying through the browser `http://localhost:8080/Lhr4BWAi`

Now, let's see the magic! 🪄💫

Now both these API calls were captured as a testcase and should be visible on the Keploy CLI. 

You should be seeing an app named `keploy folder` with the test cases we just captured and data mocks created.


## Test mode

Now that we have our testcase captured, run the test file.
```shell
keploy test -c "sudo docker run -p 8010:8010 --rm --net keploy-network --name MongoApp gin-app:1.0" --delay 10
```
So no need to setup dependencies like mongoDB, web-go locally or write mocks for your testing.

**The application thinks it's talking to
mongoDB 😄**

We will get output something like this:

![TestRun](./img/testrun-fail-1.png)
![TestRun](./img/testrun-fail-2.png)

Go to the Keploy log to get deeper insights on what testcases ran, what failed. 

The `ts` is causing be failure during testing because it'll always be different. 

### Let's add Timestamp to Noisy field:
In `test-1.yml` and `test-2.yml`, go the noisefield and under `-header.Data` add the `-body.ts`. Now, it's the time to run the test cases again.

```bash
keploy test -c "sudo docker run -p 8010:8010 --rm --net keploy-network --name MongoApp gin-app:1.0" --delay 10
```

This time all the test cases will pass.

![testruns](./img/testrun.png?raw=true "Recent testruns")

