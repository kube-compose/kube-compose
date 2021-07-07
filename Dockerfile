FROM golang:1.12
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go run .
