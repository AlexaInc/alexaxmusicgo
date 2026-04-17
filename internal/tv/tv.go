package tv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"alexamusic/internal/bot"

	"github.com/amarnathcjd/gogram/telegram"
)

// Channel represents a TV station from channels.json.
type Channel struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Category  string `json:"category"`
	Manifest  string `json:"manifest"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

var channels []Channel

// Load loads channels from the JSON file.
func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &channels)
}

// GetCategories returns a sorted list of unique categories.
func GetCategories() []string {
	catMap := make(map[string]bool)
	for _, c := range channels {
		if c.Category != "" {
			catMap[c.Category] = true
		}
	}
	cats := make([]string, 0, len(catMap))
	for k := range catMap {
		cats = append(cats, k)
	}
	sort.Strings(cats)
	return cats
}

// GetByCategory returns channels for a specific category.
func GetByCategory(cat string) []Channel {
	var out []Channel
	for _, c := range channels {
		if c.Category == cat {
			out = append(out, c)
		}
	}
	return out
}

// GetByID returns a channel by its ID.
func GetByID(id string) *Channel {
	for _, c := range channels {
		if c.ID == id {
			return &c
		}
	}
	return nil
}

// CategoryMarkup builds the main TV category keyboard.
func CategoryMarkup() *telegram.ReplyInlineMarkup {
	cats := GetCategories()
	var buttons []telegram.KeyboardButton
	for _, cat := range cats {
		buttons = append(buttons, bot.InlineKeyboardButton(cat, "tv_cat:"+cat))
	}
	var rows [][]telegram.KeyboardButton
	for i := 0; i < len(buttons); i += 3 {
		end := i + 3
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, buttons[i:end])
	}
	rows = append(rows, []telegram.KeyboardButton{
		bot.InlineKeyboardButton("❌ Close Menu", "help close"),
	})
	return bot.Markup(rows...)
}

// ChannelMarkup builds a paged keyboard for a category.
func ChannelMarkup(category string, page int) *telegram.ReplyInlineMarkup {
	list := GetByCategory(category)
	perPage := 10
	start := (page - 1) * perPage
	end := start + perPage
	if end > len(list) {
		end = len(list)
	}

	var rows [][]telegram.KeyboardButton
	for i := start; i < end; i += 2 {
		row := []telegram.KeyboardButton{
			bot.InlineKeyboardButton("📺 "+list[i].Title, "tv_ch:"+list[i].ID),
		}
		if i+1 < end {
			row = append(row, bot.InlineKeyboardButton("📺 "+list[i+1].Title, "tv_ch:"+list[i+1].ID))
		}
		rows = append(rows, row)
	}

	// Navigation
	var nav []telegram.KeyboardButton
	if page > 1 {
		nav = append(nav, bot.InlineKeyboardButton("⬅️ Back", fmt.Sprintf("tv_page:%s:%d", category, page-1)))
	}
	if end < len(list) {
		nav = append(nav, bot.InlineKeyboardButton("Next ➡️", fmt.Sprintf("tv_page:%s:%d", category, page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}

	rows = append(rows, []telegram.KeyboardButton{
		bot.InlineKeyboardButton("🔙 Back to Categories", "tv_home"),
	})
	return bot.Markup(rows...)
}

// FetchStreamURL gets the real stream URL from the viu.lk manifest endpoint.
func FetchStreamURL(manifestURL string, proxyURL string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	if proxyURL != "" {
		// handle proxy if needed, but for now simple direct
	}
	resp, err := client.Get(manifestURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		Status string `json:"status"`
		Data   struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.Status == "ok" && data.Data.URL != "" {
		return data.Data.URL, nil
	}
	return "", fmt.Errorf("manifest status not ok: %s", data.Status)
}
