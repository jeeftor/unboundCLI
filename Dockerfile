FROM alpine:latest

# Add CA certificates and tzdata for proper TLS and timezone support
RUN apk --no-cache add ca-certificates tzdata

# Create a non-privileged user to run the application
RUN adduser -D unboundcli

# Create app directory and assign ownership
WORKDIR /app

# Copy the binary from the build
COPY unboundCLI /app/unboundCLI

# Set executable permissions
RUN chmod +x /app/unboundCLI

# Use the non-privileged user
USER unboundcli

# Set the binary as the entrypoint
ENTRYPOINT ["/app/unboundCLI"]

# Default command
CMD ["--help"]
