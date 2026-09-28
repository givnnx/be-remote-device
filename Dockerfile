# Build stage
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Multi-arch cross compilation (e.g. amd64 -> arm64 or native arm64 on Orange Pi 5)
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /app/server ./cmd/api

# Run stage
FROM alpine:3.20

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/server /app/server

EXPOSE 8080

CMD ["/app/server"]
