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
from py_yt import Playlist
import json
import traceback
from aiohttp_socks import ProxyConnector

from anony import config, logger
from anony.helpers import Track, utils

# New headers matching user's JS configuration
HEADERS = {
    'accept': '*/*',
    'accept-language': 'en-GB,en-US;q=0.9,en;q=0.8',
    'content-type': 'application/json',
    'Referer': 'https://hansaka1-ytdl.hf.space/',
    'User-Agent': 'Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36'
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
        url = f"https://www.youtube.com/results?search_query={quote(query)}"
        headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
            'Accept-Language': 'en-US,en;q=0.9',
        }
        
        connector = None
        
        if config.PROXY:
            # Build URL string from the config dict
            _schema = config.PROXY.get("scheme", "http")
            _user = config.PROXY.get("username", "")
            _pass = config.PROXY.get("password", "")
            _host = config.PROXY.get("hostname", "")
            _port = config.PROXY.get("port", "")
            
            auth = f"{_user}:{_pass}@" if _user and _pass else ""
            full_proxy_url = f"{_schema}://{auth}{_host}:{_port}"
            
            # Use ProxyConnector for both HTTP and SOCKS proxies uniformly
            connector = ProxyConnector.from_url(full_proxy_url)
        
        try:
            # Pass connector for the proxy. Disable SSL to prevent certificate errors on HF proxy
            async with aiohttp.ClientSession(headers=headers, connector=connector) as session:
                async with session.get(url, timeout=15, ssl=False) as resp:
                    text = await resp.text()

                    # YouTube stores the initial data in a javascript variable called ytInitialData
                    match = re.search(r'var ytInitialData = ({.*?});</script>', text)
                    if not match:
                        logger.error("YouTube Search Failed: Could not find ytInitialData")
                        return None
                        
                    data = json.loads(match.group(1))
                    contents = data['contents']['twoColumnSearchResultsRenderer']['primaryContents']['sectionListRenderer']['contents'][0]['itemSectionRenderer']['contents']
                    
                    for item in contents:
                        if 'videoRenderer' in item:
                            video_ren = item['videoRenderer']
                            video_id = video_ren.get('videoId')
                            if not video_id:
                                continue
                                
                            title = video_ren.get('title', {}).get('runs', [{}])[0].get('text', 'Unknown Title')
                            
                            # Extract duration strings (e.g. "3:45")
                            length_text = video_ren.get('lengthText', {}).get('simpleText', '0:00')
                            
                            # Extract thumbnails
                            thumbnails = video_ren.get('thumbnail', {}).get('thumbnails', [])
                            thumbnail_url = thumbnails[-1]['url'].split("?")[0] if thumbnails else None
                            
                            # Extract channel
                            channel = video_ren.get('ownerText', {}).get('runs', [{}])[0].get('text', 'Unknown Channel')
                            
                            # Extract views
                            views = video_ren.get('viewCountText', {}).get('simpleText', '0 views').split(' ')[0]
                            
                            return Track(
                                id=video_id,
                                channel_name=channel[:25],
                                duration=length_text,
                                duration_sec=utils.to_seconds(length_text),
                                message_id=m_id,
                                title=title[:25],
                                thumbnail=thumbnail_url,
                                url=f"https://www.youtube.com/watch?v={video_id}",
                                view_count=views,
                                video=video,
                            )
            return None
        except Exception as e:
            logger.error(f"Custom YouTube search failed: {type(e).__name__} - {e}\n{traceback.format_exc()}")
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

    # --- New Hansaka API Logic ---

    async def download(self, video_id: str, video: bool = False) -> Optional[str]:
        # Construct the full YouTube URL
        link = self.base + video_id
        
        ext = "mp4" if video else "mp3" 
        filename = f"downloads/{video_id}.{ext}"

        # Return existing file if already downloaded
        if Path(filename).exists():
            return filename

        # API parameters
        api_url = "https://hansaka1-ytdl.hf.space/download"
        payload = {
            "url": link,
            "type": "video" if video else "audio"
        }

        # Download logic using the new Hansaka API (returns raw buffer)
        try:
            # Bypassing proxy for internal HF space communication
            async with aiohttp.ClientSession(headers=HEADERS, trust_env=False) as session:
                async with session.post(api_url, json=payload, timeout=aiohttp.ClientTimeout(total=300)) as resp:
                    if resp.status == 200:
                        with open(filename, "wb") as f:
                            while True:
                                chunk = await resp.content.read(1024 * 1024) # 1MB chunks
                                if not chunk:
                                    break
                                f.write(chunk)
                        
                        if os.path.getsize(filename) > 0:
                            return filename
                        else:
                            logger.error(f"Downloaded file {filename} is empty.")
                            os.remove(filename)
                    else:
                        error_text = await resp.text()
                        logger.error(f"Hansaka API Error {resp.status}: {error_text}")
        except Exception as ex:
            logger.error(f"Hansaka Download failed: {ex}")
            if os.path.exists(filename):
                os.remove(filename)
        
        return None