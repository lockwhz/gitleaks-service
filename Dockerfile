# Etapa de build
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git wget ca-certificates
WORKDIR /app
COPY go.mod .
COPY . .
RUN go mod download
RUN go build -o clone-scan main.go

# Etapa final
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/clone-scan /usr/local/bin/clone-scan
RUN wget -q -O /usr/local/bin/gitleaks https://github.com/gitleaks/gitleaks/releases/latest/download/gitleaks_linux_x64 \
    && chmod +x /usr/local/bin/gitleaks

ENV GITLEAKS_PATH="/usr/local/bin/gitleaks"

ENTRYPOINT ["/usr/local/bin/clone-scan"]
