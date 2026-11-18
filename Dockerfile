# Dockerfile
# Build stage
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /data-empty

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" -o /mocksms ./cmd/mocksms

# Final stage
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /mocksms /mocksms

# Writable data dir for the nonroot user (numeric chown needs no shell).
COPY --from=builder --chown=65532:65532 /data-empty /data

# Container-appropriate bind defaults: published ports forward to the
# container IP, never its loopback. Host allow-list checks still apply.
ENV MOCKSMS_HTTP_HOST=0.0.0.0 MOCKSMS_SMTP_HOST=0.0.0.0

EXPOSE 4010 1025

ENTRYPOINT ["/mocksms", "serve"]