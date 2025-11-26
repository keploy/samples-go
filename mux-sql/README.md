# Product Catelog
A sample url shortener app to test Keploy integration capabilities

## Installation Setup

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/mux-sql
go mod download
```

## Installation Keploy
Install keploy via one-click:-

```sh
curl --silent -O -L https://keploy.io/install.sh && source install.sh
```

### Start Postgres Instance 

Using the docker-compose file we will start our postgres instance:-

```bash
docker-compose up -d postgres
```

### Update the Host

> **Since we have setup our sample-app natively set the host to `localhost` on line 10.**

### Capture the Testcases

Now, we will create the binary of our application:-

```zsh
go build -cover
```

Once we have our binary file ready,this command will start the recording of API calls using ebpf:-

```shell
sudo -E keploy record -c "./test-app-product-catelog"
```

Make API Calls using Hoppscotch, Postman or cURL command. Keploy with capture those calls to generate the test-suites containing testcases and data mocks.

#### Generate testcases

To generate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

1. Generate shortened url

```bash
curl --request POST \
  --url http://localhost:8010/product \
  --header 'content-type: application/json' \
  --data '{
    "name":"Bubbles", 
    "price": 123
}'
```
this will return the response. 
```
{
    "id": 1,
    "name": "Bubbles",
    "price": 123
}
```

2. Fetch the Products
```bash
curl --request GET \
  --url http://localhost:8010/products
```

we will get output:

```json
[{"id":1,"name":"Bubbles","price":123},{"id":2,"name":"Bubbles","price":123}]
```

3. Fetch a single product

```sh
curl --request GET \
  --url http://localhost:8010/product/1

we will get output:-
```json
{"id":1,"name":"Bubbles","price":123}
```

Now, since these API calls were captured as editable testcases and written to ``keploy/tests folder``. The keploy directory would also have `mocks` files that contains all the outputs of postgres operations. 

![Testcase](./img/testcase.png?raw=true)

Now let's run the test mode (in the mux-sql directory, not the Keploy directory).

### Run captured testcases

```shell
sudo -E keploy test -c "./test-app-product-catelog" --delay 10 --goCoverage
```

Once done, you can see the Test Runs on the Keploy server, like this:

![Testrun](./img/testrun.png?raw=true)

So no need to setup fake database/apis like Postgres or write mocks for them. Keploy automatically mocks them and, **The application thinks it's talking to Postgres 😄**

With the three API calls that we made, we got around `55.6%` of coverage

![test coverage](./img/coverage.png?raw=true)

## Create Unit Testcase with Keploy

### Prequiste
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

Let's check the current code coverage of out application : - 

```bash
go test -v ./... -coverprofile=coverage.out && gocov convert coverage.out | gocov-xml > coverage.xml
```
We got around 44.1% of code coverage.

![Go Test](./img/mux-utg.png?raw=true)

Now, let's run create keploy to create testcases.

```bash
keploy gen --sourceFilePath="app.go" --testFilePath="app_test.go" --testCommand="go test -v ./... -coverprofile=coverage.out && gocov convert coverage.out | gocov-xml > coverage.xml" --coverageReportPath="./coverage.xml" --expectedCoverage 70 --maxIterations 2
```

With the above command, Keploy will generate new testcases in the our `app_test.go` can increase code coverage upto 70%.

![Keploy Mux UTG](./img/mux-utg-codecov.png?raw=true)
