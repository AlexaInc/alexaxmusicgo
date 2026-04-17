package db

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"alexamusic/internal/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BootTime is set when the database connects (used for uptime reporting).
var BootTime = time.Now()

// MongoDB is the main database layer with in-memory cache.
type MongoDB struct {
	client *mongo.Client
	db     *mongo.Database

	// Collections
	cacheDB     *mongo.Collection
	assistantDB *mongo.Collection
	authDB      *mongo.Collection
	chatsDB     *mongo.Collection
	langDB      *mongo.Collection
	usersDB     *mongo.Collection

	mu          sync.RWMutex
	adminList   map[int64][]int64
	ActiveCalls map[int64]int // 0=paused, 1=playing
	adminPlay   map[int64]bool
	Blacklisted []int64
	cmdDelete   map[int64]bool
	Notified    []int64
	loggerOn    bool

	assistant map[int64]int // chatID -> assistant number (1-3)
	auth      map[int64]map[int64]bool
	Chats     []int64
	Lang      map[int64]string
	Users     []int64
	Sudoers   []int64
}

var DB *MongoDB

// Connect initialises the MongoDB connection and loads cache.
func Connect(cfg *config.Config) *MongoDB {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURL))
	if err != nil {
		log.Fatalf("[db] MongoDB connect failed: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("[db] MongoDB ping failed: %v", err)
	}
	log.Println("[db] MongoDB connected.")

	mdb := client.Database("Anon")
	m := &MongoDB{
		client:      client,
		db:          mdb,
		cacheDB:     mdb.Collection("cache"),
		assistantDB: mdb.Collection("assistant"),
		authDB:      mdb.Collection("auth"),
		chatsDB:     mdb.Collection("chats"),
		langDB:      mdb.Collection("lang"),
		usersDB:     mdb.Collection("users"),

		adminList:   make(map[int64][]int64),
		ActiveCalls: make(map[int64]int),
		adminPlay:   make(map[int64]bool),
		cmdDelete:   make(map[int64]bool),
		assistant:   make(map[int64]int),
		auth:        make(map[int64]map[int64]bool),
		Lang:        make(map[int64]string),
	}
	DB = m
	m.loadCache()
	return m
}

func (m *MongoDB) Close() {
	_ = m.client.Disconnect(context.Background())
	log.Println("[db] MongoDB connection closed.")
}

// ─── ACTIVE CALLS ──────────────────────────────────────────────

func (m *MongoDB) GetCall(chatID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.ActiveCalls[chatID]
	return ok
}

func (m *MongoDB) AddCall(chatID int64) {
	m.mu.Lock()
	m.ActiveCalls[chatID] = 1
	m.mu.Unlock()
}

func (m *MongoDB) RemoveCall(chatID int64) {
	m.mu.Lock()
	delete(m.ActiveCalls, chatID)
	m.mu.Unlock()
}

// Playing returns current play state; if setVal >= 0, it sets the state first.
func (m *MongoDB) Playing(chatID int64, setVal int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if setVal >= 0 {
		m.ActiveCalls[chatID] = setVal
	}
	return m.ActiveCalls[chatID] == 1
}

func (m *MongoDB) GetActiveCallsCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.ActiveCalls)
}

// ─── AUTH ──────────────────────────────────────────────────────

func (m *MongoDB) getAuth(chatID int64) map[int64]bool {
	if m.auth[chatID] == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var doc struct {
			UserIDs []int64 `bson:"user_ids"`
		}
		_ = m.authDB.FindOne(ctx, bson.M{"_id": chatID}).Decode(&doc)
		m.auth[chatID] = make(map[int64]bool)
		for _, u := range doc.UserIDs {
			m.auth[chatID][u] = true
		}
	}
	return m.auth[chatID]
}

func (m *MongoDB) IsAuth(chatID, userID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getAuth(chatID)[userID]
}

func (m *MongoDB) AddAuth(chatID, userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := m.getAuth(chatID)
	if !a[userID] {
		a[userID] = true
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = m.authDB.UpdateOne(ctx, bson.M{"_id": chatID},
			bson.M{"$addToSet": bson.M{"user_ids": userID}},
			options.Update().SetUpsert(true))
	}
}

func (m *MongoDB) RmAuth(chatID, userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := m.getAuth(chatID)
	if a[userID] {
		delete(a, userID)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = m.authDB.UpdateOne(ctx, bson.M{"_id": chatID},
			bson.M{"$pull": bson.M{"user_ids": userID}})
	}
}

// ─── ASSISTANT ─────────────────────────────────────────────────

// NumAssistants must be set after userbot boot.
var NumAssistants int

func (m *MongoDB) SetAssistant(chatID int64) int {
	num := rand.Intn(NumAssistants) + 1
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.assistantDB.UpdateOne(ctx, bson.M{"_id": chatID},
		bson.M{"$set": bson.M{"num": num}},
		options.Update().SetUpsert(true))
	m.mu.Lock()
	m.assistant[chatID] = num
	m.mu.Unlock()
	return num
}

func (m *MongoDB) GetAssistantNum(chatID int64) int {
	m.mu.RLock()
	if n, ok := m.assistant[chatID]; ok {
		m.mu.RUnlock()
		return n
	}
	m.mu.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var doc struct {
		Num int `bson:"num"`
	}
	if err := m.assistantDB.FindOne(ctx, bson.M{"_id": chatID}).Decode(&doc); err == nil && doc.Num > 0 {
		m.mu.Lock()
		m.assistant[chatID] = doc.Num
		m.mu.Unlock()
		return doc.Num
	}
	return m.SetAssistant(chatID)
}

// ─── BLACKLIST ─────────────────────────────────────────────────

func (m *MongoDB) AddBlacklist(id int64) {
	m.mu.Lock()
	m.Blacklisted = append(m.Blacklisted, id)
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key := "bl_users"
	field := "user_ids"
	if id < 0 {
		key, field = "bl_chats", "chat_ids"
	}
	_, _ = m.cacheDB.UpdateOne(ctx, bson.M{"_id": key},
		bson.M{"$addToSet": bson.M{field: id}},
		options.Update().SetUpsert(true))
}

func (m *MongoDB) DelBlacklist(id int64) {
	m.mu.Lock()
	for i, v := range m.Blacklisted {
		if v == id {
			m.Blacklisted = append(m.Blacklisted[:i], m.Blacklisted[i+1:]...)
			break
		}
	}
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key, field := "bl_users", "user_ids"
	if id < 0 {
		key, field = "bl_chats", "chat_ids"
	}
	_, _ = m.cacheDB.UpdateOne(ctx, bson.M{"_id": key}, bson.M{"$pull": bson.M{field: id}})
}

func (m *MongoDB) IsBlacklisted(id int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.Blacklisted {
		if v == id {
			return true
		}
	}
	return false
}

// ─── CHATS ─────────────────────────────────────────────────────

func (m *MongoDB) IsChat(chatID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.Chats {
		if c == chatID {
			return true
		}
	}
	return false
}

func (m *MongoDB) AddChat(chatID int64) {
	if m.IsChat(chatID) {
		return
	}
	m.mu.Lock()
	m.Chats = append(m.Chats, chatID)
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.chatsDB.InsertOne(ctx, bson.M{"_id": chatID})
}

func (m *MongoDB) RmChat(chatID int64) {
	m.mu.Lock()
	for i, c := range m.Chats {
		if c == chatID {
			m.Chats = append(m.Chats[:i], m.Chats[i+1:]...)
			break
		}
	}
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.chatsDB.DeleteOne(ctx, bson.M{"_id": chatID})
}

func (m *MongoDB) GetChats() []int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Chats
}

// ─── CMD DELETE ───────────────────────────────────────────────

func (m *MongoDB) GetCmdDelete(chatID int64) bool {
	m.mu.RLock()
	v := m.cmdDelete[chatID]
	m.mu.RUnlock()
	return v
}

func (m *MongoDB) SetCmdDelete(chatID int64, del bool) {
	m.mu.Lock()
	m.cmdDelete[chatID] = del
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.chatsDB.UpdateOne(ctx, bson.M{"_id": chatID},
		bson.M{"$set": bson.M{"cmd_delete": del}},
		options.Update().SetUpsert(true))
}

// ─── LANGUAGE ─────────────────────────────────────────────────

func (m *MongoDB) GetLang(chatID int64) string {
	m.mu.RLock()
	if l, ok := m.Lang[chatID]; ok {
		m.mu.RUnlock()
		return l
	}
	m.mu.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var doc struct {
		Lang string `bson:"lang"`
	}
	if err := m.langDB.FindOne(ctx, bson.M{"_id": chatID}).Decode(&doc); err == nil && doc.Lang != "" {
		m.mu.Lock()
		m.Lang[chatID] = doc.Lang
		m.mu.Unlock()
		return doc.Lang
	}
	return "en"
}

func (m *MongoDB) SetLang(chatID int64, code string) {
	m.mu.Lock()
	m.Lang[chatID] = code
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.langDB.UpdateOne(ctx, bson.M{"_id": chatID},
		bson.M{"$set": bson.M{"lang": code}},
		options.Update().SetUpsert(true))
}

// ─── LOGGER ───────────────────────────────────────────────────

func (m *MongoDB) IsLogger() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loggerOn
}

func (m *MongoDB) SetLogger(on bool) {
	m.mu.Lock()
	m.loggerOn = on
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.cacheDB.UpdateOne(ctx, bson.M{"_id": "logger"},
		bson.M{"$set": bson.M{"status": on}},
		options.Update().SetUpsert(true))
}

// ─── PLAY MODE ────────────────────────────────────────────────

func (m *MongoDB) GetPlayMode(chatID int64) bool {
	m.mu.RLock()
	v := m.adminPlay[chatID]
	m.mu.RUnlock()
	return v
}

func (m *MongoDB) SetPlayMode(chatID int64, adminOnly bool) {
	m.mu.Lock()
	m.adminPlay[chatID] = adminOnly
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.chatsDB.UpdateOne(ctx, bson.M{"_id": chatID},
		bson.M{"$set": bson.M{"admin_play": adminOnly}},
		options.Update().SetUpsert(true))
}

// ─── SUDO ─────────────────────────────────────────────────────

func (m *MongoDB) AddSudo(userID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.cacheDB.UpdateOne(ctx, bson.M{"_id": "sudoers"},
		bson.M{"$addToSet": bson.M{"user_ids": userID}},
		options.Update().SetUpsert(true))
}

func (m *MongoDB) DelSudo(userID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.cacheDB.UpdateOne(ctx, bson.M{"_id": "sudoers"},
		bson.M{"$pull": bson.M{"user_ids": userID}})
}

func (m *MongoDB) GetSudoers() []int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var doc struct {
		UserIDs []int64 `bson:"user_ids"`
	}
	_ = m.cacheDB.FindOne(ctx, bson.M{"_id": "sudoers"}).Decode(&doc)
	return doc.UserIDs
}

// ─── USERS ────────────────────────────────────────────────────

func (m *MongoDB) IsUser(userID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.Users {
		if u == userID {
			return true
		}
	}
	return false
}

func (m *MongoDB) AddUser(userID int64) {
	if m.IsUser(userID) {
		return
	}
	m.mu.Lock()
	m.Users = append(m.Users, userID)
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.usersDB.InsertOne(ctx, bson.M{"_id": userID})
}

func (m *MongoDB) GetUsers() []int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Users
}

// ─── ADMINS ───────────────────────────────────────────────────

func (m *MongoDB) GetAdmins(chatID int64) []int64 {
	m.mu.RLock()
	admins := m.adminList[chatID]
	m.mu.RUnlock()
	return admins
}

func (m *MongoDB) SetAdmins(chatID int64, admins []int64) {
	m.mu.Lock()
	m.adminList[chatID] = admins
	m.mu.Unlock()
}

// ─── CACHE LOADING ────────────────────────────────────────────

func (m *MongoDB) loadCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load chats
	cur, err := m.chatsDB.Find(ctx, bson.M{})
	if err == nil {
		defer cur.Close(ctx)
		for cur.Next(ctx) {
			var doc struct {
				ID int64 `bson:"_id"`
			}
			if cur.Decode(&doc) == nil {
				m.Chats = append(m.Chats, doc.ID)
			}
		}
	}

	// Load users
	cur2, err := m.usersDB.Find(ctx, bson.M{})
	if err == nil {
		defer cur2.Close(ctx)
		for cur2.Next(ctx) {
			var doc struct {
				ID int64 `bson:"_id"`
			}
			if cur2.Decode(&doc) == nil {
				m.Users = append(m.Users, doc.ID)
			}
		}
	}

	// Load blacklisted chats
	var blChats struct {
		ChatIDs []int64 `bson:"chat_ids"`
	}
	if m.cacheDB.FindOne(ctx, bson.M{"_id": "bl_chats"}).Decode(&blChats) == nil {
		m.Blacklisted = append(m.Blacklisted, blChats.ChatIDs...)
	}

	// Load logger flag
	var logDoc struct {
		Status bool `bson:"status"`
	}
	if m.cacheDB.FindOne(ctx, bson.M{"_id": "logger"}).Decode(&logDoc) == nil {
		m.loggerOn = logDoc.Status
	}

	log.Printf("[db] Cache loaded: %d chats, %d users, %d blacklisted.",
		len(m.Chats), len(m.Users), len(m.Blacklisted))
}
