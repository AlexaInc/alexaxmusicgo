// Package bot wraps gogram client setup and helper utilities for handlers.
package bot

import (
	"fmt"
	"log"
	"strings"

	"alexamusic/internal/config"
	"alexamusic/internal/db"
	"alexamusic/internal/lang"

	"github.com/amarnathcjd/gogram/telegram"
)

// Bot is the main bot client with helpers.
type Bot struct {
	*telegram.Client
	Config  *config.Config
	SudoIDs map[int64]bool
	BLUsers map[int64]bool
}

var B *Bot

func New(cfg *config.Config) (*Bot, error) {
	client, err := telegram.NewClient(telegram.ClientConfig{
		AppID:   int32(cfg.APIID),
		AppHash: cfg.APIHash,
		Session: "alexa_bot",
	})
	if err != nil {
		return nil, fmt.Errorf("bot client: %w", err)
	}

	if err := client.LoginBot(cfg.BotToken); err != nil {
		return nil, fmt.Errorf("bot token login: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("bot client: %w", err)
	}
	b := &Bot{
		Client:  client,
		Config:  cfg,
		SudoIDs: make(map[int64]bool),
		BLUsers: make(map[int64]bool),
	}
	B = b
	return b, nil
}

func (b *Bot) Boot() error {
	if err := b.Client.Start(); err != nil {
		return err
	}
	me, err := b.Client.GetMe()
	if err != nil {
		return err
	}
	log.Printf("[bot] Started as @%s", me.Username)

	// Test logger access
	_, err = b.Client.SendMessage(b.Config.LoggerID, "✅ Bot Started – AlexaMusic Go")
	if err != nil {
		return fmt.Errorf("cannot access logger: %w", err)
	}

	// Load sudoers and BL users
	for _, id := range db.DB.GetSudoers() {
		b.SudoIDs[id] = true
	}
	for _, id := range db.DB.Blacklisted {
		b.BLUsers[id] = true
	}
	return nil
}

func (b *Bot) IsSudo(userID int64) bool {
	return userID == b.Config.OwnerID || b.SudoIDs[userID]
}

func (b *Bot) IsBlacklisted(id int64) bool {
	return b.BLUsers[id] || db.DB.IsBlacklisted(id)
}

// GetLang returns the LangMap for a given chat.
func GetLang(chatID int64) lang.LangMap {
	code := db.DB.GetLang(chatID)
	return lang.M.Get(code)
}

// InlineKeyboardRow builds a Telegram inline keyboard row.
func InlineKeyboardButton(text, data string) *telegram.KeyboardButtonCallback {
	return &telegram.KeyboardButtonCallback{Text: text, Data: []byte(data)}
}

func InlineURLButton(text, url string) *telegram.KeyboardButtonURL {
	return &telegram.KeyboardButtonURL{Text: text, URL: url}
}

// Markup builds an InlineKeyboardMarkup from rows.
func Markup(rows ...[]telegram.KeyboardButton) *telegram.ReplyInlineMarkup {
	var kbRows []*telegram.KeyboardButtonRow
	for _, row := range rows {
		kbRows = append(kbRows, &telegram.KeyboardButtonRow{Buttons: row})
	}
	return &telegram.ReplyInlineMarkup{Rows: kbRows}
}

// ControlsMarkup returns the standard music playback controls keyboard.
func ControlsMarkup(chatID int64) *telegram.ReplyInlineMarkup {
	id := fmt.Sprintf("%d", chatID)
	return Markup(
		[]telegram.KeyboardButton{
			InlineKeyboardButton("▷", "controls resume "+id),
			InlineKeyboardButton("II", "controls pause "+id),
			InlineKeyboardButton("⥁", "controls replay "+id),
			InlineKeyboardButton("‣‣I", "controls skip "+id),
			InlineKeyboardButton("▢", "controls stop "+id),
		},
	)
}

// HelpMarkup builds the help menu buttons.
func HelpMarkup(lm lang.LangMap, back bool) *telegram.ReplyInlineMarkup {
	if back {
		return Markup([]telegram.KeyboardButton{
			InlineKeyboardButton(lm.Get("back"), "help back"),
			InlineKeyboardButton(lm.Get("close"), "help close"),
		})
	}
	categories := []struct{ key, cb string }{
		{"help_0", "help admins"}, {"help_1", "help auth"}, {"help_2", "help blist"},
		{"help_3", "help lang"}, {"help_4", "help ping"}, {"help_5", "help play"},
		{"help_6", "help queue"}, {"help_7", "help stats"}, {"help_8", "help sudo"},
	}
	var btns []telegram.KeyboardButton
	for _, c := range categories {
		btns = append(btns, InlineKeyboardButton(lm.Get(c.key), c.cb))
	}
	var rows [][]telegram.KeyboardButton
	for i := 0; i < len(btns); i += 3 {
		end := i + 3
		if end > len(btns) {
			end = len(btns)
		}
		rows = append(rows, btns[i:end])
	}
	return Markup(rows...)
}

// StartMarkup returns the /start keyboard.
func StartMarkup(lm lang.LangMap, username string, private bool, cfg *config.Config) *telegram.ReplyInlineMarkup {
	rows := [][]telegram.KeyboardButton{
		{InlineURLButton(lm.Get("add_me"), "https://t.me/"+username+"?startgroup=true")},
		{InlineKeyboardButton(lm.Get("help"), "help")},
		{
			InlineURLButton(lm.Get("support"), cfg.SupportChat),
			InlineURLButton(lm.Get("channel"), cfg.SupportChannel),
		},
	}
	if private {
		rows = append(rows, []telegram.KeyboardButton{
			InlineURLButton(lm.Get("source"), "https://github.com/AnonymousX1025/AnonXMusic"),
		})
	} else {
		rows = append(rows, []telegram.KeyboardButton{
			InlineKeyboardButton(lm.Get("language"), "language"),
		})
	}
	return Markup(rows...)
}

// LangMarkup returns the language selection keyboard.
func LangMarkup(currentCode string) *telegram.ReplyInlineMarkup {
	langs := lang.M.GetLanguages()
	var btns []telegram.KeyboardButton
	for code, name := range langs {
		label := fmt.Sprintf("%s (%s)", name, code)
		if code == currentCode {
			label += " ✔️"
		}
		btns = append(btns, InlineKeyboardButton(label, "lang_change "+code))
	}
	var rows [][]telegram.KeyboardButton
	for i := 0; i < len(btns); i += 2 {
		end := i + 2
		if end > len(btns) {
			end = len(btns)
		}
		rows = append(rows, btns[i:end])
	}
	return Markup(rows...)
}

// SettingsMarkup returns the chat settings keyboard.
func SettingsMarkup(lm lang.LangMap, adminOnly, cmdDelete bool, language string, chatID int64) *telegram.ReplyInlineMarkup {
	adminVal := "Off"
	if adminOnly {
		adminVal = "On"
	}
	deleteVal := "Off"
	if cmdDelete {
		deleteVal = "On"
	}
	return Markup(
		[]telegram.KeyboardButton{
			InlineKeyboardButton(lm.Get("play_mode")+" ➜", "settings"),
			InlineKeyboardButton(adminVal, "settings play"),
		},
		[]telegram.KeyboardButton{
			InlineKeyboardButton(lm.Get("cmd_delete")+" ➜", "settings"),
			InlineKeyboardButton(deleteVal, "settings delete"),
		},
		[]telegram.KeyboardButton{
			InlineKeyboardButton(lm.Get("language")+" ➜", "settings"),
			InlineKeyboardButton(language, "language"),
		},
	)
}

// QueuedMarkup returns the "Play Now" queued keyboard.
func QueuedMarkup(chatID int64, itemID, playNowText string) *telegram.ReplyInlineMarkup {
	return Markup([]telegram.KeyboardButton{
		InlineKeyboardButton(playNowText, fmt.Sprintf("controls force %d %s", chatID, itemID)),
	})
}

// SendMessage is a helper for bot.Client.SendMessage.
func (b *Bot) SendMessage(chatID int64, text string) (*telegram.NewMessage, error) {
	return b.Client.SendMessage(chatID, text)
}

// EditMessageReplyMarkup is a helper for editing only the reply markup of a message.
func (b *Bot) EditMessageReplyMarkup(chatID int64, messageID int32, markup *telegram.ReplyInlineMarkup) (*telegram.NewMessage, error) {
	return b.Client.EditMessage(chatID, messageID, "", &telegram.SendOptions{ReplyMarkup: markup})
}

// ForwardMessage is a helper for forwarding a single message.
func (b *Bot) ForwardMessage(toID, fromID int64, messageID int32) (telegram.Updates, error) {
	fromPeer, err := b.Client.ResolvePeer(fromID)
	if err != nil {
		return nil, err
	}
	toPeer, err := b.Client.ResolvePeer(toID)
	if err != nil {
		return nil, err
	}
	return b.Client.MessagesForwardMessages(&telegram.MessagesForwardMessagesParams{
		FromPeer: fromPeer,
		ToPeer:   toPeer,
		ID:       []int32{messageID},
		RandomID: []int64{telegram.GenRandInt()},
	})
}

// ParseCommand extracts the command name and arguments from a message text.
func ParseCommand(text string) (cmd string, args []string) {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) == 0 {
		return "", nil
	}
	raw := parts[0]
	if strings.HasPrefix(raw, "/") {
		raw = raw[1:]
	}
	// Strip @BotName suffix
	if idx := strings.Index(raw, "@"); idx != -1 {
		raw = raw[:idx]
	}
	return strings.ToLower(raw), parts[1:]
}
