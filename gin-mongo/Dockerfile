# === Build Stage ===
FROM golang:1.22-bookworm AS build-stage

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done


# Copy the contents of the current directory into the build container
COPY *.go ./

# Build the binary
RUN CGO_ENABLED=0 go build -o /main 

# === Runtime Stage ===
FROM alpine:3.19

# Add groups and user to run the app and install dumb-init package
RUN addgroup -S keploy \ 
&& adduser -S keploy -G keploy -h /home/keploy \
&& mkdir -p /home/keploy/app \
&& chown -R keploy:keploy /home/keploy/app \ 
&& apk add --no-cache dumb-init \ 
&& rm -rf /var/cache/apk/*

# Change working directory
WORKDIR /home/keploy/app

# Copy the binary from build-stage and paste it in app directory
COPY --from=build-stage --chown=keploy:keploy /main /home/keploy/app/main 

# Set the entrypoint
ENTRYPOINT ["dumb-init"]

# Set user
USER keploy

# Expose the port
EXPOSE 8080

# Run the binary
CMD ["./main"]
