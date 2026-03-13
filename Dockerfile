# Stage 1: Build the tika-mcp binary
FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o tika-mcp .

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /src/tika-mcp .

# Default: connect to Tika running on the host (override via env or flag)
ENV TIKA_URL=http://tika:9998
ENV MCP_TRANSPORT=stdio

ENTRYPOINT ["/app/tika-mcp"]
CMD []
