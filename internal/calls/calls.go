package calls

import (
	"fmt"
	"log"
	"sync"

	"alexamusic/internal/db"
	"alexamusic/internal/lang"
	"alexamusic/internal/queue"
	"alexamusic/internal/userbot"

	"github.com/amarnathcjd/tgcalls"
	"github.com/amarnathcjd/gogram/telegram"
)

// Engine wraps the tgcalls voice-chat engine.
type Engine struct {
	mu      sync.RWMutex
	clients map[int]*tgcalls.GroupCall // assistant num -> GroupCall client
}

var E *Engine

func New() *Engine {
	e := &Engine{clients: make(map[int]*tgcalls.GroupCall)}
	E = e
	return e
}

// Boot initialises one tgcalls.GroupCall client per userbot session.
func (e *Engine) Boot() error {
	for i, client := range userbot.UB.Clients {
		num := i + 1
		gc := tgcalls.NewGroupCall(client)

		// Handle stream-ended and chat-leave events
		gc.OnStreamEnd(func(chatID int64) {
			log.Printf("[calls] Stream ended in %d, playing next…", chatID)
			E.PlayNext(chatID)
		})

		gc.OnLeave(func(chatID int64) {
			log.Printf("[calls] Left call in %d, stopping…", chatID)
			E.Stop(chatID)
		})

		e.mu.Lock()
		e.clients[num] = gc
		e.mu.Unlock()
	}
	log.Printf("[calls] %d tgcalls client(s) started.", len(e.clients))
	return nil
}

func (e *Engine) getClient(chatID int64) (*tgcalls.GroupCall, error) {
	num := db.DB.GetAssistantNum(chatID)
	e.mu.RLock()
	c, ok := e.clients[num]
	e.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no tgcalls client for assistant %d", num)
	}
	return c, nil
}

// ─── CONTROLS ─────────────────────────────────────────────────────────────────

func (e *Engine) Pause(chatID int64) error {
	c, err := e.getClient(chatID)
	if err != nil {
		return err
	}
	db.DB.Playing(chatID, 0)
	return c.Pause(chatID)
}

func (e *Engine) Resume(chatID int64) error {
	c, err := e.getClient(chatID)
	if err != nil {
		return err
	}
	db.DB.Playing(chatID, 1)
	return c.Resume(chatID)
}

func (e *Engine) Stop(chatID int64) {
	queue.Q.Clear(chatID)
	db.DB.RemoveCall(chatID)
	c, err := e.getClient(chatID)
	if err != nil {
		return
	}
	_ = c.LeaveCall(chatID)
}

// ─── PLAY ─────────────────────────────────────────────────────────────────────

// PlayMedia streams the given track into the group call.
// botClient is used to edit Telegram messages.
func (e *Engine) PlayMedia(botClient *telegram.Client, chatID int64, track *queue.Track, seekSec int) error {
	c, err := e.getClient(chatID)
	if err != nil {
		return err
	}

	params := &tgcalls.MediaParams{
		Path:      track.FilePath,
		Audio:     true,
		Video:     track.Video,
		SeekDelay: seekSec,
	}
	if track.Headers != nil {
		params.Headers = track.Headers
	}
	if track.FFmpegParams != "" {
		params.FFmpegArgs = track.FFmpegParams
	}

	if err := c.Play(chatID, params); err != nil {
		return err
	}

	if seekSec == 0 {
		track.Time = 1
		db.DB.AddCall(chatID)
	}
	return nil
}

// PlayNext removes the current track from the queue and plays the next.
func (e *Engine) PlayNext(chatID int64) {
	if !db.DB.GetCall(chatID) {
		return
	}
	next := queue.Q.GetNext(chatID, false)
	if next == nil {
		e.Stop(chatID)
		return
	}

	// If the file hasn't been downloaded yet, download it now
	if next.FilePath == "" {
		// Non-blocking download by importing yt package
		// (done in main package to avoid import cycles — here we log and stop)
		log.Printf("[calls] PlayNext: file not ready for %s, stopping.", next.ID)
		e.Stop(chatID)
		return
	}

	lm := lang.M.Get(db.DB.GetLang(chatID))
	sendText(botClient_ref, chatID, lm.Get("play_next"))
	_ = e.PlayMedia(botClient_ref, chatID, next, 0)
}

// Replay replays the currently playing track.
func (e *Engine) Replay(chatID int64) {
	if !db.DB.GetCall(chatID) {
		return
	}
	cur := queue.Q.GetCurrent(chatID)
	if cur == nil {
		return
	}
	lm := lang.M.Get(db.DB.GetLang(chatID))
	sendText(botClient_ref, chatID, lm.Get("play_again"))
	_ = e.PlayMedia(botClient_ref, chatID, cur, 0)
}

// ─── BOT CLIENT REFERENCE ─────────────────────────────────────────────────────
// Set by main.go after bot is started.

var botClient_ref *telegram.Client

func SetBotClient(c *telegram.Client) {
	botClient_ref = c
}

func sendText(c *telegram.Client, chatID int64, text string) {
	if c == nil {
		return
	}
	_, _ = c.SendMessage(chatID, text)
}
