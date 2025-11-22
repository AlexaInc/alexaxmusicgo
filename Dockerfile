FROM python:3.11-slim-bookworm

# 1. Set Working Directory
WORKDIR /app

# 2. Install System Dependencies
RUN apt-get update -y && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends ffmpeg curl unzip git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 3. Install Deno (Global Install)
# Move binary to /usr/local/bin so the new user can access it
RUN curl -fsSL https://deno.land/install.sh | sh \
    && mv /root/.deno/bin/deno /usr/local/bin/deno \
    && chmod 755 /usr/local/bin/deno

# ----------------------------------------------------------------------
# FIXING CKV_CHOREO_1
# We create a user with UID 10014 (Must be between 10000 - 20000)
# ----------------------------------------------------------------------
RUN useradd -m -u 10014 choreouser

# 4. Copy all files
COPY . .

# 5. Install Python Requirements
RUN pip3 install -U pip && pip3 install -U -r requirements.txt

# 6. Force Python to look in /app
ENV PYTHONPATH="/app:$PYTHONPATH"

# 7. Create the Start Script
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

# 8. GRANT PERMISSIONS (Critical Step)
# We give ownership of /app to UID 10014 so it can write the .env file
RUN chown -R 10014:10014 /app

# 9. SWITCH USER (Passes the Check)
USER 10014

# 10. Run the script
CMD ["bash", "run_bot.sh"]