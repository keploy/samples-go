# Product Catalog
A sample product catalog API (Gorilla Mux + PostgreSQL) to test Keploy integration capabilities.

## Installation Setup

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/mux-sql
go mod download
```

## Install Keploy
Install Keploy with the one-line installer:

```sh
curl --silent -O -L https://keploy.io/install.sh && source install.sh
```

## Configuration

The app reads its database location from environment variables, so it runs unchanged in Docker and natively:

| Variable | Default | Used for |
| --- | --- | --- |
| `DB_HOST` | `localhost` | Postgres host. `docker-compose.yml` sets it to `postgres`. |
| `DB_PORT` | `5432` | Postgres port the app connects to. |
| `POSTGRES_HOST_PORT` | `5432` | Host port Compose publishes Postgres on. Change it if `5432` is already taken on your machine. |

## Option 1: Run with Docker Compose

### Capture the testcases

```bash
keploy record -c "docker compose up" --container-name "muxSqlApp" --build-delay 50
```

The `go-app` service uses `pull_policy: build`, so its image is rebuilt (from cache) on every run and Keploy always exercises your latest code.

Then make the API calls from [Generate testcases](#generate-testcases) below and stop recording with `Ctrl+C`.

### Run captured testcases

```bash
keploy test -c "docker compose up" --container-name "muxSqlApp" --build-delay 50 --delay 10
```

## Option 2: Run natively (Linux / WSL)

### Start Postgres instance

Using the docker-compose file we will start our Postgres instance:

```bash
docker compose up -d postgres
```

The app connects to `localhost:5432` by default. If you published Postgres on another port, run `export DB_PORT=<port>` first.

### Capture the testcases

Now, we will create the binary of our application:

```zsh
go build -cover
```

Once we have our binary file ready, this command will start the recording of API calls using eBPF:

```shell
sudo -E keploy record -c "./test-app-product-catelog"
```

Make API calls using Hoppscotch, Postman or cURL. Keploy will capture those calls to generate test suites containing testcases and data mocks.

### Run captured testcases

```shell
sudo -E keploy test -c "./test-app-product-catelog" --delay 10 --goCoverage
```

## Generate testcases

To generate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`.

1. Create a product

```bash
curl --request POST \
  --url http://localhost:8010/product \
  --header 'content-type: application/json' \
  --data '{
    "name":"Bubbles",
    "price": 123
}'
```

This will return the response:

```json
{"id":1,"name":"Bubbles","price":123}
```

2. Fetch the products

```bash
curl --request GET \
  --url http://localhost:8010/products
```

We will get the output:

```json
[{"id":1,"name":"Bubbles","price":123}]
```

3. Fetch a single product

```sh
curl --request GET \
  --url http://localhost:8010/product/1
```

We will get the output:

```json
{"id":1,"name":"Bubbles","price":123}
```

These API calls are captured as editable testcases in `keploy/test-set-0/tests/`. The same test set also has a `mocks.yaml` file that contains all the Postgres interactions.

![Testcase](./img/testcase.png?raw=true)

Once the test run is done, you can see the test runs on the Keploy server, like this:

![Testrun](./img/testrun.png?raw=true)

So no need to set up a fake database or write mocks for Postgres. Keploy automatically mocks them, and **the application thinks it's talking to Postgres 😄**

With the three API calls that we made, we got around `55.6%` of coverage

![test coverage](./img/coverage.png?raw=true)

## Create Unit Testcase with Keploy

### Prerequisites
AI model API_KEY to use:

- OpenAI's GPT-4o.
- Alternative LLMs via [litellm](https://github.com/BerriAI/litellm?tab=readme-ov-file#quick-start-proxy---cli).

### Setup

Download Cobertura formatted coverage report dependencies.

```bash
go install github.com/axw/gocov/gocov@v1.1.0
go install github.com/AlekSi/gocov-xml@v1.1.0 
```

Get API key from [OpenAI](https://platform.openai.com/) or API Key from other LLM

```bash
export API_KEY=<LLM_MODEL_API_KEY>
```

### Generate Unit tests

Let's check the current code coverage of our application:

```bash
go test -v ./... -coverprofile=coverage.out && gocov convert coverage.out | gocov-xml > coverage.xml
```
We got around 44.1% of code coverage.

![Go Test](./img/mux-utg.png?raw=true)

Now, let's run Keploy to create testcases.

```bash
keploy gen --sourceFilePath="app.go" --testFilePath="app_test.go" --testCommand="go test -v ./... -coverprofile=coverage.out && gocov convert coverage.out | gocov-xml > coverage.xml" --coverageReportPath="./coverage.xml" --expectedCoverage 70 --maxIterations 2
```

With the above command, Keploy will generate new testcases in our `app_test.go` that can increase code coverage up to 70%.

![Keploy Mux UTG](./img/mux-utg-codecov.png?raw=true)