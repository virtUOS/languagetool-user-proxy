FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:3.23
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
