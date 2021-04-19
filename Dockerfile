# syntax = docker/dockerfile:experimental

#
# ----- Go Builder Image ------
#
FROM golang:1.16-alpine AS builder

# curl git bash
RUN apk add --no-cache curl git bash make
COPY --from=golangci/golangci-lint:v1.24-alpine /usr/bin/golangci-lint /usr/bin

#
# ----- Build and Test Image -----
#
FROM builder as build

ARG target=all

# set working directory
RUN mkdir -p /go/src/kube-secrets-init
WORKDIR /go/src/kube-secrets-init

# load dependency
COPY go.mod .
COPY go.sum .
RUN --mount=type=cache,target=/go/mod go mod download

# copy sources
COPY . .

# build
RUN make ${target}


#
# ------ get latest CA certificates
#
FROM alpine:3.11 as certs
RUN apk --update add ca-certificates


#
# ------ secrets-init release Docker image ------
#
FROM scratch

# copy CA certificates
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# this is the last commabd since it's never cached
COPY --from=build /go/src/kube-secrets-init/.bin/github.com/doitintl/kube-secrets-init /kube-secrets-init

ENTRYPOINT ["/kube-secrets-init"]