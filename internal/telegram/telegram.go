package telegram

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

const (
	dlDir   = "downloads"
	maxSize = 200 * 1024 * 1024 // 200 MB
)

// Helper provides Telegram file operations.
type Helper struct {
	client      *telegram.Client
	mu          sync.Mutex
	activeDowns map[int64]chan struct{} // msgID -> cancel channel
}

var H *Helper

func New(client *telegram.Client) *Helper {
	h := &Helper{
		client:      client,
		activeDowns: make(map[int64]chan struct{}),
	}
	H = h
	return h
}

// MediaInfo describes an audible/visual Telegram attachment.
type MediaInfo struct {
	FilePath    string
	Title       string
	Duration    string
	DurationSec int
	Video       bool
	Size        int64
}

// GetMedia detects an audio/video/document from a Telegram message.
// Returns nil if none found.
func GetMedia(msg *telegram.NewMessage) *MediaInfo {
	if msg == nil || msg.Message == nil {
		return nil
	}
	// Check for audio
	if msg.Message.Media != nil {
		switch m := msg.Message.Media.(type) {
		case *telegram.MessageMediaDocument:
			doc := m.Document.(*telegram.DocumentObj)
			for _, attr := range doc.Attributes {
				switch a := attr.(type) {
				case *telegram.DocumentAttributeAudio:
					dur := fmt.Sprintf("%d:%02d", int(a.Duration)/60, int(a.Duration)%60)
					return &MediaInfo{
						Title:       a.Title,
						Duration:    dur,
						DurationSec: int(a.Duration),
						Video:       a.Voice,
						Size:        doc.Size,
					}
				case *telegram.DocumentAttributeVideo:
					dur := fmt.Sprintf("%d:%02d", int(a.Duration)/60, int(a.Duration)%60)
					return &MediaInfo{
						Duration:    dur,
						DurationSec: int(a.Duration),
						Video:       true,
						Size:        doc.Size,
					}
				}
			}
		}
	}
	return nil
}

// DownloadMedia downloads a Telegram file to downloads/ and returns its path.
func (h *Helper) DownloadMedia(msg *telegram.NewMessage, progressMsgID int64) (string, error) {
	if err := os.MkdirAll(dlDir, 0755); err != nil {
		return "", err
	}

	info := GetMedia(msg)
	if info == nil {
		return "", fmt.Errorf("no media in message")
	}
	if info.Size > maxSize {
		return "", fmt.Errorf("file too large (max 200 MB)")
	}

	ext := ".mp3"
	if info.Video {
		ext = ".mp4"
	}
	filename := filepath.Join(dlDir, fmt.Sprintf("tg_%d%s", msg.Message.ID, ext))

	// Register cancel channel
	cancelCh := make(chan struct{})
	h.mu.Lock()
	h.activeDowns[int64(progressMsgID)] = cancelCh
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.activeDowns, int64(progressMsgID))
		h.mu.Unlock()
	}()

	// Download using gogram's built-in downloader
	start := time.Now()
	path, err := h.client.DownloadMedia(msg.Message.Media, &telegram.DownloadOptions{
		FileName: filename,
	})
	if err != nil {
		return "", fmt.Errorf("telegram download: %w", err)
	}

	log.Printf("[telegram] Downloaded %s in %.2fs", path, time.Since(start).Seconds())
	return path, nil
}

// Cancel cancels an active download for a message.
func (h *Helper) Cancel(msgID int64) bool {
	h.mu.Lock()
	ch, ok := h.activeDowns[msgID]
	h.mu.Unlock()
	if ok {
		close(ch)
	}
	return ok
}

// DownloadURL downloads a file from a URL and saves it locally.
func DownloadURL(url, dest string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
