# FastHttp-Postgres

A sample application that get, create, update, and delete the data of a user in the database: - 

## Start Users-Profile sample application
```
git clone https://github.com/keploy/samples-go && cd fasthttp-postgres

```

## Installation

```bash
curl --silent -O -L https://keploy.io/install.sh && source install.sh
```
## Run the application without Docker

### Prerequisites
- Go 1.22+
- PostgreSQL running locally

### Setup Database
Create a database and apply migrations:
```bash
createdb db
psql db < migrations/init.sql
```
### Set environment variables
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=password
export DB_NAME=db

```
### Run the server
```bash
go run main.go
```
The server will start at:
```md
```bash
 http://localhost:8080
 ```
 

Keploy can be used on Linux, Windows and MacOS through [Docker](https://docs.docker.com/engine/install/).

> Note: To run Keploy on MacOS through [Docker](https://docs.docker.com/desktop/release-notes/#4252) the version must be ```4.25.2``` or above.

### Let's start the Server
Using the docker-compose file we will start :-
```bash
 docker-compose up --build
```

### Capture the Testcases

```shell
keploy record -c "docker-compose up" --container-name=fasthttp_app
```

To generate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`: -

1. Post Requests
```shell
curl -X POST -H "Content-Type: application/json" -d '{"name":"Author Name"}' http://localhost:8080/authors

curl -X POST -H "Content-Type: application/json" -d '{"title":"Book Title","author_id":1}' http://localhost:8080/books
```

2. Get Requests
```bash
curl -i http://localhost:8080/books
```

![Keploy Testcases](./img/testcases.png)

### Run captured tests

Now that we have our testcase captured, run the test file.

```shell
keploy test -c "docker-compose up" --container-name=fasthttp_app --delay 10
```

![alt text](./img/testrun.png)

_Voila! Our testcases have passed🥳_ . We can also notice that by capturing just a few API calls we achieved high test coverage with Keploy generated testcases


If you like the sample application, Don't forget to star us ✨
