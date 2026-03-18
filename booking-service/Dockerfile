FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git ca-certificates
COPY . /src
WORKDIR /src/booking-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -o /server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /server .
COPY --from=builder /src/booking-service/migrations ./migrations

EXPOSE 8080
CMD ["/app/server"]