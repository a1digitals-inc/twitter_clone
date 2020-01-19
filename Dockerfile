# Use the offical Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.12

WORKDIR /go/src/github.com/dustinnewman98/twitter_clone

# Copy local code to the container image.
ADD . /go/src/github.com/dustinnewman98/twitter_clone

RUN go get -d

# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN go install github.com/dustinnewman98/twitter_clone

# Run the outyet command by default when the container starts.
ENTRYPOINT /go/bin/twitter_clone
EXPOSE 8000