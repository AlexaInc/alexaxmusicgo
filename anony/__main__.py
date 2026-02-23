# Copyright (c) 2025 AnonymousX1025
# Licensed under the MIT License.
# This file is part of AnonXMusic

import asyncio
import importlib
from aiohttp import web # <--- 1. Import web module
from pyrogram import idle

from anony import (anon, app, config, db,
                   logger, stop, userbot, yt)
from anony.plugins import all_modules

# --- 2. Define the Fake Health Route ---
async def health_check(request):
    return web.Response(text="Alive")

async def start_health_server():
    # Create a simple web app
    server = web.Application()
    server.router.add_get("/", health_check)
    server.router.add_get("/health", health_check)
    
    # Create the runner
    runner = web.AppRunner(server)
    await runner.setup()
    
    # Listen on port (Hugging Face Default is 7860)
    site = web.TCPSite(runner, "0.0.0.0", config.PORT)
    await site.start()
    logger.info(f"Health Server started on port {config.PORT}")
# ---------------------------------------

async def main():
    await db.connect()
    await app.boot()
    await userbot.boot()
    await anon.boot()

    # --- 3. Start the Fake Server ---

    # --------------------------------

    for module in all_modules:
        importlib.import_module(f"anony.plugins.{module}")
    logger.info(f"Loaded {len(all_modules)} modules.")

    if config.COOKIES_URL:
        await yt.save_cookies(config.COOKIES_URL)

    sudoers = await db.get_sudoers()
    app.sudoers.update(sudoers)
    app.bl_users.update(await db.get_blacklisted())
    logger.info(f"Loaded {len(app.sudoers)} sudo users.")

    await idle()
    await stop()


if __name__ == "__main__":
    try:
        asyncio.get_event_loop().run_until_complete(main())
    except KeyboardInterrupt:
        pass