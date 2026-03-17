FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git ca-certificates
COPY . /src
WORKDIR /src/flight-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /server .
COPY --from=builder /src/flight-service/migrations ./migrations

EXPOSE 9090
CMD ["/app/server"]