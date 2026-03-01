import json
import os
import httpx
from pyrogram.types import InlineKeyboardMarkup, InlineKeyboardButton

# Load channels data
CHANNELS_FILE = os.path.join(os.path.dirname(__file__), "channels.json")

def load_channels():
    try:
        with open(CHANNELS_FILE, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception as e:
        print(f"Error loading channels.json: {e}")
        return []

def get_categories(channels=None):
    if channels is None:
        channels = load_channels()
    categories = list(set([c["category"] for c in channels if "category" in c]))
    categories.sort()
    return categories

def get_channels_by_category(category, channels=None):
    if channels is None:
        channels = load_channels()
    return [c for c in channels if c.get("category") == category]

def category_markup():
    categories = get_categories()
    keyboard = []
    row = []
    
    # Create 3-column rows
    for cat in categories:
        row.append(InlineKeyboardButton(cat, callback_data=f"tv_cat:{cat}"))
        if len(row) >= 3:
            keyboard.append(row)
            row = []
    
    if row:
        keyboard.append(row)
        
    keyboard.append([InlineKeyboardButton("❌ Close Play Menu", callback_data="help close")])
    return InlineKeyboardMarkup(keyboard)

def channel_markup(category, page=1):
    channels = get_channels_by_category(category)
    keyboard = []
    
    # 10 channels per page
    items_per_page = 10
    start_idx = (page - 1) * items_per_page
    end_idx = start_idx + items_per_page
    
    current_channels = channels[start_idx:end_idx]

    
    for ch in current_channels:
        # Two columns for channels
        if len(keyboard) > 0 and len(keyboard[-1]) < 2:
            keyboard[-1].append(InlineKeyboardButton(f"📺 {ch['title']}", callback_data=f"tv_ch:{ch['id']}"))
        else:
            keyboard.append([InlineKeyboardButton(f"📺 {ch['title']}", callback_data=f"tv_ch:{ch['id']}")])

            
    # Pagination
    nav_row = []
    if page > 1:
        nav_row.append(InlineKeyboardButton("⬅️ Back", callback_data=f"tv_page:{category}:{page-1}"))
    if end_idx < len(channels):
        nav_row.append(InlineKeyboardButton("Next ➡️", callback_data=f"tv_page:{category}:{page+1}"))
        
    if nav_row:
        keyboard.append(nav_row)
        
    # Back to categories
    keyboard.append([InlineKeyboardButton("🔙 Back to Categories", callback_data="tv_home")])
    return InlineKeyboardMarkup(keyboard)

async def fetch_stream_url(manifest_url):
    """Fetches the actual stream URL from the manifest endpoint."""
    from anony import config
    
    proxy_url = None
    if hasattr(config, "PROXY_URL") and config.PROXY_URL:
        proxy_url = config.PROXY_URL

    async with httpx.AsyncClient(proxy=proxy_url) as client:
        try:
            response = await client.get(manifest_url)
            response.raise_for_status()
            data = response.json()
            if data.get("status") == "ok" and "url" in data.get("data", {}):
                return data["data"]["url"]
            return None
        except Exception as e:
            print(f"Error fetching TV manifest: {e}")
            return None
