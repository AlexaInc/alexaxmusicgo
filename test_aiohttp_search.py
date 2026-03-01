import asyncio
import os
import aiohttp
import re

async def test_aiohttp_search():
    proxy_url = "http://145.241.223.213:57135"
    os.environ["http_proxy"] = proxy_url
    os.environ["https_proxy"] = proxy_url
    
    print(f"Testing aiohttp search with proxy: {proxy_url}")
    
    try:
        query = "test"
        url = f"https://www.youtube.com/results?search_query={query}"
        
        async with aiohttp.ClientSession(trust_env=True) as session:
            # aiohttp automatically uses HTTP_PROXY / HTTPS_PROXY env vars if trust_env=True
            async with session.get(url, timeout=10) as resp:
                print(f"Status: {resp.status}")
                text = await resp.text()
                
                # Basic regex to find the first video ID
                match = re.search(r'"videoId":"([a-zA-Z0-9_-]{11})"', text)
                if match:
                    print(f"Success! Found video ID: {match.group(1)}")
                else:
                    print("Failed to find video ID in response.")
    except Exception as e:
        print(f"aiohttp request failed: {e}")

if __name__ == "__main__":
    asyncio.run(test_aiohttp_search())
