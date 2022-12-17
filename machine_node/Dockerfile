FROM golang:1.18.8-alpine3.15

# The latest alpine images don't have some tools like (`git` and `bash`).
# Adding git, bash and openssh to the image
RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh gcc musl-dev && \
    apk add iptables

LABEL maintainer="Daniil Gavrilovsky"

WORKDIR /gostreaming

# Set depends
COPY go.mod go.sum ./
RUN go mod download
COPY util ./util
COPY machine_node ./machine_node
COPY runtime ./runtime

#create binary
RUN go build -o ./bin/runtime ./runtime/*.go
RUN go build -o ./bin/machine_node ./machine_node/*.go
RUN chmod +x ./bin/machine_node

CMD ["/bin/sh"]
