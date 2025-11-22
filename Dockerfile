FROM python:3.11-slim-bookworm

# 1. Set Working Directory
WORKDIR /app

# 2. Install System Dependencies
RUN apt-get update -y && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends ffmpeg curl unzip git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 3. Install Deno
RUN curl -fsSL https://deno.land/install.sh | sh \
    && ln -s /root/.deno/bin/deno /usr/local/bin/deno

# 4. Copy all files to /app
COPY . .

# 5. Install Python Requirements
RUN pip3 install -U pip && pip3 install -U -r requirements.txt

# 6. Force Python to look in /app for the 'anony' folder
ENV PYTHONPATH="/app:$PYTHONPATH"

# 7. START SCRIPT (Explicit /app/.env path)
RUN echo "#!/bin/bash" > run_bot.sh && \
    echo "echo 'Building .env file in /app...'" >> run_bot.sh && \
    # We write specifically to /app/.env to be safe
    echo "echo \"API_ID=\${API_ID}\" > /app/.env" >> run_bot.sh && \
    echo "echo \"API_HASH=\${API_HASH}\" >> /app/.env" >> run_bot.sh && \
    echo "echo \"BOT_TOKEN=\${BOT_TOKEN}\" >> /app/.env" >> run_bot.sh && \
    echo "echo \"MONGO_URL=\${MONGO_URL}\" >> /app/.env" >> run_bot.sh && \
    echo "echo \"OWNER_ID=\${OWNER_ID}\" >> /app/.env" >> run_bot.sh && \
    echo "echo \"SESSION=\${SESSION}\" >> /app/.env" >> run_bot.sh && \
    echo "echo \"LOGGER_ID=\${LOGGER_ID}\" >> /app/.env" >> run_bot.sh && \
    echo "echo 'Starting Bot...'" >> run_bot.sh && \
    echo "python3 -m anony" >> run_bot.sh && \
    chmod +x run_bot.sh

# 8. Run the script
CMD ["bash", "run_bot.sh"]
