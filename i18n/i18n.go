package i18n

import "sync"

const (
	LangTJ = "tj"
	LangRU = "ru"
	LangEN = "en"
)

var (
	mu      sync.RWMutex
	storage = make(map[string]map[string]string)
)

// Register adds translations to the global registry.
// Called from init() functions — safe for concurrent use after init phase.
func Register(data map[string]map[string]string) {
	mu.Lock()
	defer mu.Unlock()

	for code, translations := range data {
		storage[code] = translations
	}
}

// Get returns the translation for the given code and language.
// Fallback order: requested lang → RU → code itself.
func Get(code, lang string) string {
	mu.RLock()
	defer mu.RUnlock()

	translations, ok := storage[code]
	if !ok {
		return code
	}

	if msg, ok := translations[lang]; ok && msg != "" {
		return msg
	}

	if msg, ok := translations[LangRU]; ok && msg != "" {
		return msg
	}

	return code
}
