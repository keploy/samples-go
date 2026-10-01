FROM golang:1.22

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
# proxy.golang.org now and then resets a download mid-transfer ("stream error:
# ... INTERNAL_ERROR"), which fails an un-retried fetch and the whole build.
# Retried: 5 attempts, 5-20s apart, then fail.
RUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
ADD https://keployenterprise.blob.core.windows.net/releases/latest/assets/go_freeze_time_arm64 /lib/keploy/go_freeze_time_arm64

#set suitable permissions
RUN chmod +x /lib/keploy/go_freeze_time_arm64

# run the binary
RUN /lib/keploy/go_freeze_time_arm64
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /jwt-go

EXPOSE 8000


# Run
CMD ["/jwt-go"]