FROM golang:1.22

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
ADD https://keploy-enterprise.s3.us-west-2.amazonaws.com/releases/latest/assets/go_freeze_time_amd64 /lib/keploy/go_freeze_time_amd64

#set suitable permissions
RUN chmod +x /lib/keploy/go_freeze_time_amd64

# run the binary
RUN /lib/keploy/go_freeze_time_amd64
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -tags=faketime -o /jwt-go

EXPOSE 8000


# Run
CMD ["/jwt-go"]