ARG GO_VERSION=1.13

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --update --no-cache ca-certificates git curl

ARG GOPROXY=https://proxy.golang.org

RUN mkdir -p /build
WORKDIR /build

COPY go.* /build/
RUN go mod download

COPY . /build
RUN CGO_ENABLED=0 go install ./cmd/aws-secrets-webhook

FROM alpine:3.10

RUN apk add --update libcap && rm -rf /var/cache/apk/*

COPY --from=builder /go/bin/aws-secrets-webhook /usr/local/bin/aws-secrets-webhook
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV DEBUG false
USER 65534

ENTRYPOINT ["/usr/local/bin/aws-secrets-webhook"]
