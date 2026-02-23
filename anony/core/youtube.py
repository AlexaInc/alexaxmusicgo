# Copyright (c) 2025 AnonymousX1025
# Licensed under the MIT License.
# This file is part of AnonXMusic

import os
import re
import asyncio
import aiohttp
from urllib.parse import quote
from pathlib import Path
from typing import Optional, Union

from pyrogram import enums, types
from py_yt import Playlist, VideosSearch

from anony import logger
from anony.helpers import Track, utils

# Configuration matching the JS axios defaults
HEADERS = {
    'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
    'Origin': 'https://frame.y2meta-uk.com',
    'Referer': 'https://frame.y2meta-uk.com/',
    'Accept': 'application/json, text/plain, */*'
}

class YouTube:
    def __init__(self):
        self.base = "https://www.youtube.com/watch?v="
        self.regex = re.compile(
            r"(https?://)?(www\.|m\.|music\.)?"
            r"(youtube\.com/(watch\?v=|shorts/|playlist\?list=)|youtu\.be/)"
            r"([A-Za-z0-9_-]{11}|PL[A-Za-z0-9_-]+)([&?][^\s]*)?"
        )

    def valid(self, url: str) -> bool:
        return bool(re.match(self.regex, url))

    def url(self, message_1: types.Message) -> Union[str, None]:
        messages = [message_1]
        link = None
        if message_1.reply_to_message:
            messages.append(message_1.reply_to_message)

        for message in messages:
            text = message.text or message.caption or ""

            if message.entities:
                for entity in message.entities:
                    if entity.type == enums.MessageEntityType.URL:
                        link = text[entity.offset : entity.offset + entity.length]
                        break

            if message.caption_entities:
                for entity in message.caption_entities:
                    if entity.type == enums.MessageEntityType.TEXT_LINK:
                        link = entity.url
                        break

        if link:
            return link.split("&si")[0].split("?si")[0]
        return None

    async def search(self, query: str, m_id: int, video: bool = False) -> Track | None:
        _search = VideosSearch(query, limit=1)
        results = await _search.next()
        if results and results["result"]:
            data = results["result"][0]
            return Track(
                id=data.get("id"),
                channel_name=data.get("channel", {}).get("name"),
                duration=data.get("duration"),
                duration_sec=utils.to_seconds(data.get("duration")),
                message_id=m_id,
                title=data.get("title")[:25],
                thumbnail=data.get("thumbnails", [{}])[-1].get("url").split("?")[0],
                url=data.get("link"),
                view_count=data.get("viewCount", {}).get("short"),
                video=video,
            )
        return None

    async def playlist(self, limit: int, user: str, url: str, video: bool) -> list[Track | None]:
        tracks = []
        try:
            plist = await Playlist.get(url)
            for data in plist["videos"][:limit]:
                track = Track(
                    id=data.get("id"),
                    channel_name=data.get("channel", {}).get("name", ""),
                    duration=data.get("duration"),
                    duration_sec=utils.to_seconds(data.get("duration")),
                    title=data.get("title")[:25],
                    thumbnail=data.get("thumbnails")[-1].get("url").split("?")[0],
                    url=data.get("link").split("&list=")[0],
                    user=user,
                    view_count="",
                    video=video,
                )
                tracks.append(track)
        except:
            pass
        return tracks

    # --- API Logic (CNV.cx Implementation) ---

    async def _try_request(self, session: aiohttp.ClientSession, method: str, url: str, **kwargs):
        attempts = 3
        for attempt in range(1, attempts + 1):
            try:
                async with session.request(method, url, timeout=aiohttp.ClientTimeout(total=60), **kwargs) as response:
                    if response.status == 200:
                        return await response.json()
            except Exception as e:
                if attempt == attempts:
                    logger.warning(f"Request to {url} failed after {attempts} attempts: {e}")
                else:
                    await asyncio.sleep(attempt)
        return None

    async def _get_cnv_key(self, session: aiohttp.ClientSession) -> Optional[str]:
        data = await self._try_request(session, "GET", "https://cnv.cx/v2/sanity/key")
        return data.get("key") if data else None

    async def _cnv_converter(self, link: str, video: bool) -> Optional[str]:
        async with aiohttp.ClientSession(headers=HEADERS, trust_env=False) as session:
            api_key = await self._get_cnv_key(session)
            if not api_key:
                logger.error("CNV: Could not fetch API Key")
                return None

            fmt = "mp4" if video else "mp3"
            quality = "360" if video else "64"
            
            payload = {
                "link": link,
                "format": fmt,
                "audioBitrate": "64" if video else quality,
                "videoQuality": quality if video else "360",
                "filenameStyle": "pretty",
                "vCodec": "h264"
            }

            headers = {**HEADERS, "key": api_key}
            data = await self._try_request(session, "POST", "https://cnv.cx/v2/converter", data=payload, headers=headers)
            
            if data and data.get("url"):
                return data["url"]
            
            logger.error(f"CNV: Conversion returned no URL. Response: {data}")
            return None

    async def _get_download_url(self, link: str, video: bool) -> Optional[str]:
        # 1. Primary: CNV.cx (User Requested Method)
        direct_url = await self._cnv_converter(link, video)
        if direct_url:
            return direct_url
            
        return None

    async def download(self, video_id: str, video: bool = False) -> Optional[str]:
        # Construct the full YouTube URL
        link = self.base + video_id
        
        # Determine extension based on type
        # Note: APIs typically return mp3 for audio, whereas yt-dlp defaulted to webm
        # Both are valid for ffmpeg/pyrogram to upload/stream.
        ext = "mp4" if video else "mp3" 
        filename = f"downloads/{video_id}.{ext}"

        # Return existing file if already downloaded
        if Path(filename).exists():
            return filename

        # Get direct URL via APIs
        direct_url = await self._get_download_url(link, video)
        
        if not direct_url:
            logger.error(f"Failed to get download link for {video_id} from all APIs.")
            return None

        # Download the actual file buffer to local disk
        try:
            async with aiohttp.ClientSession(headers=HEADERS) as session:
                async with session.get(direct_url) as resp:
                    if resp.status == 200:
                        with open(filename, "wb") as f:
                            while True:
                                chunk = await resp.content.read(1024 * 1024) # 1MB chunks
                                if not chunk:
                                    break
                                f.write(chunk)
                        return filename
                    else:
                        logger.error(f"Failed to download file content: Status {resp.status}")
        except Exception as ex:
            logger.error(f"Download IO failed: {ex}")
            if os.path.exists(filename):
                os.remove(filename)
        
        return None