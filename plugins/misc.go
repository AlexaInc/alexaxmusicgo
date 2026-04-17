package plugins

import (
	"context"
	"fmt"
	"log"
	"time"

	"alexamusic/internal/bot"
	"alexamusic/internal/calls"
	"alexamusic/internal/db"
	"alexamusic/internal/queue"
	"alexamusic/internal/youtube"

	"github.com/amarnathcjd/gogram/telegram"
)

// StartWorkers launches all background goroutines.
func StartWorkers(ctx context.Context, b *bot.Bot, y *youtube.YouTube, q *queue.Manager) {
	go trackTime(ctx, q)
	go updateTimer(ctx, b, y, q)
}

// trackTime increments track playback time every second.
func trackTime(ctx context.Context, q *queue.Manager) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for chatID, state := range db.DB.ActiveCalls {
				if state == 1 {
					q.IncrementTime(chatID)
				}
			}
		}
	}
}

// updateTimer updates the playback progress bar in the playing message every 7 s.
func updateTimer(ctx context.Context, b *bot.Bot, y *youtube.YouTube, q *queue.Manager) {
	ticker := time.NewTicker(7 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for chatID, state := range db.DB.ActiveCalls {
				if state != 1 {
					continue
				}
				track := q.GetCurrent(chatID)
				if track == nil || track.DurationSec == 0 || track.MessageID == 0 {
					continue
				}
				played := track.Time
				remaining := track.DurationSec - played
				if remaining < 0 {
					continue
				}

				const length = 10
				pos := played * length / track.DurationSec
				if pos >= length {
					pos = length - 1
				}

				var timerStr string
				remove := remaining < 10
				if !remove {
					bar := repeatStr("—", pos) + "◉" + repeatStr("—", length-pos-1)
					timerStr = fmt.Sprintf("%s | %s | -%s",
						formatTime(played), bar, formatTime(remaining))
				} else {
					timerStr = "◉"
				}

				// Pre-fetch next track
				if remaining <= 30 {
					next := q.GetNext(chatID, true)
					if next != nil && next.FilePath == "" {
						go func(n *queue.Track) {
							fp, err := y.Download(n.ID, n.Video)
							if err == nil {
								n.FilePath = fp
							}
						}(next)
					}
				}

				// Update the message markup with the timer
				lm := bot.GetLang(chatID)
				status := timerStr
				if remove {
					status = lm.Get("stopped")
				}
				markup := timerMarkup(chatID, status, remove, lm)
				_, _ = b.EditMessageReplyMarkup(chatID, int32(track.MessageID), markup)
			}
		}
	}
}

func timerMarkup(chatID int64, status string, remove bool, lm interface{ Get(string, ...interface{}) string }) *telegram.ReplyInlineMarkup {
	id := fmt.Sprintf("%d", chatID)
	rows := [][]telegram.KeyboardButton{
		{bot.InlineKeyboardButton(status, "controls status "+id)},
	}
	if !remove {
		rows = append(rows, []telegram.KeyboardButton{
			bot.InlineKeyboardButton("▷", "controls resume "+id),
			bot.InlineKeyboardButton("II", "controls pause "+id),
			bot.InlineKeyboardButton("⥁", "controls replay "+id),
			bot.InlineKeyboardButton("‣‣I", "controls skip "+id),
			bot.InlineKeyboardButton("▢", "controls stop "+id),
		})
	}
	return bot.Markup(rows...)
}

func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func formatTime(secs int) string {
	m, s := secs/60, secs%60
	return fmt.Sprintf("%02d:%02d", m, s)
}

// VCWatcher stops empty voice chats after 30 s of no participants.
func VCWatcher(ctx context.Context, b *bot.Bot, q *queue.Manager) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for chatID := range db.DB.ActiveCalls {
				cur := q.GetCurrent(chatID)
				if cur == nil || cur.Time < 30 {
					continue
				}
				lm := bot.GetLang(chatID)
				calls.E.Stop(chatID)
				_, _ = b.SendMessage(chatID, lm.Get("auto_left"))
				log.Printf("[misc] Auto-left empty VC in %d", chatID)
			}
		}
	}
}
