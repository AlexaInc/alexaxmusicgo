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
    && mv /root/.deno/bin/deno /usr/local/bin/deno \
    && chmod 755 /usr/local/bin/deno

# 4. Create User (ID 10014)
RUN useradd -m -u 10014 choreouser

# 5. Copy Files
COPY . .

# 6. Install Python Requirements
RUN pip3 install -U pip && pip3 install -U -r requirements.txt

# 7. Force Python to look in /app
ENV PYTHONPATH="/app:$PYTHONPATH"

# ----------------------------------------------------------------------
# FIX 1: LOG FILE HACK (For Read-Only Systems)
# ----------------------------------------------------------------------
RUN sed -i 's/"log.txt"/"\/tmp\/log.txt"/g' anony/__init__.py

# ----------------------------------------------------------------------
# FIX 2: EXPOSE A PORT (Satisfies Back4App Error)
# ----------------------------------------------------------------------
EXPOSE 8080

# 8. Grant Permissions
RUN chown -R 10014:10014 /app

# 9. Switch User
USER 10014

# 10. Start Bot
CMD ["python3", "-m", "anony"]