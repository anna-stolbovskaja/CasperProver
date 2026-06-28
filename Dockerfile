FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY engine/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /casperprover ./cmd/casperprover

FROM scratch
COPY --from=builder /casperprover /casperprover
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/casperprover"]
