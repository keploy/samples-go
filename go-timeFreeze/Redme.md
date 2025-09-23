## before building, kindly ensure to move the time freezing agent binary to this directory


## command to build docker image 

```
docker build -t go-jwt-app-normal .
```


## command to run the keploy tests

```
keploy test -c "docker run --name my-app --network keploy-network -p 8080:8080 go-jwt-app-normal" --container-name=my-app --freezeTime
```