#!/bin/bash

mkdir _build

set -e

echo "Building client..."; go build -o _build/gostreaming ../../gostreaming/*.go
echo "Building num_gen for alpine..."; env GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o _build/num_gen ./num_gen/*.go
echo "Building filter for alpine..."; env GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o _build/filter ./filter/*.go
echo "Building printer for alpine..."; env GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o _build/printer ./printer/*.go

echo "Uploading binary actions..."
_build/gostreaming 127.0.0.1:5555 actions new -n num_gen -f _build/num_gen
_build/gostreaming 127.0.0.1:5555 actions new -n filter -f _build/filter
_build/gostreaming 127.0.0.1:5555 actions new -n printer -f _build/printer

echo "Uploading schema..."
_build/gostreaming 127.0.0.1:5555 schemas new -f simplepipe.yaml

echo "Starting pipeline..."
_build/gostreaming 127.0.0.1:5555 schemas run -n simplepipe

echo "Pipeline started!"
echo "Check: http://127.0.0.1:5555/v1/schemas/simplepipe/dashboard?send_period=3s to see debug graph"
echo "Run '_build/gostreaming 127.0.0.1:5555 schemas stop -n simplepipe' to stop pipeline"
