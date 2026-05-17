# builder
FROM golang:1.26-alpine AS builder
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/openscout ./cmd/openscout

# runtime
FROM alpine:3.23
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/openscout /bin/openscout
WORKDIR /app
ENV DATABASE_URL=postgres://openscout:openscoutpass@db:5432/openscout?sslmode=disable
EXPOSE 8080
ENTRYPOINT ["/bin/openscout"]