    async def _get_download_url(self, link: str, video: bool) -> Optional[str]:
        # Encode URL to handle special characters safely
        encoded_link = quote(link, safe='')
        
        async with aiohttp.ClientSession(headers=HEADERS) as session:
            # 1. Try Okatsu API
            try:
                if video:
                    api_url = f"https://okatsu-rolezapiiz.vercel.app/downloader/ytmp4?url={encoded_link}"
                    data = await self._fetch_json(session, api_url)
                    if data and data.get("result", {}).get("mp4"):
                        return data["result"]["mp4"]
                else:
                    api_url = f"https://okatsu-rolezapiiz.vercel.app/downloader/ytmp3?url={encoded_link}"
                    data = await self._fetch_json(session, api_url)
                    if data and data.get("dl"):
                        return data["dl"]
            except Exception as e:
                logger.warning(f"Okatsu API failed: {e}")

            # 2. Try Izumi API (Fallback)
            try:
                logger.info("⚠️ Okatsu failed, trying Izumi fallback...")
                fmt = "720" if video else "mp3"
                api_url = f"https://izumiiiiiiii.dpdns.org/downloader/youtube?url={encoded_link}&format={fmt}"
                data = await self._fetch_json(session, api_url)
                
                if data and data.get("result", {}).get("download"):
                    return data["result"]["download"]
            except Exception as e:
                logger.warning(f"Izumi API failed: {e}")
            
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