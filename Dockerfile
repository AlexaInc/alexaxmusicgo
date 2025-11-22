FROM python:3.11-slim-bookworm

# 1. Set Working Directory
WORKDIR /app

# 2. Install System Dependencies
RUN apt-get update -y && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends ffmpeg curl unzip git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 3. Install Deno (Global Install)
# We move it to /usr/local/bin so the standard user can run it
RUN curl -fsSL https://deno.land/install.sh | sh \
    && mv /root/.deno/bin/deno /usr/local/bin/deno \
    && chmod 755 /usr/local/bin/deno

# 4. Create Choreo User (ID 10014)
# This satisfies the CKV_CHOREO_1 security requirement
RUN useradd -m -u 10014 choreouser

# 5. Copy all files
COPY . .

# 6. Install Python Requirements
RUN pip3 install -U pip && pip3 install -U -r requirements.txt

# 7. Force Python to look in /app
ENV PYTHONPATH="/app:$PYTHONPATH"

# ----------------------------------------------------------------------
# CRITICAL FIX: READ-ONLY FILE SYSTEM
# Choreo does not allow writing to /app/log.txt.
# We modify the code during build to write logs to /tmp/log.txt instead.
# ----------------------------------------------------------------------
RUN sed -i 's/"log.txt"/"\/tmp\/log.txt"/g' anony/__init__.py

# 8. Grant Permissions
# We give ownership to user 10014 (Best practice, even on RO systems)
RUN chown -R 10014:10014 /app

# 9. Switch to the Secure User
USER 10014

# 10. Start the Bot Directly
# No .env generation needed. It reads directly from Choreo Settings.
CMD ["python3", "-m", "anony"]