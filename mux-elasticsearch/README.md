# Introduction
This is a sample go project to show the crud operations of golang with elasticsearch and mux.

## Installation Setup

```bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/mux-elasticsearch
go mod download
```

## Installation Keploy
Install keploy via one-click:-

```sh
curl --silent -O -L https://keploy.io/install.sh && source install.sh
```

### Install And Run Elastic 

Using the docker we will start our elastic instance:-

```bash
docker run --name=es01 -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" -e "xpack.security.enabled=false" docker.elastic.co/elasticsearch/elasticsearch:8.14.3
```

In the above command we are passing two environment variables to run the elastic in dev mode and disabling the need to pass the password to connect to elastic database. If you want to enable it you can follow steps mentioned on this [page](https://www.elastic.co/guide/en/elasticsearch/reference/current/docker.html) . Then you have to pass the password and http cert file to connect.


### Capture the Testcases

Now, we will create the binary of our application:-

```zsh
go build -cover
```

Once we have our binary file ready,this command will start the recording of API calls using ebpf:-

```shell
sudo -E keploy record -c "./mux-elasticsearch"
```

Make API Calls using Hoppscotch, Postman or cURL command. Keploy with capture those calls to generate the test-suites containing testcases and data mocks.

#### Generate testcases

To genereate testcases we just need to make some API calls. You can use [Postman](https://www.postman.com/), [Hoppscotch](https://hoppscotch.io/), or simply `curl`

1. Post Command To Insert Document

```bash
curl --request POST \
      --url http://localhost:8000/documents \
      --header 'Content-Type: application/json' \
      --data '{
        "title" : "somethingTitle",
        "content" : "something22"
    }'
```
this will return the response which includes the id of the inserted document. 
```json
{"id":"1b5wJ5EBIPW7ZBPsse8e"}
```

2. Fetch the Products
```bash
curl --request GET \
  --url http://localhost:8000/documents/1b5wJ5EBIPW7ZBPsse8e
```

we will get output:

```json
{"content":"something22","title":"somethingTitle"}
```


Now let's run the test mode (in the mux-elasticsearch directory, not the Keploy directory).

### Run captured testcases

```shell
sudo -E keploy test -c "./mux-elasticsearch" --delay 10 --goCoverage
```


