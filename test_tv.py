import asyncio
import os
import traceback
import httpx

# Assume load_channels works if imported
from anony.helpers.tv import fetch_stream_url, load_channels

# Reuse the bot's proxy for test
proxy_url = "http://145.241.223.213:443"
os.environ["http_proxy"] = proxy_url
os.environ["https_proxy"] = proxy_url

async def test_fetch():
    channels = load_channels()
    target = next((c for c in channels if c["id"] == "hirutv"), None)
    
    # Custom fetch to see exactly what fails
    manifest_url = target["manifest"]
    print(f"Fetching from: {manifest_url}")
    
    async with httpx.AsyncClient(proxy=proxy_url) as client:
        try:
            response = await client.get(manifest_url)
            print(f"Status Code: {response.status_code}")
            print(f"Response Headers: {response.headers}")
            print(f"Response Body: {response.text}")
        except Exception as e:
            print(f"Exception: {e}")
            traceback.print_exc()

if __name__ == "__main__":
    asyncio.run(test_fetch())
