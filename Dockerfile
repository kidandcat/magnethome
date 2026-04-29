# syntax=docker/dockerfile:1.6

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY server/go.mod server/go.sum ./server/
RUN cd server && go mod download
COPY server/ ./server/
RUN cd server && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/mh-server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1000 app
WORKDIR /app
COPY --from=build /out/mh-server /app/mh-server
# Landing site assets — everything except the server source and infra files.
COPY index.html ./
COPY css ./css
COPY js ./js
COPY img ./img
COPY fonts ./fonts
COPY CNAME ./
RUN mkdir -p /data && chown -R app:app /data /app
USER app
ENV HTTP_ADDR=:8080 \
    DATA_DIR=/data \
    RAFT_BIND=127.0.0.1:9000 \
    LANDING_DIR=/app
EXPOSE 8080
CMD ["/app/mh-server"]
