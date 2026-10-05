FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api \
  && CGO_ENABLED=0 GOOS=linux go build -o /out/cron ./cmd/cron \
  && CGO_ENABLED=0 GOOS=linux go build -o /out/cli ./cmd/cli
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.1

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl \
  && adduser -D -u 10001 app \
  && mkdir -p /data/uploads /app/logs \
  && chown -R app /data /app
WORKDIR /app
COPY --from=build /out/api /out/cron /out/cli /app/
COPY --from=build /go/bin/goose /app/goose
COPY migrations /app/migrations
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh
USER app
ENV UPLOAD_FOLDER=/data/uploads \
    LOG_OUTPUT=stdout \
    LOG_FORMAT=json
EXPOSE 8181
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD curl -fsS http://localhost:${PORT:-8181}/health || exit 1
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/api"]
