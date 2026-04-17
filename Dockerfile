# Build Stage
FROM golang:1.24-bookworm AS builder

ENV GOTOOLCHAIN=auto

# Install build dependencies INCLUDING dev libraries that libntgcalls.so links against
RUN apt-get update && apt-get install -y --no-install-recommends \
    git build-essential gcc g++ \
    libx11-dev libxrandr-dev libxcomposite-dev libxdamage-dev \
    libxext-dev libxfixes-dev libxtst-dev libxrender-dev \
    libglib2.0-dev libgbm-dev libdrm-dev \
    libgio2.0-cil-dev libdbus-1-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Clone the repository
RUN git init . && \
    git remote add origin https://github.com/AlexaInc/alexaxmusicgo.git && \
    git fetch origin master && \
    git checkout master -f

# Make the linker find libntgcalls.so in the vendor directory
ENV CGO_LDFLAGS="-L/app/vendor_src/tgcalls -lntgcalls -Wl,-rpath=/usr/lib -Wl,--allow-shlib-undefined"
RUN ldconfig

# Build the binary
RUN go build -o alexa_music .

# Final Stage
FROM debian:bookworm-slim

# Install runtime libraries
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg ca-certificates curl \
    libx11-6 libxrandr2 libxcomposite1 libxdamage1 \
    libxext6 libxfixes3 libxtst6 libxrender1 \
    libglib2.0-0 libgbm1 libdrm2 \
    libdbus-1-3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary
COPY --from=builder /app/alexa_music .

# Copy shared library
COPY --from=builder /app/vendor_src/tgcalls/libntgcalls.so /usr/lib/x86_64-linux-gnu/libntgcalls.so
RUN ldconfig

# Copy assets
COPY --from=builder /app/assets ./assets

EXPOSE 7860
ENV PORT=7860

CMD ["./alexa_music"]
