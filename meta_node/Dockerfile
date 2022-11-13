FROM golang:1.16.4-alpine3.13

# The latest alpine images don't have some tools like (`git` and `bash`).
# Adding git, bash and openssh to the image
RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh gcc musl-dev

LABEL maintainer="Daniil Gavrilovsky"

WORKDIR /gostreaming

# Set depends
COPY go.mod go.sum ./
RUN go mod download
COPY util ./util
COPY meta_node ./meta_node

#create binary
RUN go build -o ./bin/meta_node ./meta_node/*.go
RUN chmod +x ./bin/meta_node

EXPOSE 8888

CMD ["/bin/sh"]
