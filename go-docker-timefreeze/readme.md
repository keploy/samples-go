
---

## Time Freeze Agent Setup

### amd64/x86_64 🖥️

```dockerfile
# Download the time freeze agent
ADD https://keploy-enterprise.s3.us-west-2.amazonaws.com/releases/latest/assets/go_freeze_time_amd64 /lib/keploy/go_freeze_time_amd64

# Set suitable permissions
RUN chmod +x /lib/keploy/go_freeze_time_amd64

# Run the binary
RUN /lib/keploy/go_freeze_time_amd64

# Build your binary with fake time (during test mode)
RUN go build -tags=faketime <your_main_file>
```

---

### arm64/aarch64 📱

```dockerfile
# Download the time freeze agent
ADD https://keploy-enterprise.s3.us-west-2.amazonaws.com/releases/latest/assets/go_freeze_time_arm64 /lib/keploy/go_freeze_time_arm64

# Set suitable permissions
RUN chmod +x /lib/keploy/go_freeze_time_arm64

# Run the binary
RUN /lib/keploy/go_freeze_time_arm64

# Build your binary with fake time (during test mode)
RUN go build -tags=faketime <your_main_file>
```

> **Note:** Only add the `faketime` tag to your build script during **Test MODE**.

Re-build your Docker image.

Now add the `--freeze-time` flag when running your tests with Keploy, like so:

```bash
keploy test -c "<appCmd>" --freeze-time
```


## before building, kindly ensure to move the time freezing agent binary to this directory


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
