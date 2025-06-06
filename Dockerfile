FROM golang:alpine

ADD . /app
WORKDIR /app
RUN go build
RUN go test -v ./mpegts_parser