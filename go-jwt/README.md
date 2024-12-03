# Go-JWT

A sample app with CRUD operations using Go and JWT.

## Installation Setup

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/go-jwt
go mod download
```

## Installation Keploy

Install keploy via one-click:-

```sh
curl --silent -O -L https://keploy.io/install.sh && source install.sh
```

### Start the Postgres Database

```zsh
docker compose up -d db
```

### Capture the Testcases

Now, we will create the binary of our application:-

```zsh
go build .
```

Once we have our binary file ready,this command will start the recording of API calls using ebpf:-

```shell
sudo -E keploy record -c "./go-jwt"
```

Make API Calls using Hoppscotch, Postman or cURL command. Keploy with capture those calls to generate the test-suites containing testcases and data mocks.

#### Generate testcases

To genereate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

1. Generate shortned url

```bash
curl --request GET \
      --url http://localhost:8000/health \
      --header 'Accept: */*' \
      --header 'Host: localhost:8000' \
      --header 'User-Agent: curl/7.81.0'
```

this will return the response.

```
{"status": "healthy"}
```

2. Fetch the Products

```bash
curl --request GET \
      --url http://localhost:8000/generate-token \
      --header 'Host: localhost:8000' \
      --header 'User-Agent: curl/7.81.0' \
      --header 'Accept: */*'
```

we will get output:

```json
{ "token": "<your_jwt_token>" }
```

3. Fetch a single product

```sh
curl --request GET \
      --url http://localhost:8000/check-token?token=<your_jwt_token> \
      --header 'Accept: */*' \
      --header 'Host: localhost:8000' \
      --header 'User-Agent: curl/7.81.0'
```

we will get output:-

```json
{ "username": "example_user" }
```

Now, since these API calls were captured as editable testcases and written to `keploy/tests folder`. The keploy directory would also have `mocks` files that contains all the outputs.

![Testcase](./img/testcase.png?raw=true)

Now let's run the test mode (in the mux-sql directory, not the Keploy directory).

### Run captured testcases

```shell
sudo -E keploy test -c "./go-jwt" --delay 10
```

Once done, you can see the Test Runs on the Keploy server, like this:

![Testrun](./img/testrun.png?raw=true)
