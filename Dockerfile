# Build
# $ docker build .
# Use the offical golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.19-buster as builder

# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.* ./
RUN go mod download

# Copy local code to the container image.
COPY . ./

# Build the binary.
RUN go build -v -o sdr-framerelay-tcp

# Deploy
#
# Use the official Debian slim image for a lean production container.
# https://hub.docker.com/_/debian
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/sdr-framerelay-tcp /app/sdr-framerelay-tcp

# 
# Usage of ./sdr-framerelay-tcp:
#   -compress string
#     	what end of transport will be compressed/decompress. Possible options: 'decode' on last hop, 'encode' on first hop, and 'no' (default "no")
#   -connect string
#     	connect to IP:Port. (default "127.0.0.1:9002")
#   -listen string
#     	listen IP:Port. (default "0.0.0.0:9001")
#   -speed string
#     	The compressing level. Options: Fastest (lvl 1), Default (lvl 3), Better (lvl 7), Best (lvl 11) (default "Default")
# 
# $ docker run sdr-framerelay-tcp -connect endpointIP:PORT 
#
# Run the service on container startup.
ENTRYPOINT ["/app/sdr-framerelay-tcp"]