# =========================
# BUILD STAGE
# =========================
FROM golang:1.25.3-alpine AS build

WORKDIR /app

RUN apk add --no-cache ca-certificates git

# Сначала зависимости (кэшируется)
COPY go.mod go.sum ./
RUN go mod download

# Потом весь код
COPY . .

# Сборка бинарей

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/publisher ./cmd/publisher
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/consumer ./cmd/consumer
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/replay ./cmd/replay

# =========================
# RUNTIME STAGE
# =========================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates


WORKDIR /app

COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/publisher /usr/local/bin/publisher
COPY --from=build /out/consumer /usr/local/bin/consumer
COPY --from=build /out/replay /usr/local/bin/replay

# По умолчанию запускаем API
CMD ["api"]
