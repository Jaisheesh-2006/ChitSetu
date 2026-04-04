FROM alpine:3.20 AS builder
WORKDIR /app
COPY README.md /app/README.md

FROM alpine:3.20 AS runner
WORKDIR /app
COPY --from=builder /app/README.md /app/README.md
CMD ["sh", "-c", "echo 'web3 service placeholder container'; sleep 3600"]
