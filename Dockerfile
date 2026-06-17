FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /monolith ./cmd/monolith/

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache wget
COPY --from=builder /monolith .
COPY web/ web/
COPY migrations/ migrations/
EXPOSE 8080
CMD ["/app/monolith"]
