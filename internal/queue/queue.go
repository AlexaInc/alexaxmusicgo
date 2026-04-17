package queue

import (
	"sync"
)

// Q is the global queue manager singleton, set by main.go.
var Q *Manager

// Track represents a YouTube or Telegram media item.
type Track struct {
	ID           string
	ChannelName  string
	Duration     string
	DurationSec  int
	Title        string
	URL          string
	FilePath     string
	MessageID    int
	Time         int // seconds played
	Thumbnail    string
	User         string
	ViewCount    string
	Video        bool
	StreamType   string // "live", ""
	Headers      map[string]string
	FFmpegParams string
}

// Media represents a directly downloaded Telegram media.
type Media struct {
	ID          string
	Duration    string
	DurationSec int
	FilePath    string
	MessageID   int
	Title       string
	URL         string
	Time        int
	User        string
	Video       bool
}

// Manager is a thread-safe per-chat queue.
type Manager struct {
	mu     sync.Mutex
	queues map[int64][]*Track
}

func NewManager() *Manager {
	return &Manager{queues: make(map[int64][]*Track)}
}

// Add appends a track and returns its 1-based queue position (0 = now playing).
func (m *Manager) Add(chatID int64, t *Track) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queues[chatID] = append(m.queues[chatID], t)
	return len(m.queues[chatID]) - 1
}

// ForceAdd replaces the current track and optionally removes an item at pos.
func (m *Manager) ForceAdd(chatID int64, t *Track, removePos int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := m.queues[chatID]
	if len(q) > 0 {
		q = q[1:] // drop current
	}
	if removePos > 0 && removePos <= len(q) {
		q = append(q[:removePos-1], q[removePos:]...)
	}
	m.queues[chatID] = append([]*Track{t}, q...)
}

// GetCurrent returns the first item (currently playing) or nil.
func (m *Manager) GetCurrent(chatID int64) *Track {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := m.queues[chatID]
	if len(q) == 0 {
		return nil
	}
	return q[0]
}

// GetNext removes current and returns the next, or nil if queue is empty.
func (m *Manager) GetNext(chatID int64, checkOnly bool) *Track {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := m.queues[chatID]
	if len(q) == 0 {
		return nil
	}
	if checkOnly {
		if len(q) > 1 {
			return q[1]
		}
		return nil
	}
	q = q[1:]
	m.queues[chatID] = q
	if len(q) == 0 {
		return nil
	}
	return q[0]
}

// CheckItem searches the queue for a Track with the given ID.
func (m *Manager) CheckItem(chatID int64, id string) (int, *Track) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.queues[chatID] {
		if t.ID == id {
			return i, t
		}
	}
	return -1, nil
}

// GetQueue returns a copy of the full queue.
func (m *Manager) GetQueue(chatID int64) []*Track {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Track, len(m.queues[chatID]))
	copy(out, m.queues[chatID])
	return out
}

// RemoveCurrent drops the first item.
func (m *Manager) RemoveCurrent(chatID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.queues[chatID]) > 0 {
		m.queues[chatID] = m.queues[chatID][1:]
	}
}

// Clear empties the queue for a chat.
func (m *Manager) Clear(chatID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.queues, chatID)
}

// IncrementTime adds 1 second to the currently playing track.
func (m *Manager) IncrementTime(chatID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if q := m.queues[chatID]; len(q) > 0 {
		q[0].Time++
	}
}
