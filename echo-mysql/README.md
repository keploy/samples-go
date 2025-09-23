# echo-mysql
A simple golang based url shortner


# Requirments to run
1. Golang [How to install Golang](https://go.dev/doc/install)
2. Docker [How to install Docker?](https://docs.docker.com/engine/install/)


# Setting up the project
Run the following commands to clone the repository and download the necessary Go modules


``` bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/echo-mysql
go mod download
```


# Running app

## Let's start the MySql Instance

``` bash
sudo docker run --name mysql-container -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=uss -p 3306:3306 --rm mysql:latest
```
## Build the application 

``` bash
 go build -o echo-mysql . 
 ```

# Capture the Testcases

``` bash
sudo -E env PATH=$PATH oss record -c "./echo-mysql"
```

To generate testcases we just need to make some API calls. You can use Postman, Hoppscotch, or simply curl


### 1) Basics

```bash
# health
curl -X GET http://localhost:9090/

# hello
curl -X GET http://localhost:9090/healthcheck

# create a short url
curl -X POST http://localhost:9090/shorten -H "Content-Type: application/json" -d '{"url": "https://github.com"}'

# resolve a short code
curl -X GET http://localhost:9090/resolve/4KepjkTT
```

### 2) Seed (single)

Seeds a single record with a far-future end time.

```bash
curl -i -X POST http://localhost:9090/seed
```

### 3) Seed datetime edge-cases

This **inserts a suite of tricky date/time rows** to help catch regressions:

* `dt-sentinel-9999-01-01T00:00:00Z` → end\_time `9999-01-01`
* `dt-max-9999-12-31T23:59:59.999999Z` → end\_time `9999-12-31 23:59:59.999999`
* `dt-min-1000-01-01T00:00:00Z` → end\_time `1000-01-01`
* `dt-epoch-1970-01-01T00:00:00Z` → end\_time `1970-01-01`
* `dt-leap-2020-02-29T12:34:56Z` → end\_time `2020-02-29 12:34:56`
* `dt-offset-2023-07-01T18:30:00+05:30` → end\_time normalized to local (e.g., `2023-07-01 13:00:00` if `loc=Local`)
* `dt-now-trunc` → end\_time is “now”, truncated to microseconds

Each is tagged with `created_by = "keploy.io/dates"`.

```bash
curl -i -X POST http://localhost:9090/seed/dates
```

### 4) Query helpers for replay validation

# Find rows with end_time exactly equal to the given timestamp (ISO-8601 accepted)
# e.g., date-only, full datetime, fractional seconds, or 'Z'
curl -i "http://localhost:9090/query/by-endtime?ts=9999-12-31T23:59:59.999999Z"
curl -i "http://localhost:9090/query/by-endtime?ts=9999-01-01T00:00:00Z"

# Return just the two sentinel extremes (min/max)
curl -i http://localhost:9090/query/sentinels

# Get a seeded row by 'short_code' label
curl -i http://localhost:9090/query/label/dt-leap-2020-02-29T12:34:56Z

# List ONLY the seeded date cases (created_by = keploy.io/dates)
curl -i http://localhost:9090/query/dates
```

> These queries are designed so Keploy can record exact MySQL traffic and later replay it to **catch subtle encoder/decoder regressions** in date/time handling.


Now both these API calls were captured as a testcase and should be visible on the Keploy CLI. You should be seeing an app named keploy folder with the test cases we just captured and data mocks created.

![alt text](https://github.com/Hermione2408/samples-go/blob/app/echo-mysql/img/keploy_record.png?raw=true)

# Run the captured testcases

Now that we have our testcase captured, run the test file.

```bash
sudo -E env PATH=$PATH oss test -c "./echo-mysql" --delay 20
```

So no need to setup dependencies like MySQL, web-go locally or write mocks for your testing.

oss test runs the test cases captured in the previous step. It replays the captured API calls against the application to verify its behavior. 

The application thinks it's talking to MySQL 😄

We will get output something like this:

![alt text](https://github.com/Hermione2408/samples-go/blob/app/echo-mysql/img/keploy_test.png?raw=true)