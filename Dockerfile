# Dockerfile
# Build stage
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}" -o /mocksms ./cmd/mocksms

# Final stage
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /mocksms /mocksms

EXPOSE 4010 1025

ENTRYPOINT ["/mocksms", "serve"]