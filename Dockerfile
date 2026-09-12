FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o echos ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/echos /usr/local/bin/echos
ENTRYPOINT ["echos"]
