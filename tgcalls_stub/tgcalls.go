package tgcalls

import (
	"log"
	"github.com/amarnathcjd/gogram/telegram"
)

type Track struct {
	ID       string
	FilePath string
	Audio    bool
}

type MediaParams struct {
	Path       string
	Audio      bool
	Video      bool
	SeekDelay  int
	Headers    map[string]string
	FFmpegArgs string
}

type GroupCall struct {
	client *telegram.Client
}

func NewGroupCall(client *telegram.Client) *GroupCall {
	return &GroupCall{client: client}
}

func (t *GroupCall) Play(chatID int64, params *MediaParams) error {
	log.Printf("[stub] Playing %s in %d", params.Path, chatID)
	return nil
}

func (t *GroupCall) Stop(chatID int64) error {
	log.Printf("[stub] Stopping %d", chatID)
	return nil
}

func (t *GroupCall) Pause(chatID int64) error {
	log.Printf("[stub] Pausing %d", chatID)
	return nil
}

func (t *GroupCall) Resume(chatID int64) error {
	log.Printf("[stub] Resuming %d", chatID)
	return nil
}

func (t *GroupCall) JoinCall(chatID int64) error {
	log.Printf("[stub] Joining %d", chatID)
	return nil
}

func (t *GroupCall) LeaveCall(chatID int64) error {
	log.Printf("[stub] Leaving %d", chatID)
	return nil
}

func (t *GroupCall) OnStreamEnd(f func(int64)) {
	log.Println("[stub] OnStreamEnd handler registered")
}

func (t *GroupCall) OnLeave(f func(int64)) {
	log.Println("[stub] OnLeave handler registered")
}
