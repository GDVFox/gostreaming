#!/bin/bash

mkdir _build

set -e

echo "Building client..."; go build -o _build/gostreaming ../../gostreaming/*.go
echo "Building tcs-account tool..."; go build -o _build/tcs-account ./tcs-account/*.go
echo "Building stocks-gen for alpine..."; env GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o _build/stocks-gen ./stocks-gen/*.go
echo "Building stocks-portfolio for alpine..."; env GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o _build/stocks-portfolio ./stocks-portfolio/*.go
echo "Building stocks-db for alpine..."; env GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o _build/stocks-db ./stocks-db/*.go

echo "Uploading binary actions..."
_build/gostreaming 127.0.0.1:5555 actions new -n stocks-gen -f _build/stocks-gen
_build/gostreaming 127.0.0.1:5555 actions new -n stocks-portfolio -f _build/stocks-portfolio
_build/gostreaming 127.0.0.1:5555 actions new -n stocks-db -f _build/stocks-db

echo "Uploading schema..."
_build/gostreaming 127.0.0.1:5555 schemas new -f stocks.yaml

echo "Starting pipeline..."
_build/gostreaming 127.0.0.1:5555 schemas run -n stocks

echo "Pipeline started!"
echo "Check: http://127.0.0.1:5555/v1/schemas/stocks/dashboard?send_period=3s to see debug graph"
echo "Run '_build/gostreaming 127.0.0.1:5555 schemas stop -n stocks' to stop pipeline"
