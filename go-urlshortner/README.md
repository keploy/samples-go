# go-urlshortner
A simple golang based url shortner


# Requirments to run
1. Golang [How to install Golang](https://go.dev/doc/install)
2. Docker [How to install Docker?](https://docs.docker.com/engine/install/)


# Setting up the project
Run the following commands to setup the project


``` bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/go-urlshortner
go mod download
```


# How to use


```bash
## Shorten


curl --location 'localhost:9090/shorten' \
--header 'Content-Type: application/json' \
--data '{
   "url" : "github.com/labstack/echo"
}'


## Expand


curl --location 'localhost:9090/resolve/H2eBb6CN'


```


# Additional Endpoints


1. Root Endpoint:
```bash
-> curl -X GET http://localhost:9090/
```


2. Health Check:
```bash
-> curl -X GET http://localhost:9090/healthcheck
```


3. Short URL:
```bash
-> curl -X POST http://localhost:9090/shorten -H "Content-Type: application/json" -d '{"url": "https://github.com"}'
```


4. Resolve short code:
```bash
-> curl -X GET http://localhost:9090/resolve/4KepjkTT
```

