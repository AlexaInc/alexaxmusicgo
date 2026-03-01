import asyncio
from ytmusicapi import YTMusic
import os

async def test_search():
    # Try testing with proxies defined
    proxy_url = "http://145.241.223.213:57135"
    os.environ["HTTP_PROXY"] = proxy_url
    os.environ["HTTPS_PROXY"] = proxy_url
    
    print(f"Testing ytmusicapi with proxy: {proxy_url}")
    try:
        # Initialize YTMusic without authentication (public searches)
        ytmusic = YTMusic()
        
        # We need "songs" or "videos"
        results = ytmusic.search("Lana Del Rey Summertime Sadness", filter="songs", limit=1)
        
        if results:
            item = results[0]
            print("SUCCESS!")
            print(f"Title: {item.get('title')}")
            print(f"Video ID: {item.get('videoId')}")
            print(f"Duration: {item.get('duration')}s")
            
            artists = item.get('artists', [])
            artist_name = artists[0]['name'] if artists else "Unknown"
            print(f"Artist: {artist_name}")
            
        else:
            print("Found no results.")
            
    except Exception as e:
        print(f"ytmusicapi Failed: {type(e).__name__} - {e}")

if __name__ == "__main__":
    asyncio.run(test_search())
