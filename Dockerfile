FROM python:3.11-slim-bookworm

# 1. Set Working Directory
WORKDIR /app

# 2. Install System Dependencies
RUN apt-get update -y && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends ffmpeg curl unzip git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 3. Install Deno (Global Install Fix)
# We move the binary to /usr/local/bin so the non-root user can access it
RUN curl -fsSL https://deno.land/install.sh | sh \
    && mv /root/.deno/bin/deno /usr/local/bin/deno \
    && chmod 755 /usr/local/bin/deno

# 4. Create a Non-Root User (Fixes CKV_DOCKER_3)
RUN useradd -m -u 1000 user

# 5. Copy all files
COPY . .

# 6. Install Python Requirements
RUN pip3 install -U pip && pip3 install -U -r requirements.txt

# 7. Force Python to look in /app
ENV PYTHONPATH="/app:$PYTHONPATH"

# 8. Create the Start Script
# We write this as root first, then we will give permission to the user
RUN echo "#!/bin/bash" > run_bot.sh && \
    echo "echo 'Building .env file...'" >> run_bot.sh && \
    # echo "echo \"API_ID=\${API_ID}\" > /app/.env" >> run_bot.sh && \
    # echo "echo \"API_HASH=\${API_HASH}\" >> /app/.env" >> run_bot.sh && \
    # echo "echo \"BOT_TOKEN=\${BOT_TOKEN}\" >> /app/.env" >> run_bot.sh && \
    # echo "echo \"MONGO_URL=\${MONGO_URL}\" >> /app/.env" >> run_bot.sh && \
    # echo "echo \"OWNER_ID=\${OWNER_ID}\" >> /app/.env" >> run_bot.sh && \
    # echo "echo \"SESSION=\${SESSION}\" >> /app/.env" >> run_bot.sh && \
    # echo "echo \"LOGGER_ID=\${LOGGER_ID}\" >> /app/.env" >> run_bot.sh && \
    echo "echo 'Starting Bot...'" >> run_bot.sh && \
    echo "python3 -m anony" >> run_bot.sh && \
    chmod +x run_bot.sh

# 9. GRANT PERMISSIONS (Critical Step)
# Give the new user ownership of the /app folder so it can write the .env file
RUN chown -R user:user /app

# 10. Switch to Non-Root User
USER user

# 11. Run the script
CMD ["bash", "run_bot.sh"]