FROM alpine:latest

# Add CA certificates and tzdata for proper TLS and timezone support
RUN apk --no-cache add ca-certificates tzdata

# Create a non-privileged user to run the application
RUN adduser -D caddydnssync

# Create app directory and assign ownership
WORKDIR /app

# Copy the binary from the build
COPY caddy-dns-sync /app/caddy-dns-sync

# Set executable permissions
RUN chmod +x /app/caddy-dns-sync

# Use the non-privileged user
USER caddydnssync

# Set the binary as the entrypoint
ENTRYPOINT ["/app/caddy-dns-sync"]

# Default command
CMD ["--help"]
