# Build Stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Clone the repository using init+checkout to handle non-empty dir
RUN git init . && \
    git remote add origin https://github.com/AlexaInc/alexaxmusicgo.git && \
    git fetch origin master && \
    git checkout master -f

# Build the binary
RUN go build -o alexa_music .

# Final Stage
FROM alpine:latest

# Install runtime dependencies (ffmpeg for audio/video processing)
RUN apk add --no-cache ffmpeg ca-certificates curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/alexa_music .

# Copy assets
COPY --from=builder /app/assets ./assets

# Expose port (default 7860 for Hugging Face)
EXPOSE 7860

# Metadata
ENV PORT=7860

# Start command
CMD ["./alexa_music"]
