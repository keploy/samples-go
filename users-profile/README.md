# Users-Profile

A sample application that get, create, update, and delete the data of a user in the database (MongoDB for this application).



## Prerequisites
1. [Go](https://go.dev/doc/install) 1.16 or later
2. [Docker](https://docs.docker.com/engine/install/) for running Keploy server
3. [Thunder Client](https://marketplace.visualstudio.com/items?itemName=rangav.vscode-thunder-client) / [Postman Desktop Agent](https://www.postman.com/downloads/postman-agent/) for testing localhost APIs
4. Code Editor ([VSCode](https://code.visualstudio.com/download), [Sublime Text](https://www.sublimetext.com/download), etc.)




## Start Keploy
> Note that Testcases are exported as files in the repo by default

### MacOS 
```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_darwin_all.tar.gz" | tar xz -C /tmp

sudo mv /tmp/keploy /usr/local/bin && keploy
```

### Linux
```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_linux_amd64.tar.gz" | tar xz -C /tmp


sudo mv /tmp/keploy /usr/local/bin && keploy
```

### Linux ARM
```shell
curl --silent --location "https://github.com/keploy/keploy/releases/latest/download/keploy_linux_arm64.tar.gz" | tar xz -C /tmp


sudo mv /tmp/keploy /usr/local/bin && keploy
```

The UI can be accessed at http://localhost:6789


## Start Docker and MongoDB locally
Open Docker application
```
docker container run -it -p27017:27017 mongo
```
This will initiate MongoDB locally


## Start Users-Profile sample application
```
git clone https://github.com/keploy/samples-go
cd samples-go
cd users-profile
go get .
export KEPLOY_MODE="record" 
go run .
```

> export KEPLOY_MODE="record" changes the env to record test cases




## Routes
> Sample Application Port: http://localhost:8080
- `/user` : POST - Create a new user in the database
- `/user/:userId` : GET - Get a user from the database
- `/user/:userId` : PUT - Edit an existing user in the database
- `/user/:userId` : DELETE - Delete an existing user from the database
- `/users` : GET - Get all users from the database


## Generate Test Cases
> Keploy Port: http://localhost:6789/testlist

To generate Test Cases, you need to make some API calls. It could be using Thunder Client, Postman Desktop Agent, or your preferred API testing tool.
here you can use this object template below for testing:
```shell
you can copy paste this object template:
curl -X POST -H "Content-Type: application/json" -d '{
  "username": "CurlyParadox",
  "name": "Nishant Mishra",
  "nationality": "Indian",
  "title": "Developer Advocate at Keploy",
  "hobbies": "Drumming",
  "linkedin": "@curlyparadox",
  "twitter": "@curlyParadox"
}' "http://localhost:8080/user"
```

**Post Request**
![POST-request](assets/POST-request.png)

**GET Request**
![GET-request](assets/GET-request.png)

Once done, you can see the Test Cases on the Keploy server, like this:

![test-cases-ss](assets/keploy-test-cases.png)

## Generate Test Runs

To generate Test Runs, close the application and run the below command:
```
export KEPLOY_MODE="test"
go test -v -coverpkg=./... -covermode=atomic  ./...
```


Once done, you can see the Test Runs on the Keploy server, like this:

![test-runs](assets/test-runs.png)


>NOTE:  To start Keploy CLI, run KEPLOY in ur terminal
```
Keploy
```

## Check the MongoDB database

To check the actual data being changed in the database. Open [MongoDB Compass](https://www.mongodb.com/products/compass) and enter the URI below to check the data.

> mongodb://localhost:27017



### If you like the sample application, Don't forget to star us ✨
