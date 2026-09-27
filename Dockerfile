 # ---- build stage ----
FROM golang:1.27-alpine AS build

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# modernc/glebarez sqlite is pure Go, so CGO can stay disabled.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/todos-api .

# ---- run stage ----
FROM alpine:3.20

WORKDIR /app
COPY --from=build /out/todos-api /app/todos-api

# Persist the SQLite file outside the container.
ENV PORT=8000 \
    DB_DSN=/data/todos.db
VOLUME ["/data"]
EXPOSE 8000

ENTRYPOINT ["/app/todos-api"]
