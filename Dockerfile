FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

WORKDIR /app

COPY src/go.mod src/go.sum ./

COPY src/ ./

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o ua3f

FROM --platform=$BUILDPLATFORM alpine

WORKDIR /app

COPY --from=builder /app/ua3f .

EXPOSE 1080

ENTRYPOINT ["/app/ua3f", "-b", "0.0.0.0"]