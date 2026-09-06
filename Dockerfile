# syntax=docker/dockerfile:1

# ---- 1. build the Vue frontend ----
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build          # -> /web/dist

# ---- 2. build the Go server (embeds /web/dist) ----
FROM golang:1.25-alpine AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY server/ ./server/
COPY --from=web /web/dist ./server/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/serbian-app ./server

# ---- 3. runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
WORKDIR /app
COPY --from=server /out/serbian-app /app/serbian-app
COPY content/ /app/content/
RUN mkdir -p /app/data && chown -R app:app /app
USER app
EXPOSE 8080
VOLUME ["/app/data"]
ENV TZ=Europe/Belgrade
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["/app/serbian-app"]
CMD ["-addr", ":8080", "-content", "/app/content", "-db", "/app/data/app.db"]
