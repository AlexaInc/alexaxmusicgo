import asyncio
import os
import httpx
from py_yt import VideosSearch

async def test_search():
    # Use the proxy from the logs
    proxy_url = "http://145.241.223.213:57135"
    os.environ["http_proxy"] = proxy_url
    os.environ["https_proxy"] = proxy_url
    
    print(f"Testing VideosSearch with proxy: {proxy_url}")
    
    try:
        # Test basic httpx request first
        async with httpx.AsyncClient(proxy=proxy_url) as client:
            resp = await client.get("https://www.youtube.com", timeout=10)
            print(f"Basic HTTPX request to YouTube: {resp.status_code}")
    except Exception as e:
        print(f"Basic HTTPX request failed: {e}")

    try:
        # Test py_yt search
        print("Searching via py_yt...")
        _search = VideosSearch("test", limit=1)
        results = await _search.next()
        if results and results.get("result"):
            print("py_yt search success!")
        else:
            print("py_yt search returned no results.")
    except Exception as e:
        print(f"py_yt search failed: {e}")

if __name__ == "__main__":
    asyncio.run(test_search())
