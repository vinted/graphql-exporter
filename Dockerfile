############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder
# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git
WORKDIR $GOPATH/src/github.com/ubbleai/graphql-exporter/
COPY . .
# Fetch dependencies.
# Using go get.
RUN go get ./...
ARG TARGETOS
ARG TARGETARCH
# Build the binary.
RUN CGO_ENABLED=0   go build -a -o /go/bin/graphql-exporter ./cmd/graphql-exporter
############################
# STEP 2 build a small image
############################
FROM scratch
# Copy our static executable.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/graphql-exporter /go/bin/graphql-exporter
COPY config_example.yaml /config.yaml
EXPOSE 9353
# Run the hello binary.
ENTRYPOINT ["/go/bin/graphql-exporter"]
CMD ["-config_path", "/config.yaml"]