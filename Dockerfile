ARG BASE_IMAGE=golang:alpine

FROM ${BASE_IMAGE} AS builder

RUN apk --no-cache add git

WORKDIR /app/src

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main .

FROM scratch

WORKDIR /app

COPY --from=builder /app/main .

ENTRYPOINT ["/app/main"]