FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=1.0.0" \
    -o mocktool \
    .

# Stage 2: Create minimal runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -g 1000 mocktool && \
    adduser -D -u 1000 -G mocktool mocktool
WORKDIR /home/mocktool
COPY --from=builder /app/mocktool .
COPY --from=builder /app/web ./web
RUN chown -R mocktool:mocktool /home/mocktool
USER mocktool
EXPOSE 8081 8082
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1
ENTRYPOINT ["./mocktool"]
CMD ["service"]
