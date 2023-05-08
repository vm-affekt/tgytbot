FROM golang:1.19-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o tgytbot /build/cmd/tgytbot
FROM alpine:latest
RUN apk add --no-cache ffmpeg
WORKDIR /tgytbot
COPY --from=builder /build/tgytbot ./tgytbot
CMD ["./tgytbot"]