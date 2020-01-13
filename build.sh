#!/bin/sh

echo "Compiling binaries..."
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo main.go
docker-compose build
docker-compose up