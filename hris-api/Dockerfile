FROM golang:1.26.6-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api && CGO_ENABLED=0 go build -o /seed ./cmd/seed
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D app
WORKDIR /app
COPY --from=build /api /seed ./
USER app
EXPOSE 8088
CMD ["./api"]
