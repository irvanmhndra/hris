FROM golang:1.26.6-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate \
 && CGO_ENABLED=0 go build -o /out/seed ./cmd/seed
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D app \
 && mkdir -p /data/uploads && chown app:app /data/uploads
WORKDIR /app
COPY --from=build /out/ ./
USER app
EXPOSE 8088
CMD ["./api"]
