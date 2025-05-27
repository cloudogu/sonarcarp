FROM golang:1.24.3 AS builder
WORKDIR app
copy go.* .
COPY *.go .
COPY build /build

RUN make vendor compile-generic