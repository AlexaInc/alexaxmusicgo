import asyncio
import aiohttp
import re
import json
import os

async def search_youtube(query: str, limit: int = 1):
    url = f"https://www.youtube.com/results?search_query={query}"
    
    # Optional: Set Headers if needed to simulate browser
    headers = {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        'Accept-Language': 'en-US,en;q=0.9',
    }
    
    proxy_url = "http://145.241.223.213:57135"  # Harcoded for testing

    async with aiohttp.ClientSession(headers=headers) as session:
        async with session.get(url, proxy=proxy_url, timeout=15) as resp:
            text = await resp.text()

            # YouTube stores the initial data in a javascript variable called ytInitialData
            match = re.search(r'var ytInitialData = ({.*?});</script>', text)
            if not match:
                print("Failed to find ytInitialData")
                return []
                
            try:
                data = json.loads(match.group(1))
                contents = data['contents']['twoColumnSearchResultsRenderer']['primaryContents']['sectionListRenderer']['contents'][0]['itemSectionRenderer']['contents']
                
                results = []
                for item in contents:
                    if limit <= 0:
                        break
                    
                    if 'videoRenderer' in item:
                        video = item['videoRenderer']
                        video_id = video.get('videoId')
                        title = video.get('title', {}).get('runs', [{}])[0].get('text', 'No Title')
                        
                        # Extract duration strings (e.g. "3:45")
                        length_text = video.get('lengthText', {}).get('simpleText', '0:00')
                        
                        # Extract thumbnails
                        thumbnails = video.get('thumbnail', {}).get('thumbnails', [])
                        thumbnail_url = thumbnails[-1]['url'] if thumbnails else None
                        
                        # Extract channel
                        channel = video.get('ownerText', {}).get('runs', [{}])[0].get('text', 'Unknown Channel')
                        
                        # Extract views
                        views = video.get('viewCountText', {}).get('simpleText', '0 views').split(' ')[0]
                        
                        results.append({
                            "id": video_id,
                            "title": title,
                            "duration": length_text,
                            "thumbnail": thumbnail_url,
                            "channel": channel,
                            "views": views,
                            "link": f"https://www.youtube.com/watch?v={video_id}"
                        })
                        limit -= 1
                        
                return results
                
            except Exception as e:
                print(f"Error parsing JSON: {e}")
                return []

async def test():
    results = await search_youtube("Lana Del Rey Summertime Sadness")
    for r in results:
        print(f"[{r['duration']}] {r['title']} - {r['channel']} ({r['views']} views)\nURL: {r['link']}\nThumb: {r['thumbnail']}\n")

if __name__ == "__main__":
    asyncio.run(test())
