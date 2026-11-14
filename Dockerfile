FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder

ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

RUN apt-get update && apt-get install -y gcc-aarch64-linux-gnu gcc
RUN go install github.com/evanw/esbuild/cmd/esbuild@latest

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN esbuild index.jsx --bundle --minify --outfile=assets/bundle.js --jsx=transform --jsx-factory=React.createElement --jsx-fragment=React.Fragment
RUN CGO_ENABLED=1 CC=$([ "$TARGETARCH" = "arm64" ] && echo "aarch64-linux-gnu-gcc" || echo "gcc") GOOS=$TARGETOS GOARCH=$TARGETARCH go build --tags "fts5" -v -o ./squad .

FROM --platform=$TARGETPLATFORM debian:bookworm-slim

# Bots call OpenRouter and the tool APIs over HTTPS, and schedules need time zone data for TZ.
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/squad /squad

VOLUME /data

ENV DATA_FOLDER=/data

CMD ["/squad"]
