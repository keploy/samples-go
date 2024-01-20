# === Build Stage ===
FROM golang:1.20-bookworm AS build-stage

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download


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
