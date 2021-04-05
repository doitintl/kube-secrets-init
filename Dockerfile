FROM golang:1.15-alpine as builder

WORKDIR /app

COPY . .

RUN go mod download
RUN cd cmd/secrets-init-webhook && go build -o /app .

FROM alpine:latest

COPY --from=builder /app/secrets-init-webhook /kube-secrets-init

RUN chmod +x /kube-secrets-init

ENTRYPOINT ["/kube-secrets-init"]
