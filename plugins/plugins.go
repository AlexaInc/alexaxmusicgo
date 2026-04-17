package plugins

import (
	"fmt"
	"strings"
	"time"

	"alexamusic/internal/bot"
	"alexamusic/internal/calls"
	"alexamusic/internal/config"
	"alexamusic/internal/db"
	"alexamusic/internal/queue"
	"alexamusic/internal/tv"
	"alexamusic/internal/youtube"

	"github.com/amarnathcjd/gogram/telegram"
)

// RadioLinks maps callback data → (name, stream URL)
var RadioLinks = map[string][2]string{
	"radio_1":  {"Hiru FM", "https://radio.lotustechnologieslk.net:2020/stream/hirufmgarden"},
	"radio_2":  {"Shaa FM", "https://radio.lotustechnologieslk.net:2020/stream/shaafmgarden"},
	"radio_3":  {"FM Derana", "https://cp12.serverse.com/proxy/fmderana/stream"},
	"radio_4":  {"ITN FM", "https://cp12.serverse.com/proxy/itnfm?mp=/stream"},
	"radio_5":  {"Rhythm FM", "https://dc02.onlineradio.voaplus.com/rhythmfm"},
	"radio_6":  {"NuWaaV K-Pop", "https://streaming.live365.com/a46701"},
	"radio_7":  {"Sirasa FM", "http://live.trusl.com:1170/listen.pls"},
	"radio_8":  {"Kiss FM", "https://srv01.onlineradio.voaplus.com/kissfm"},
	"radio_9":  {"Lakhada FM", "https://cp12.serverse.com/proxy/itnfm?mp=/stream"},
	"radio_10": {"ABC Gold FM", "https://radio.lotustechnologieslk.net:8000/stream/1/"},
	"radio_11": {"bestcoast.fm", "https://streams.radio.co/sea5dddd6b/listen"},
	"radio_12": {"Bathusha Radio", "https://eu10.fastcast4u.com:14550/stream"},
	"radio_13": {"E FM", "http://207.148.74.192:7860/stream.mp3"},
	"radio_14": {"Fox", "https://cp11.serverse.com/proxy/foxfm/stream/;stream.mp3"},
	"radio_15": {"Freefm.lk", "https://stream.zeno.fm/z7q96fbw7rquv"},
	"radio_16": {"Imai FM Radio", "https://centova71.instainternet.com/proxy/imaifmradio?mp=/stream/1/"},
	"radio_17": {"Krushi Radio", "https://radioserver.krushiradio.lk:8000/radio.mp3"},
	"radio_18": {"Lite FM", "https://srv01.onlineradio.voaplus.com/lite878"},
	"radio_19": {"LiveFM", "https://cp11.serverse.com/proxy/livefm?mp=/stream"},
	"radio_20": {"Neth FM", "http://cp11.serverse.com:7669/stream/1/"},
	"radio_21": {"Ran FM", "http://207.148.74.192:7860/ran.mp3"},
	"radio_22": {"Rangiri Sri Lanka Radio", "https://rangiri.radioca.st/stream/1/"},
	"radio_23": {"Rasa FM", "https://sonic01.instainternet.com/8084/stream"},
	"radio_24": {"Real Radio", "https://srv01.onlineradio.voaplus.com/realfm"},
	"radio_25": {"Shakthi FM", "http://live.trusl.com:1160/stream/1/"},
	"radio_26": {"Red FM", "https://shaincast.caster.fm:47830/listen.mp3"},
	"radio_27": {"Shraddha Radio", "https://cp11.serverse.com/proxy/kqxjpewq?mp=/stream/1/"},
	"radio_28": {"Shree FM", "http://207.148.74.192:7860/stream2.mp3"},
	"radio_29": {"Siyatha FM", "https://dc02.onlineradio.voaplus.com/siyathafm"},
	"radio_30": {"Sitha FM", "https://shaincast.caster.fm:48148/listen.mp3"},
	"radio_31": {"SLBC City FM", "https://stream.zeno.fm/53g2h8033d0uv"},
	"radio_32": {"SLBC Kandurata FM", "http://220.247.227.20:8000/kandystream"},
	"radio_33": {"SLBC Radio Sri Lanka", "http://220.247.227.20:8000/RSLstream"},
	"radio_34": {"SLBC Tamil National Service", "http://220.247.227.6:8000/Tnsstream"},
	"radio_35": {"SLBC Sinhala Commercial Service", "https://stream.zeno.fm/fkq6fvc43d0uv.aac"},
	"radio_36": {"SLBC Thendral FM", "http://220.247.227.20:8000/Threndralstream"},
	"radio_37": {"Sun FM", "https://radio.lotustechnologieslk.net:2020/stream/sunfmgarden/stream/1/"},
	"radio_38": {"SLCB Sinhala National Service", "http://220.247.227.6:8000/Snsstream"},
	"radio_39": {"Sooriyan FM", "https://radio.lotustechnologieslk.net:2020/stream/sooriyanfmgarden/stream/1/"},
	"radio_40": {"V FM Radio", "https://dc1.serverse.com/proxy/fmlanka/stream/1/"},
	"radio_41": {"Vasantham", "https://cp12.serverse.com/proxy/vasanthamfm/stream/1/"},
	"radio_42": {"Yes FM", "http://live.trusl.com:1150/stream/1/"},
	"radio_43": {"Waharaka Radio", "http://s6.voscast.com:8112/stream/1/"},
	"radio_44": {"Y FM", "https://mbc.thestreamtech.com:7032/"},
}

// RadioMarkup returns a paged inline keyboard (8 per page, 2 columns).
func RadioMarkup(page int) *telegram.ReplyInlineMarkup {
	keys := make([]string, 0, len(RadioLinks))
	for i := 1; i <= len(RadioLinks); i++ {
		keys = append(keys, fmt.Sprintf("radio_%d", i))
	}
	perPage := 8
	total := (len(keys) + perPage - 1) / perPage
	start := (page - 1) * perPage
	end := start + perPage
	if end > len(keys) {
		end = len(keys)
	}
	var rows [][]telegram.KeyboardButton
	current := keys[start:end]
	for i := 0; i < len(current); i += 2 {
		row := []telegram.KeyboardButton{
			bot.InlineKeyboardButton(RadioLinks[current[i]][0], current[i]),
		}
		if i+1 < len(current) {
			row = append(row, bot.InlineKeyboardButton(RadioLinks[current[i+1]][0], current[i+1]))
		}
		rows = append(rows, row)
	}
	var nav []telegram.KeyboardButton
	if page > 1 {
		nav = append(nav, bot.InlineKeyboardButton("⬅️ Back", fmt.Sprintf("radio_page_%d", page-1)))
	}
	nav = append(nav, bot.InlineKeyboardButton(fmt.Sprintf("Page %d/%d", page, total), "none"))
	if page < total {
		nav = append(nav, bot.InlineKeyboardButton("Next ➡️", fmt.Sprintf("radio_page_%d", page+1)))
	}
	rows = append(rows, nav)
	rows = append(rows, []telegram.KeyboardButton{
		bot.InlineKeyboardButton("❌ Close Menu", "help close"),
	})
	return bot.Markup(rows...)
}

func RegisterPlay(b *bot.Bot, yt *youtube.YouTube, q *queue.Manager, cfg *config.Config) {
	b.Client.OnCommand("play", func(m *telegram.NewMessage) error {
		return handlePlay(b, yt, q, cfg, m, false, false)
	}).Filter(telegram.IsGroup)

	b.Client.OnCommand("vplay", func(m *telegram.NewMessage) error {
		return handlePlay(b, yt, q, cfg, m, false, true)
	}).Filter(telegram.IsGroup)

	b.Client.OnCommand("playforce", func(m *telegram.NewMessage) error {
		return handlePlay(b, yt, q, cfg, m, true, false)
	}).Filter(telegram.IsGroup)

	b.Client.OnCommand("vplayforce", func(m *telegram.NewMessage) error {
		return handlePlay(b, yt, q, cfg, m, true, true)
	}).Filter(telegram.IsGroup)

	b.Client.OnCommand("radio", func(m *telegram.NewMessage) error {
		_, err := m.Reply("📻 <b>Radio Station Menu</b>\nChoose a station to start streaming:",
			&telegram.SendOptions{ReplyMarkup: RadioMarkup(1)})
		return err
	}).Filter(telegram.IsGroup)

	b.Client.OnCommand("tv", func(m *telegram.NewMessage) error {
		_, err := m.Reply("📺 <b>TV Station Categories</b>\nChoose a category to find a station:",
			&telegram.SendOptions{ReplyMarkup: tv.CategoryMarkup()})
		return err
	}).Filter(telegram.IsGroup)

	b.Client.OnCommand("stop", func(m *telegram.NewMessage) error { return handleStop(b, q, m) }).Filter(telegram.IsGroup)
	b.Client.OnCommand("end", func(m *telegram.NewMessage) error { return handleStop(b, q, m) }).Filter(telegram.IsGroup)
	b.Client.OnCommand("pause", func(m *telegram.NewMessage) error { return handlePause(m) }).Filter(telegram.IsGroup)
	b.Client.OnCommand("resume", func(m *telegram.NewMessage) error { return handleResume(m) }).Filter(telegram.IsGroup)
	b.Client.OnCommand("skip", func(m *telegram.NewMessage) error { return handleSkip(m) }).Filter(telegram.IsGroup)
}

func handlePlay(b *bot.Bot, yt *youtube.YouTube, q *queue.Manager, cfg *config.Config, m *telegram.NewMessage, force, video bool) error {
	chatID := m.ChatID()
	userID := m.Sender.ID
	if b.IsBlacklisted(userID) || b.IsBlacklisted(chatID) {
		return nil
	}
	lm := bot.GetLang(chatID)
	_, args := bot.ParseCommand(m.Text())
	if db.DB.GetPlayMode(chatID) || force {
		admins := db.DB.GetAdmins(chatID)
		isSudo, isAdmin, isAuth := b.IsSudo(userID), false, db.DB.IsAuth(chatID, userID)
		for _, a := range admins {
			if a == userID {
				isAdmin = true
				break
			}
		}
		if !isAdmin && !isAuth && !isSudo {
			_, err := m.Reply(lm.Get("play_admin"))
			return err
		}
	}
	if len(q.GetQueue(chatID)) >= cfg.QueueLimit {
		_, err := m.Reply(lm.Get("play_queue_full", cfg.QueueLimit))
		return err
	}
	sent, err := m.Reply(lm.Get("play_searching"))
	if err != nil {
		return err
	}
	query := strings.Join(args, " ")
	if query == "" {
		_, _ = sent.Edit(lm.Get("play_usage"))
		return nil
	}
	track, err := yt.Search(query, int(sent.ID), video)
	if err != nil || track == nil {
		_, _ = sent.Edit(lm.Get("play_not_found", cfg.SupportChat))
		return nil
	}
	if track.DurationSec > cfg.DurationLimit {
		_, _ = sent.Edit(lm.Get("play_duration_limit", cfg.DurationLimit/60))
		return nil
	}
	track.User = m.Sender.FirstName
	if force {
		q.ForceAdd(chatID, track, 0)
	} else {
		pos := q.Add(chatID, track)
		if db.DB.GetCall(chatID) {
			_, _ = sent.Edit(lm.Get("play_queued", pos, track.URL, track.Title, track.Duration, track.User),
				&telegram.SendOptions{ReplyMarkup: bot.QueuedMarkup(chatID, track.ID, lm.Get("play_now"))})
			return nil
		}
	}
	_, _ = sent.Edit(lm.Get("play_downloading"))
	filePath, err := yt.Download(track.ID, video)
	if err != nil || filePath == "" {
		_, _ = sent.Edit(lm.Get("error_no_file", cfg.SupportChat))
		return nil
	}
	track.FilePath = filePath
	if err := calls.E.PlayMedia(b.Client, chatID, track, 0); err != nil {
		_, _ = sent.Edit(lm.Get("error_no_call"))
		return nil
	}
	_, _ = sent.Edit(lm.Get("play_media", track.URL, track.Title, track.Duration, track.User),
		&telegram.SendOptions{ReplyMarkup: bot.ControlsMarkup(chatID)})
	return nil
}

func handleStop(b *bot.Bot, q *queue.Manager, m *telegram.NewMessage) error {
	chatID := m.ChatID()
	lm := bot.GetLang(chatID)
	if !db.DB.GetCall(chatID) {
		_, err := m.Reply(lm.Get("not_playing"))
		return err
	}
	calls.E.Stop(chatID)
	_, err := m.Reply(lm.Get("play_stopped", m.Sender.FirstName))
	return err
}

func handlePause(m *telegram.NewMessage) error {
	chatID := m.ChatID()
	lm := bot.GetLang(chatID)
	if !db.DB.GetCall(chatID) {
		_, err := m.Reply(lm.Get("not_playing"))
		return err
	}
	if !db.DB.Playing(chatID, -1) {
		_, err := m.Reply(lm.Get("play_already_paused"))
		return err
	}
	_ = calls.E.Pause(chatID)
	_, err := m.Reply(lm.Get("play_paused", m.Sender.FirstName))
	return err
}

func handleResume(m *telegram.NewMessage) error {
	chatID := m.ChatID()
	lm := bot.GetLang(chatID)
	if !db.DB.GetCall(chatID) {
		_, err := m.Reply(lm.Get("not_playing"))
		return err
	}
	if db.DB.Playing(chatID, -1) {
		_, err := m.Reply(lm.Get("play_not_paused"))
		return err
	}
	_ = calls.E.Resume(chatID)
	_, err := m.Reply(lm.Get("play_resumed", m.Sender.FirstName))
	return err
}

func handleSkip(m *telegram.NewMessage) error {
	chatID := m.ChatID()
	lm := bot.GetLang(chatID)
	if !db.DB.GetCall(chatID) {
		_, err := m.Reply(lm.Get("not_playing"))
		return err
	}
	calls.E.PlayNext(chatID)
	_, err := m.Reply(lm.Get("play_skipped", m.Sender.FirstName))
	return err
}

func RegisterSeek(b *bot.Bot, q *queue.Manager) {
	b.Client.OnCommand("seek", func(m *telegram.NewMessage) error {
		chatID := m.ChatID()
		lm := bot.GetLang(chatID)
		_, args := bot.ParseCommand(m.Text())
		if len(args) == 0 {
			_, err := m.Reply(lm.Get("play_seek_usage", "seek"))
			return err
		}
		secs := 0
		fmt.Sscan(args[0], &secs)
		if secs < 10 {
			_, err := m.Reply(lm.Get("play_seek_min"))
			return err
		}
		cur := q.GetCurrent(chatID)
		if cur == nil {
			_, err := m.Reply(lm.Get("not_playing"))
			return err
		}
		_ = calls.E.PlayMedia(b.Client, chatID, cur, secs)
		_, err := m.Reply(lm.Get("play_seeked", "forward", fmt.Sprintf("%d:%02d", secs/60, secs%60), m.Sender.FirstName))
		return err
	}).Filter(telegram.IsGroup)
}

func RegisterQueue(b *bot.Bot, q *queue.Manager) {
	b.Client.OnCommand("queue", func(m *telegram.NewMessage) error {
		chatID := m.ChatID()
		lm := bot.GetLang(chatID)
		items := q.GetQueue(chatID)
		if len(items) == 0 {
			_, err := m.Reply(lm.Get("not_playing"))
			return err
		}
		cur := items[0]
		text := lm.Get("queue_curr", cur.URL, cur.Title, cur.Duration, cur.User)
		for i, t := range items[1:] {
			text += lm.Get("queue_item", i+1, t.Title, t.Duration)
		}
		_, err := m.Reply(text, &telegram.SendOptions{ReplyMarkup: bot.Markup([]telegram.KeyboardButton{bot.InlineKeyboardButton(lm.Get("paused"), fmt.Sprintf("controls pause %d q", chatID))})})
		return err
	}).Filter(telegram.IsGroup)
}

func RegisterActive(b *bot.Bot) {
	b.Client.OnCommand("active", func(m *telegram.NewMessage) error {
		if !b.IsSudo(m.Sender.ID) {
			return nil
		}
		lm := bot.GetLang(m.ChatID())
		if len(db.DB.ActiveCalls) == 0 {
			_, err := m.Reply(lm.Get("vc_empty"))
			return err
		}
		ids := []string{}
		for cid := range db.DB.ActiveCalls {
			ids = append(ids, fmt.Sprintf("<code>%d</code>", cid))
		}
		_, err := m.Reply(lm.Get("vc_list") + "\n" + strings.Join(ids, "\n"))
		return err
	})
}

func RegisterAuth(b *bot.Bot) {
	b.Client.OnCommand("auth", func(m *telegram.NewMessage) error {
		cID, uID := m.ChatID(), m.Sender.ID
		lm := bot.GetLang(cID)
		if !b.IsSudo(uID) && !isAdmin(b, cID, uID) {
			_, err := m.Reply(lm.Get("user_not_admin"))
			return err
		}
		tID := getReplyUserID(m)
		if tID == 0 {
			_, err := m.Reply(lm.Get("user_not_found"))
			return err
		}
		db.DB.AddAuth(cID, tID)
		_, err := m.Reply(lm.Get("auth_added", fmt.Sprintf("<code>%d</code>", tID)))
		return err
	}).Filter(telegram.IsGroup)
 
	b.Client.OnCommand("unauth", func(m *telegram.NewMessage) error {
		cID, uID := m.ChatID(), m.Sender.ID
		lm := bot.GetLang(cID)
		if !b.IsSudo(uID) && !isAdmin(b, cID, uID) {
			_, err := m.Reply(lm.Get("user_not_admin"))
			return err
		}
		tID := getReplyUserID(m)
		if tID == 0 {
			_, err := m.Reply(lm.Get("user_not_found"))
			return err
		}
		db.DB.RmAuth(cID, tID)
		_, err := m.Reply(lm.Get("auth_removed", fmt.Sprintf("<code>%d</code>", tID)))
		return err
	}).Filter(telegram.IsGroup)
}

func isAdmin(b *bot.Bot, chatID, userID int64) bool {
	for _, a := range db.DB.GetAdmins(chatID) {
		if a == userID {
			return true
		}
	}
	return false
}

func getReplyUserID(m *telegram.NewMessage) int64 {
	if m.IsReply() {
		return m.ReplySenderID()
	}
	return 0
}

func RegisterBlacklist(b *bot.Bot) {
	b.Client.OnCommand("blacklist", func(m *telegram.NewMessage) error {
		if !b.IsSudo(m.Sender.ID) { return nil }
		_, args := bot.ParseCommand(m.Text())
		if len(args) == 0 { return nil }
		var id int64
		fmt.Sscan(args[0], &id)
		db.DB.AddBlacklist(id)
		return nil
	})
}

func RegisterBroadcast(b *bot.Bot) {
	b.Client.OnCommand("broadcast", func(m *telegram.NewMessage) error {
		if !b.IsSudo(m.Sender.ID) || m.ReplyToMsgID() == 0 { return nil }
		lm := bot.GetLang(m.ChatID())
		sent, failed := 0, 0
		replyID := m.ReplyToMsgID()
		for _, cid := range db.DB.GetChats() {
			_, err := b.ForwardMessage(cid, m.ChatID(), replyID)
			if err != nil { failed++ } else { sent++ }
			time.Sleep(100 * time.Millisecond)
		}
		_, err := m.Reply(lm.Get("gcast_end", sent, failed))
		return err
	})
}

func RegisterPing(b *bot.Bot) {
	b.Client.OnCommand("ping", func(m *telegram.NewMessage) error {
		lm := bot.GetLang(m.ChatID())
		start := time.Now()
		sent, _ := m.Reply(lm.Get("pinging"))
		latency := time.Since(start).Milliseconds()
		uptime := time.Since(db.BootTime)
		uptimeStr := fmt.Sprintf("%02d:%02d:%02d", int(uptime.Hours()), int(uptime.Minutes())%60, int(uptime.Seconds())%60)
		_, err := sent.Edit(lm.Get("ping_pong", latency, uptimeStr, "0", "0", "0", "0"), &telegram.SendOptions{ReplyMarkup: bot.Markup([]telegram.KeyboardButton{bot.InlineURLButton(lm.Get("support"), b.Config.SupportChat)})})
		return err
	})
}

func RegisterStart(b *bot.Bot, cfg *config.Config) {
	b.Client.OnCommand("start", func(m *telegram.NewMessage) error {
		chatID, userID := m.ChatID(), m.Sender.ID
		lm := bot.GetLang(chatID)
		if b.IsBlacklisted(userID) {
			_, err := m.Reply(lm.Get("bl_user_notify", cfg.SupportChat))
			return err
		}
		me, _ := b.GetMe()
		isPrivate := chatID == userID
		text := lm.Get("start_gp", cfg.MusicBotName)
		if isPrivate {
			text = lm.Get("start_pm", m.Sender.FirstName, cfg.MusicBotName)
		}
		key := bot.StartMarkup(lm, me.Username, isPrivate, cfg)
		_, err := b.Client.SendMedia(chatID, cfg.StartImg, &telegram.MediaOptions{Caption: text, ReplyMarkup: key})
		if err != nil {
			_, err = m.Reply(text, &telegram.SendOptions{ReplyMarkup: key})
		}
		if isPrivate && !db.DB.IsUser(userID) {
			db.DB.AddUser(userID)
		} else if !isPrivate && !db.DB.IsChat(chatID) {
			db.DB.AddChat(chatID)
		}
		return err
	})

	b.Client.OnCommand("help", func(m *telegram.NewMessage) error {
		lm := bot.GetLang(m.ChatID())
		_, err := m.Reply(lm.Get("help_menu"), &telegram.SendOptions{ReplyMarkup: bot.HelpMarkup(lm, false)})
		return err
	})

	b.Client.OnCommand("settings", func(m *telegram.NewMessage) error {
		cID := m.ChatID()
		lm := bot.GetLang(cID)
		_, err := m.Reply(lm.Get("start_settings", m.Chat.Title), &telegram.SendOptions{ReplyMarkup: bot.SettingsMarkup(lm, db.DB.GetPlayMode(cID), db.DB.GetCmdDelete(cID), db.DB.GetLang(cID), cID)})
		return err
	}).Filter(telegram.IsGroup)
}

func RegisterStats(b *bot.Bot, cfg *config.Config) {
	b.Client.OnCommand("stats", func(m *telegram.NewMessage) error {
		chatID := m.ChatID()
		lm := bot.GetLang(chatID)
		text := lm.Get("stats_user", cfg.MusicBotName, len(db.DB.ActiveCalls), cfg.AutoLeave, len(db.DB.Blacklisted), 0, len(b.SudoIDs), len(db.DB.GetChats()), len(db.DB.GetUsers()))
		_, err := m.Reply(text)
		return err
	}).Filter(telegram.IsGroup)
}

func RegisterLanguage(b *bot.Bot) {
	b.Client.OnCommand("lang", func(m *telegram.NewMessage) error {
		chatID := m.ChatID()
		lm := bot.GetLang(chatID)
		_, err := m.Reply(lm.Get("lang_choose"), &telegram.SendOptions{ReplyMarkup: bot.LangMarkup(db.DB.GetLang(chatID))})
		return err
	}).Filter(telegram.IsGroup)
}

func RegisterSudo(b *bot.Bot) {
	b.Client.OnCommand("addsudo", func(m *telegram.NewMessage) error {
		if !b.IsSudo(m.Sender.ID) {
			_, _ = m.Reply("❌ <b>Sudo access required.</b>")
			return nil
		}
		_, args := bot.ParseCommand(m.Text())
		if len(args) == 0 { return nil }
		var uid int64
		fmt.Sscan(args[0], &uid)
		db.DB.AddSudo(uid)
		b.SudoIDs[uid] = true
		return nil
	})
}

func RegisterCallbacks(b *bot.Bot, ytHelper *youtube.YouTube, q *queue.Manager, cfg *config.Config) {
	b.Client.OnCallback("", func(c *telegram.CallbackQuery) error {
		data := c.DataString()
		chatID := c.GetChatID()
		lm := bot.GetLang(chatID)
		userMention := c.Sender.FirstName

		if strings.HasPrefix(data, "controls ") {
			parts := strings.Fields(data)
			if len(parts) < 3 { return nil }
			action, cidStr := parts[1], parts[2]
			var cid int64
			fmt.Sscan(cidStr, &cid)
			if action == "pause" { _ = calls.E.Pause(cid) } else if action == "resume" { _ = calls.E.Resume(cid) } else if action == "skip" { calls.E.PlayNext(cid) } else if action == "stop" { calls.E.Stop(cid) } else if action == "replay" { calls.E.Replay(cid) }
			_, _ = c.Answer(lm.Get("processing"))
			return nil
		}
		if data == "tv_home" { _, _ = b.EditMessageReplyMarkup(chatID, c.MessageID, tv.CategoryMarkup()) }
		if strings.HasPrefix(data, "tv_cat:") {
			cat := strings.TrimPrefix(data, "tv_cat:")
			_, _ = c.Edit(fmt.Sprintf("📺 <b>Category: %s</b>\nChoose a station:", cat), &telegram.SendOptions{ReplyMarkup: tv.ChannelMarkup(cat, 1)})
		}
		if strings.HasPrefix(data, "tv_page:") {
			parts := strings.Split(data, ":")
			cat := parts[1]
			var p int
			fmt.Sscan(parts[2], &p)
			_, _ = c.Edit(fmt.Sprintf("📺 <b>Category: %s</b>\nChoose a station (Page %d):", cat, p), &telegram.SendOptions{ReplyMarkup: tv.ChannelMarkup(cat, p)})
		}
		if strings.HasPrefix(data, "tv_ch:") {
			id := strings.TrimPrefix(data, "tv_ch:")
			channel := tv.GetByID(id)
			if channel == nil {
				_, _ = c.Answer("Station Not Found!", &telegram.CallbackOptions{Alert: true})
				return nil
			}
			_, _ = c.Answer(fmt.Sprintf("Fetching %s stream...", channel.Title))
			url, _ := tv.FetchStreamURL(channel.Manifest, cfg.ProxyURL)
			if url == "" { return nil }
			track := &queue.Track{ID: "tv_live", Title: "TV: " + channel.Title, URL: url, FilePath: url, User: userMention, Video: true, StreamType: "live", Headers: map[string]string{"User-Agent": "Mozilla/11.0", "Referer": "https://viu.lk/"}, Thumbnail: channel.Thumbnail}
			q.ForceAdd(chatID, track, 0)
			_ = calls.E.PlayMedia(b.Client, chatID, track, 0)
			_, _ = c.Edit(fmt.Sprintf("📡 <b>Now Streaming:</b> %s (Low Quality)\nRequested by: %s", channel.Title, userMention), &telegram.SendOptions{ReplyMarkup: bot.ControlsMarkup(chatID)})
		}
		if strings.HasPrefix(data, "radio_page_") {
			var p int
			fmt.Sscan(strings.TrimPrefix(data, "radio_page_"), &p)
			_, _ = b.EditMessageReplyMarkup(chatID, c.MessageID, RadioMarkup(p))
		}
		if strings.HasPrefix(data, "radio_") {
			link, ok := RadioLinks[data]
			if !ok { return nil }
			_, _ = c.Answer(fmt.Sprintf("Switching to %s...", link[0]))
			track := &queue.Track{ID: "radio_live", Title: "Radio: " + link[0], URL: link[1], FilePath: link[1], User: userMention, Video: false, StreamType: "live", Thumbnail: cfg.DefaultThumb}
			q.ForceAdd(chatID, track, 0)
			_ = calls.E.PlayMedia(b.Client, chatID, track, 0)
			_, _ = c.Edit(fmt.Sprintf("📡 <b>Now Streaming:</b> <code>%s</code>\n👤 <b>Requested by:</b> %s", link[0], userMention), &telegram.SendOptions{ReplyMarkup: bot.ControlsMarkup(chatID)})
		}
		if strings.HasPrefix(data, "lang_change ") {
			code := strings.TrimPrefix(data, "lang_change ")
			db.DB.SetLang(chatID, code)
			_, _ = c.Answer(bot.GetLang(chatID).Get("lang_changed", code), &telegram.CallbackOptions{Alert: true})
		}
		if strings.HasPrefix(data, "settings ") {
			parts := strings.Fields(data)
			if len(parts) >= 2 {
				if parts[1] == "play" {
					db.DB.SetPlayMode(chatID, !db.DB.GetPlayMode(chatID))
				} else if parts[1] == "delete" {
					db.DB.SetCmdDelete(chatID, !db.DB.GetCmdDelete(chatID))
				}
			}
			_, _ = b.EditMessageReplyMarkup(chatID, c.MessageID, bot.SettingsMarkup(lm, db.DB.GetPlayMode(chatID), db.DB.GetCmdDelete(chatID), db.DB.GetLang(chatID), chatID))
		}
		if data == "help close" { _, _ = c.Delete() }
		if strings.HasPrefix(data, "help ") {
			sub := strings.TrimPrefix(data, "help ")
			if sub == "back" {
				_, _ = c.Edit(lm.Get("help_menu"), &telegram.SendOptions{ReplyMarkup: bot.HelpMarkup(lm, false)})
			} else {
				_, _ = c.Edit(lm.Get("help_"+sub), &telegram.SendOptions{ReplyMarkup: bot.HelpMarkup(lm, true)})
			}
		}
		return nil
	})
}

func RegisterAll(b *bot.Bot, yt *youtube.YouTube, q *queue.Manager, cfg *config.Config) {
	RegisterStart(b, cfg); RegisterPlay(b, yt, q, cfg); RegisterSeek(b, q); RegisterQueue(b, q); RegisterActive(b); RegisterAuth(b); RegisterBlacklist(b); RegisterBroadcast(b); RegisterCallbacks(b, yt, q, cfg); RegisterLanguage(b); RegisterPing(b); RegisterStats(b, cfg); RegisterSudo(b)
}
