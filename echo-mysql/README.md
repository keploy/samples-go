# Echo-MySQL

A simple Go-based URL shortener application.

# Prerequisites

1. Golang: [Install Golang](https://go.dev/doc/install)
2. Docker: [Install Docker](https://docs.docker.com/engine/install/)

# Setting up the project

Clone the repository and download the necessary Go modules by running the following commands:

```bash
 git clone https://github.com/keploy/samples-go.git && cd samples-go/echo-mysql
 go mod download
```

# Running app

## Let's start the MySql Instance

Use Docker to run a MySQL container:

```bash
 sudo docker run --name mysql-container -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=uss -p 3306:3306 --rm mysql:latest
```

## Build the application

Compile the application with:

```bash
 go build -o echo-mysql .
```

# Capture the Testcases

To capture test cases, run the application in record mode:

```bash
 sudo -E env PATH=$PATH oss record -c "./echo-mysql"
```

To generate testcases we just need to make some API calls. You can use Postman, Hoppscotch, or simply curl

1. Root Endpoint:

```bash
 curl -X GET http://localhost:9090/
```

2. Health Check:

```bash
 curl -X GET http://localhost:9090/healthcheck
```

3. Short URL:

```bash
 curl -X POST http://localhost:9090/shorten -H "Content-Type: application/json" -d '{"url": "https://github.com"}'
```

4. Resolve short code:

```bash
 curl -X GET http://localhost:9090/resolve/4KepjkTT
```

Captured API calls will appear as test cases in the Keploy CLI. A new directory named keploy will contain the recorded test cases and data mocks.

## Example Screenshot

![alt text](https://github.com/Hermione2408/samples-go/blob/app/echo-mysql/img/keploy_record.png?raw=true)

# Run the captured testcases

Once the test cases are captured, you can run them with the following command:

```bash
 sudo -E env PATH=$PATH oss test -c "./echo-mysql" --delay 20
```

## Explanation

- The oss test command replays captured API calls to verify application behavior.
- Dependencies like MySQL are mocked during testing, so no additional setup is required.

The application will behave as if it is connected to MySQL, enabling seamless testing without manual mocks.

## Example Output

![alt text](https://github.com/Hermione2408/samples-go/blob/app/echo-mysql/img/keploy_test.png?raw=true)
