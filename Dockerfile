# syntax = docker/dockerfile:experimental

#
# ----- Go Builder Image ------
#
FROM --platform=${BUILDPLATFORM} golang:1.21-alpine AS builder

# curl git bash
RUN apk add --no-cache curl git bash make tzdata ca-certificates
COPY --from=golangci/golangci-lint:v1.54-alpine /usr/bin/golangci-lint /usr/bin

#
# ----- Build and Test Image -----
#
FROM builder as build

# passed by buildkit
ARG TARGETOS
ARG TARGETARCH

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
RUN --mount=type=cache,target=/root/.cache/go-build TARGETOS=${TARGETOS} TARGETARCH=${TARGETARCH} make

#
# ------ secrets-init release Docker image ------
#
FROM scratch

# copy certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# copy timezone settings
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# this is the last commabd since it's never cached
COPY --from=build /go/src/kube-secrets-init/.bin/github.com/doitintl/kube-secrets-init /kube-secrets-init

ENTRYPOINT ["/kube-secrets-init"]