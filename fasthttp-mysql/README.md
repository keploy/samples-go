# Movie List

A sample application to get th list of movies from the API and display it in a list.

## Prerequisites

- [Go](https://go.dev/doc/install) 1.16 or higher
- [Docker](https://docs.docker.com/engine/install/) for running mySQL database and keploy
- API client like [Postman](https://www.postman.com/downloads/), [Insomnia](https://insomnia.rest/download/), [Hoppscotch](https://hoppscotch.io/) or cURL.

## Start Keploy

> Note that Testcases are exported as files in the repo by default

### MacOS

```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_darwin_all.tar.gz" | tar xz -C /tmp

sudo mv /tmp/keploy /usr/local/bin && keploy
```

### Linux

```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_linux_amd64.tar.gz" | tar xz -C /tmp


sudo mv /tmp/keploy /usr/local/bin && keploy
```

### Linux ARM

```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_linux_arm64.tar.gz" | tar xz -C /tmp


sudo mv /tmp/keploy /usr/local/bin && keploy
```

The UI can be accessed at <http://localhost:6789>

## Start mySQL database

```bash
docker build -t ksql . && docker run -itd --rm -p 3306:3306 ksql
```

## Start Users-Profile sample application

```bash
git clone https://github.com/keploy/samples-go
cd samples-go
cd fasthttp-sql
go get .
export KEPLOY_MODE="record" 
go run .
```

> export KEPLOY_MODE="record" changes the env to record test cases

## Routes
>
> Sample Application Port: <http://localhost:8081>

- POST /movie - Add a movie
- GET /movie - Get last added movie
- GET /movies - Get all movies
<!-- - GET /movies?year={value} - Get all movies of year {value}
- GET /movies?rating={value} - Get all movies of rating {value}
- GET /movies?year={value}&rating={value2} - Get all movies of year {value} and rating {value2} -->

## Generate Test Cases

> Keploy Port: <http://localhost:6789/testlist>

Start keploy server in a new terminal

```bash
keploy
```

To generate Test Cases, you need to make some API calls. It could be using Thunder Client, Postman Desktop Agent, or your preferred API testing tool.

POST movie in the database

```bash
curl --request POST \
 --url http://localhost:8080/movie \
 --header 'content-type: application/json' \
 --data '{
    "title":"Everything Everywhere All At Once",
    "year":2022,
    "rating":10
}'
```

GET movie in the database

```bash
curl --request GET \
 --url http://localhost:8080/movie
```

GET all movies in the database

```bash
curl --request GET \
 --url http://localhost:8080/movies
```

## Generate Test Runs

To generate Test Runs, close the application and run the below command:

```bash
export KEPLOY_MODE="test"
go test -v -coverpkg=./... -covermode=atomic  ./...
```

Once done, you can see the Test Runs on the Keploy server, like this:

<!-- 
Add Test Runs image here
 -->

```bash
Keploy
```

## Check the MySQL database

To check the actual data being changed in the database. Open [MySQL Workbench](https://www.mysql.com/products/workbench/) and enter the URI below to check the data.

```bash
mysql://root:password@localhost:3306/movies
```

### If you like the sample application, Don't forget to star us ✨
