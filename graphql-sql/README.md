# Blog Post App

A sample  app to test Keploy integration capabilities using [Go Chi](https://go-chi.io/), [GraphQL](https://graphql.org/) and PostgreSQL. 

## Installation Setup

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/graphql-sql
go mod download
```

## Installation Keploy

Keploy can be installed on Linux directly and on Windows with the help of WSL. Based on your system archieture, install the keploy latest binary release

**1. AMD Architecture**

```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_linux_amd64.tar.gz" | tar xz -C /tmp

sudo mkdir -p /usr/local/bin && sudo mv /tmp/keploy /usr/local/bin && keploy
```

<details>
<summary> 2. ARM Architecture </summary>

```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_linux_arm64.tar.gz" | tar xz -C /tmp

sudo mkdir -p /usr/local/bin && sudo mv /tmp/keploy /usr/local/bin && keploy
```

</details>

### Start Postgres Instance 

Using the docker-compose file we will start our postgres instance:-

```bash
# Start Postgres
docker-compose up -d
```

### Update the Host

> **Since we have setup our sample-app natively, we need to set the Postgres host on line 18, in `main.go` to `localhost`.**

### Capture the testcases

Now, we will create the binary of our application:-

```zsh
go build
```

Once we have our binary file ready,this command will start the recording of API calls using ebpf:-

```shell
sudo -E keploy record -c "./keploy-gql"
```

Make API Calls using Hoppscotch, Postman or cURL command. Keploy with capture those calls to generate the test-suites containing testcases and data mocks.


#### Generate testcases

To generate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

### Create Author

```bash
curl --request POST \
  --url http://localhost:8080/graphql \
  --header 'content-type: application/json' \
  --data '{
  mutation CreateAuthor {
    createAuthor(name: "Abc xyz", email: "abc@xyz.com") {
        created_at
        email
        id
        name
    }
}'
```

this will return the shortened url. The ts would automatically be ignored during testing because it'll always be different.

```
{
    "data": {
        "createAuthor": {
            "created_at": "2023-09-21 05:28:10.873001673 +0000 UTC m=+9.250422297",
            "email": "abc@xyz.com",
            "id": 0,
            "name": "Abc xyz"
        }
    }
}
```

### Redirect to original URL from shortened URL 

1. By using Curl Command
```bash
curl --request GET \
  --url http://localhost:8080/GuwHCgoQ
```

2. Or by querying through the browser `http://localhost:8082/GuwHCgoQ`

Now both these API calls were captured as **editable** testcases and written to `keploy/tests` folder. The keploy directory would also have `mocks` file that contains all the outputs of postgres operations. Here's what the folder structure look like:

![Testcase](./img/testcases.png?raw=true)

Now, let's see the magic! ✨💫

## Run the Testcases

Now that we have our testcase captured, we will add `ts` to noise field in `test-*.yaml` files. 

Now let's run the test mode (in the echo-sql directory, not the Keploy directory).

```shell
sudo -E keploy test -c "./keploy-gql" --delay 10
```

output should look like

![Testrun](./img/testrun.png?raw=true)

So no need to setup fake database/apis like Postgres or write mocks for them. Keploy automatically mocks them and, **The application thinks it's talking to Postgres 😄**