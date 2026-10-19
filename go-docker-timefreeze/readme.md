---

## Time freezing

The app signs JWTs that expire, so a replay a day later fails unless the
app's clock is frozen at the recording's time. Keploy does that with the
`--freezeTime` flag on `keploy test`. Build and record the image as it is:
there is nothing to add to the Dockerfile and nothing to rebuild.

When you replay, Keploy patches a copy of the image's Go binary (`/main`) so
that it reads the frozen clock, and mounts the copy over the original in the
container. Your image is not changed. Keploy logs a `time-freeze[go]` line
that says whether it patched the binary and, if it could not, why.

This needs a local Docker daemon that shares the host's temporary directory
with Keploy (not a remote daemon, or one in a VM such as Docker Desktop
without file sharing, colima, lima or podman machine). Where that is not
possible, see the fallback, a replay-only build with Keploy's `faketime`
runtime, in the
[time freezing docs](https://keploy.io/docs/keploy-cloud/time-freezing/#fallback-a-replay-only-faketime-build).
Never record with that build.

## command to build docker image

```
docker build -t go-jwt-app-normal .
```

## command to run keploy record

Use the `keploy record` command to capture API calls and generate test cases. While this is running, run the `./test_time_endpoint.sh` script to generate tests.

```
keploy record -c "docker run --name my-app --network keploy-network -p 8080:8080 go-jwt-app-normal" --container-name=my-app
```

## command to run the keploy tests

```
keploy test -c "docker run --name my-app --network keploy-network -p 8080:8080 go-jwt-app-normal" --container-name=my-app --freezeTime
```
