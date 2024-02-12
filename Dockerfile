FROM golang:1.22.0 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main . && \
    useradd -u 10001 appuser

FROM scratch
COPY --from=builder /src/main /app/main
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /etc/passwd /etc/passwd

USER appuser
WORKDIR /app
ENTRYPOINT [ "/app/main" ]
