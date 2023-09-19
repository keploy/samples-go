# Use the Go 1.19 base image with the bullseye tag
FROM golang:1.20-bookworm


RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* 

# Set the working directory inside the container
WORKDIR /app

COPY go.mod /app/
COPY go.sum /app/

RUN go mod download

# Copy the contents from the local directory to the working directory in the container
COPY . /app

RUN go build -o app .

EXPOSE 8080

# Run the application server using "go run main.go"
CMD ["./app"]