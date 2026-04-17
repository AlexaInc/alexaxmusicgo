package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all environment-based configuration for AlexaMusic bot.
type Config struct {
	Port       int
	APIID      int
	APIHash    string
	BotToken   string
	MongoURL   string

	LoggerID int64
	OwnerID  int64

	DurationLimit  int // seconds
	QueueLimit     int
	PlaylistLimit  int

	Session1 string
	Session2 string
	Session3 string

	SupportChannel string
	SupportChat    string

	AutoEnd    bool
	AutoLeave  bool
	VideoPlay  bool
	CookiesURL []string

	DefaultThumb string
	PingImg      string
	StartImg     string
	MusicBotName string

	ProxyURL string

	// Version string
	Version string
}

var C *Config

func Load() *Config {
	_ = godotenv.Load()

	c := &Config{
		Port:           envInt("PORT", 7860),
		APIID:          envInt("API_ID", 0),
		APIHash:        envStr("API_HASH", ""),
		BotToken:       envStr("BOT_TOKEN", ""),
		MongoURL:       envStr("MONGO_URL", ""),
		LoggerID:       int64(envInt("LOGGER_ID", 0)),
		OwnerID:        int64(envInt("OWNER_ID", 0)),
		DurationLimit:  envInt("DURATION_LIMIT", 60) * 60,
		QueueLimit:     envInt("QUEUE_LIMIT", 20),
		PlaylistLimit:  envInt("PLAYLIST_LIMIT", 20),
		Session1:       envStr("SESSION", ""),
		Session2:       envStr("SESSION2", ""),
		Session3:       envStr("SESSION3", ""),
		SupportChannel: envStr("SUPPORT_CHANNEL", "https://t.me/AlexaInc_updates"),
		SupportChat:    envStr("SUPPORT_CHAT", "https://t.me/+_9LokVOOdrdlOGQ1"),
		AutoEnd:        envBool("AUTO_END", false),
		AutoLeave:      envBool("AUTO_LEAVE", false),
		VideoPlay:      envBool("VIDEO_PLAY", true),
		DefaultThumb:   envStr("DEFAULT_THUMB", "https://te.legra.ph/file/3e40a408286d4eda24191.jpg"),
		PingImg:        envStr("PING_IMG", "https://files.catbox.moe/haagg2.png"),
		StartImg:       envStr("START_IMG", "https://files.catbox.moe/zvziwk.jpg"),
		MusicBotName:   envStr("MUSIC_BOT_NAME", "Alexa Music"),
		ProxyURL:       strings.Trim(envStr("PROXY_URL", ""), "'\""),
		Version:        "3.0.1-go",
	}

	// Parse cookies URLs (only batbin.me links)
	for _, u := range strings.Split(envStr("COOKIES_URL", ""), " ") {
		if u != "" && strings.Contains(u, "batbin.me") {
			c.CookiesURL = append(c.CookiesURL, u)
		}
	}

	C = c
	return c
}

func (c *Config) Check() {
	missing := []string{}
	if c.APIID == 0 {
		missing = append(missing, "API_ID")
	}
	if c.APIHash == "" {
		missing = append(missing, "API_HASH")
	}
	if c.BotToken == "" {
		missing = append(missing, "BOT_TOKEN")
	}
	if c.MongoURL == "" {
		missing = append(missing, "MONGO_URL")
	}
	if c.LoggerID == 0 {
		missing = append(missing, "LOGGER_ID")
	}
	if c.OwnerID == 0 {
		missing = append(missing, "OWNER_ID")
	}
	if c.Session1 == "" {
		missing = append(missing, "SESSION")
	}
	if len(missing) > 0 {
		log.Fatalf("[config] Missing required environment variables: %s", strings.Join(missing, ", "))
	}
}

// --- helpers ---

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}
