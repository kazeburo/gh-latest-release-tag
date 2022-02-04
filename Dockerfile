FROM golang:1.17-alpine as builder

WORKDIR /go/src

COPY go.mod go.sum ./
RUN go mod download

COPY ./main.go  ./

ENV CGO_ENABLED 0
ENV GOOS linux

RUN go build \
    -o /go/bin/main \
    -ldflags '-s -w'

FROM alpine:latest as runner

COPY --from=builder /go/bin/main /app/main

ENTRYPOINT ["/app/main"]