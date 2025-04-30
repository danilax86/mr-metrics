# SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
#
# SPDX-License-Identifier: MIT

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o mr-metrics ./cmd/app

# Final stage
FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/mr-metrics .
COPY internal/web/templates ./web/templates
COPY internal/web/style.css ./web/style.css
COPY migrations ./migrations

EXPOSE 8080
ENTRYPOINT ["./mr-metrics"]
