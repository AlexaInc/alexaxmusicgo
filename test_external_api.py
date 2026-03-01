import asyncio
import aiohttp
import json

async def test_search(query: str):
    url = "https://search.nnmn.store/"
    
    headers = {
        "accept": "*/*",
        "accept-language": "en-GB,en-US;q=0.9,en;q=0.8",
        "origin": "https://v6.www-y2mate.com",
        "referer": "https://v6.www-y2mate.com/",
        "user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36"
    }

    form_data = aiohttp.FormData()
    form_data.add_field("search_query", query)

    try:
        async with aiohttp.ClientSession(headers=headers) as session:
            async with session.post(url, data=form_data, timeout=15) as resp:
                print(f"Status: {resp.status}")
                if resp.status == 200:
                    data = await resp.json()
                    print(json.dumps(data, indent=2))
                else:
                    text = await resp.text()
                    print(text)
    except Exception as e:
        print(f"API Error: {e}")

if __name__ == "__main__":
    asyncio.run(test_search("dana danath"))
