FROM golang:alpine AS builder

WORKDIR /build

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-w -s" -o server .

FROM alpine:latest

WORKDIR /application

COPY --from=builder /build/server .

HEALTHCHECK NONE

CMD ["./server"]