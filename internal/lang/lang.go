package lang

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// LangMap holds string keys → string values for one locale.
type LangMap map[string]string

// Get formats a locale string by replacing {0}, {1}, … with args.
func (l LangMap) Get(key string, args ...interface{}) string {
	tpl, ok := l[key]
	if !ok {
		return key
	}
	for i, a := range args {
		tpl = strings.ReplaceAll(tpl, fmt.Sprintf("{%d}", i), fmt.Sprint(a))
	}
	return tpl
}

// Manager loads and serves locale dictionaries.
type Manager struct {
	languages map[string]LangMap
	codes     map[string]string
}

var M *Manager

func Load(localeDir string) *Manager {
	mgr := &Manager{
		languages: make(map[string]LangMap),
		codes: map[string]string{
			"ar": "العربية",
			"de": "Deutsch",
			"en": "English",
			"es": "Español",
			"fr": "Français",
			"hi": "हिन्दी",
			"ja": "日本語",
			"my": "မြန်မာဘာသာ",
			"pa": "ਪੰਜਾਬੀ",
			"pt": "Português",
			"ru": "Русский",
			"zh": "中文",
		},
	}

	entries, err := os.ReadDir(localeDir)
	if err != nil {
		log.Printf("[lang] Could not read locale dir %s: %v", localeDir, err)
		// load fallback empty english
		mgr.languages["en"] = LangMap{}
		M = mgr
		return mgr
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		code := strings.TrimSuffix(e.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(localeDir, e.Name()))
		if err != nil {
			log.Printf("[lang] Failed to read %s: %v", e.Name(), err)
			continue
		}
		var lm LangMap
		if err := json.Unmarshal(data, &lm); err != nil {
			log.Printf("[lang] Failed to parse %s: %v", e.Name(), err)
			continue
		}
		mgr.languages[code] = lm
	}
	log.Printf("[lang] Loaded locales: %s", strings.Join(mgr.codes_loaded(), ", "))
	M = mgr
	return mgr
}

func (m *Manager) codes_loaded() []string {
	var out []string
	for k := range m.languages {
		out = append(out, k)
	}
	return out
}

// Get returns the LangMap for a language code (falls back to "en").
func (m *Manager) Get(code string) LangMap {
	if lm, ok := m.languages[code]; ok {
		return lm
	}
	if lm, ok := m.languages["en"]; ok {
		return lm
	}
	return LangMap{}
}

// GetLanguages returns the map of code→name for all loaded locales.
func (m *Manager) GetLanguages() map[string]string {
	out := make(map[string]string)
	for code := range m.languages {
		if name, ok := m.codes[code]; ok {
			out[code] = name
		} else {
			out[code] = code
		}
	}
	return out
}
