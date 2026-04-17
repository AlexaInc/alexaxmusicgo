package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"alexamusic/internal/bot"
	"alexamusic/internal/calls"
	"alexamusic/internal/config"
	"alexamusic/internal/dashboard"
	"alexamusic/internal/db"
	"alexamusic/internal/lang"
	"alexamusic/internal/queue"
	"alexamusic/internal/userbot"
	"alexamusic/internal/youtube"
	"alexamusic/internal/tv"
	"alexamusic/plugins"
)

func main() {
	// 1. Load configuration
	cfg := config.Load()
	cfg.Check()

	// 2. Start HTTP dashboard early so health checks pass immediately
	dashboard.Start(cfg)

	// 3. Load i18n locales
	lang.Load("anony/locales")

	// 3.5. Load TV stations
	if err := tv.Load("anony/helpers/channels.json"); err != nil {
		log.Printf("[main] Warning: TV channels failed to load: %v", err)
	}

	// 4. Connect to MongoDB
	database := db.Connect(cfg)
	defer database.Close()

	// 5. Initialise queue
	q := queue.NewManager()
	queue.Q = q

	// 6. Boot userbot sessions
	ub := userbot.New(cfg)
	if err := ub.Boot(); err != nil {
		log.Fatalf("[main] Userbot boot failed: %v", err)
	}
	defer ub.Stop()

	// 7. Update number of assistants in db for random assignment
	db.NumAssistants = len(ub.Clients)

	// 8. Initialise voice-call engine
	callEngine := calls.New()
	if err := callEngine.Boot(); err != nil {
		log.Fatalf("[main] Calls boot failed: %v", err)
	}

	// 9. Create bot client
	b, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("[main] Bot creation failed: %v", err)
	}
	if err := b.Boot(); err != nil {
		log.Fatalf("[main] Bot boot failed: %v", err)
	}

	// Share bot reference with calls engine (for sending messages)
	calls.SetBotClient(b.Client)

	// 10. Create YouTube helper
	yt := youtube.New()

	// 11. Register all command plugins
	plugins.RegisterAll(b, yt, q, cfg)

	// 12. Start background workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	plugins.StartWorkers(ctx, b, yt, q)
	if cfg.AutoEnd {
		go plugins.VCWatcher(ctx, b, q)
	}

	// 13. Wait for shutdown signal
	log.Println("[main] AlexaMusic Go bot is running. Press Ctrl+C to stop.")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("[main] Shutting down...")
	cancel()
}
