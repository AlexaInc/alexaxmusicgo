package userbot

import (
	"fmt"
	"log"

	"alexamusic/internal/config"

	"github.com/amarnathcjd/gogram/telegram"
)

// Userbot manages up to 3 userbot sessions for voice-call participation.
type Userbot struct {
	Clients []*telegram.Client
	cfg     *config.Config
}

var UB *Userbot

func New(cfg *config.Config) *Userbot {
	ub := &Userbot{cfg: cfg}
	UB = ub
	return ub
}

// Boot starts all configured userbot sessions.
func (u *Userbot) Boot() error {
	sessions := []struct {
		name   string
		strSes string
	}{
		{"AlexaHelper1", u.cfg.Session1},
		{"AlexaHelper2", u.cfg.Session2},
		{"AlexaHelper3", u.cfg.Session3},
	}

	for _, s := range sessions {
		if s.strSes == "" {
			continue
		}
		client, err := u.startClient(s.name, s.strSes)
		if err != nil {
			return fmt.Errorf("userbot %s: %w", s.name, err)
		}
		u.Clients = append(u.Clients, client)
	}

	if len(u.Clients) == 0 {
		return fmt.Errorf("no userbot sessions configured")
	}

	log.Printf("[userbot] %d assistant(s) started.", len(u.Clients))
	return nil
}

func (u *Userbot) startClient(name, session string) (*telegram.Client, error) {
	client, err := telegram.NewClient(telegram.ClientConfig{
		AppID:         int32(u.cfg.APIID),
		AppHash:       u.cfg.APIHash,
		StringSession: session,
		MemorySession: true,
		Logger:        nil,
	})
	if err != nil {
		return nil, err
	}

	if err := client.Start(); err != nil {
		return nil, err
	}

	// Send startup message to logger
	_, err = client.SendMessage(u.cfg.LoggerID, fmt.Sprintf("[%s] Assistant Started ✅", name))
	if err != nil {
		log.Printf("[userbot] %s could not send startup message: %v", name, err)
	}
	if err != nil {
		log.Printf("[userbot] %s could not send startup message: %v", name, err)
	}

	me, _ := client.GetMe()
	if me != nil {
		log.Printf("[userbot] %s started as @%s", name, me.Username)
	}

	return client, nil
}

// Stop gracefully stops all userbot clients.
func (u *Userbot) Stop() {
	for _, c := range u.Clients {
		_ = c.Stop()
	}
	log.Println("[userbot] All assistants stopped.")
}
