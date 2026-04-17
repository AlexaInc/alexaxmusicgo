# Build Stage - Use Debian (glibc) to be compatible with libntgcalls.so
FROM golang:1.24-bookworm AS builder

ENV GOTOOLCHAIN=auto

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git build-essential gcc g++ libstdc++6 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Clone the repository
RUN git init . && \
    git remote add origin https://github.com/AlexaInc/alexaxmusicgo.git && \
    git fetch origin master && \
    git checkout master -f

# Copy shared libraries to system path so linker can find them
RUN cp /app/vendor_src/tgcalls/libntgcalls.so /usr/lib/x86_64-linux-gnu/libntgcalls.so && \
    ldconfig

# Build the binary
RUN go build -o alexa_music .

# Final Stage - Debian slim for glibc runtime compatibility
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg ca-certificates curl \
    libglib2.0-0 libx11-6 libxrandr2 libxcomposite1 \
    libxdamage1 libxext6 libxfixes3 libxtst6 libxrender1 \
    libdrm2 libgbm1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/alexa_music .

# Copy shared library
COPY --from=builder /app/vendor_src/tgcalls/libntgcalls.so /usr/lib/x86_64-linux-gnu/libntgcalls.so

# Copy assets
COPY --from=builder /app/assets ./assets

# Expose port (default 7860 for Hugging Face)
EXPOSE 7860

ENV PORT=7860
ENV LD_LIBRARY_PATH=/usr/lib/x86_64-linux-gnu:$LD_LIBRARY_PATH

# Start command
CMD ["./alexa_music"]
