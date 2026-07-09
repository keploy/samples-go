# Go-JWT
A sample app 

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

### Start the MySQL Database

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

Make API Calls using Hoppscotch, Postman or cURL command. Keploy will capture those calls to generate the test-suites containing testcases and data mocks.

#### Generate testcases

To genereate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

1. Check Health

```bash
curl --request GET \
      --url http://localhost:8000/health \
      --header 'Accept: */*' \
      --header 'Host: localhost:8000' \
      --header 'User-Agent: curl/7.81.0' 
```
This will return the response: 
```json
{"status": "healthy"}
```

2. Generate a token
```bash
curl --request GET \
      --url http://localhost:8000/generate-token?expiry=15 \
      --header 'Host: localhost:8000' \
      --header 'User-Agent: curl/7.81.0' \
      --header 'Accept: */*' 
```

You will get the following output:

```json
{"token":"<your_jwt_token>"}
```

3. Check the token

```sh
curl --request GET \
      --url http://localhost:8000/check-token?token=<your_jwt_token> \
      --header 'Accept: */*' \
      --header 'Host: localhost:8000' \
      --header 'User-Agent: curl/7.81.0' 
```

You will get the following output:
```json
{"username" : "example_user"}
```

4. Test the time-sensitive endpoint

This sample includes a script `test_time_endpoint.sh` to easily test an endpoint that depends on the current time.

First, make the script executable:
```sh
chmod +x test_time_endpoint.sh
```

Now, run the script. It will automatically use the current time for the API call.
```sh
./test_time_endpoint.sh
```

This will send a request to the `/check-time` endpoint and you should see a successful response:

Now, since these API calls were captured as editable testcases and written to the `keploy/tests` folder. The keploy directory would also have `mocks` files that contain all the outputs. 

![Testcase](./img/testcase.png?raw=true)

Now let's run the test mode (in the go-jwt directory, not the Keploy directory).

### Run captured testcases

```shell
sudo -E keploy test -c "./go-jwt" --delay 10
```

Once done, you can see the Test Runs on the Keploy server, like this:

![Testrun](./img/testrun.png?raw=true)