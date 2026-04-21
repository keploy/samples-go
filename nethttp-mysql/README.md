# nethttp-mysql

Minimal Go sample that uses `net/http` + `database/sql` with the MySQL
driver. The app exposes three endpoints (`/health`, `/users`,
`/users/add`) and connects to a standard MySQL 8 instance.

This sample is wired into Keploy's end-to-end pipeline as a regression
guard for the MySQL replay command-phase loop: in replay mode the app
must successfully complete `db.Ping()` (which sends commands to the
mocked MySQL connection) and bind `:8080`. If the proxy ever re-
introduces a zero/overflow read deadline in the MySQL command phase the
Go driver blocks inside `db.Ping()`, the HTTP listener never starts,
and every replayed request fails with `connection reset by peer` —
making the regression loud.

## Run locally

```sh
# Terminal 1
docker network create keploy-network
docker run -d --name mysql --network keploy-network \
  -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=testdb mysql:8.0

# Terminal 2
docker build -t nethttp-mysql .
docker run --rm -p 8080:8080 --network keploy-network \
  -e MYSQL_HOST=mysql -e MYSQL_USER=root -e MYSQL_PASSWORD=password \
  -e MYSQL_DATABASE=testdb nethttp-mysql
```

Then hit:

```sh
curl http://localhost:8080/health
curl http://localhost:8080/users
curl "http://localhost:8080/users/add?name=charlie&email=charlie@test.com"
```

## Use with Keploy

```sh
keploy record -c "docker run -p 8080:8080 --name nethttp-mysql --network keploy-network \
  -e MYSQL_HOST=mysql -e MYSQL_USER=root -e MYSQL_PASSWORD=password \
  -e MYSQL_DATABASE=testdb nethttp-mysql" \
  --container-name nethttp-mysql --network-name keploy-network --buildDelay 60

# ... exercise the endpoints ...

# Stop MySQL, then replay against mocks:
docker rm -f mysql
keploy test -c "docker run -p 8080:8080 --name nethttp-mysql --network keploy-network \
  -e MYSQL_HOST=mysql -e MYSQL_USER=root -e MYSQL_PASSWORD=password \
  -e MYSQL_DATABASE=testdb nethttp-mysql" \
  --containerName nethttp-mysql --network-name keploy-network \
  --apiTimeout 60 --delay 20
```
