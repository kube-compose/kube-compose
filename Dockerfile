FROM golang:alpine AS builder
RUN apk --no-cache add git
WORKDIR /app/src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
RUN ls -la && \
  CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main .

FROM alpine:latest
COPY --from=builder /app/main /app/main
ENTRYPOINT ["/app/main"]