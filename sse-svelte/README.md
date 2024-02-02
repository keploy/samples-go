# Server Side events
A sample app to test Keploy integration capabilities with realtime subscriptions such as SSE

## Installation Setup

#### Server
```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/sse-svelte
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

### Start MongoDB Instance 
Using the docker-compose file we will start our mongodb instance:-

```bash
# Start Postgres
docker-compose up mongo
```

### Build Application Binary
Now, we will create the binary of our application:-

```bash
go build -o application .
```
Once we have our applicaiton binary ready, we will start the application with keploy to start capturing the testcases.

## Capture the test cases
```bash
sudo -E keploy record "./application"
```

### Start the UI
We will capture our test from the UI written in Svelte.js
```bash
cd svelte-app && npm install && npm run dev
```

## Run the Testcases
Now let's run the test mode :-

```shell
sudo -E keploy test -c "./keploy-gql" --delay 10
```

output should look like

![Testrun](./img/testrun.png?raw=true)

So no need to setup fake database/apis like Postgres or write mocks for them. Keploy automatically mocks them and, **The application thinks it's talking to MongoDb 😄**