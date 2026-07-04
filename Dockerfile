FROM golang:1.22 AS builder

WORKDIR /app

# Install dependencies first (for better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nicovideo_tag_rss main.go

# Use distroless for a minimal and secure runtime image
FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /app/nicovideo_tag_rss /nicovideo_tag_rss

EXPOSE 8080

ENTRYPOINT ["/nicovideo_tag_rss"]
