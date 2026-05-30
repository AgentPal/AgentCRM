package i18n

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

//go:embed messages_en.json
var enJSON []byte

//go:embed messages_zh.json
var zhJSON []byte

var (
	mu           sync.RWMutex
	current      = "en"
	messages     map[string]map[string]string // lang -> key -> value
	initOnce     sync.Once
)

// lazyInit ensures i18n is initialized before any T/Tn call.
// This is necessary because init() functions in other packages may call
// T() before any explicit initialization in main().
func lazyInit() {
	initOnce.Do(func() {
		m := make(map[string]map[string]string)
		for lang, data := range map[string][]byte{"en": enJSON, "zh": zhJSON} {
			var kv map[string]string
			if err := json.Unmarshal(data, &kv); err != nil {
				panic("i18n: corrupt embedded json: " + err.Error())
			}
			m[lang] = kv
		}
		mu.Lock()
		messages = m
		mu.Unlock()
	})
}

// MustInit loads embedded message files and selects the active language.
// The cmdLang parameter is the --lang CLI flag value (may be empty).
// Priority: cmdLang > AGENTCRM_LANG env > "en".
// Config.json integration is deferred — applied later via SetLang once
// config is loaded.
func MustInit(cmdLang string) {
	m := make(map[string]map[string]string)
	for lang, data := range map[string][]byte{"en": enJSON, "zh": zhJSON} {
		var kv map[string]string
		if err := json.Unmarshal(data, &kv); err != nil {
			panic("i18n: corrupt embedded json: " + err.Error())
		}
		m[lang] = kv
	}

	mu.Lock()
	messages = m
	mu.Unlock()

	lang := "en"
	switch {
	case cmdLang != "":
		lang = cmdLang
	case os.Getenv("AGENTCRM_LANG") != "":
		lang = os.Getenv("AGENTCRM_LANG")
	}
	// TODO: integrate config.json "lang" field — deferred because config
	// is loaded after store init, which happens after MustInit. Root
	// command handler should call SetLang(cfg.Lang) once config is
	// available, when no higher-priority source (flag, env) was set.

	mu.Lock()
	current = lang
	mu.Unlock()
}

// SetLang switches the active language at runtime.
// Accepts "en" or "zh". Returns error for unsupported codes.
func SetLang(lang string) error {
	switch lang {
	case "en", "zh":
		mu.Lock()
		current = lang
		mu.Unlock()
		return nil
	default:
		return fmt.Errorf("unsupported language: %s", lang)
	}
}

// CurrentLang returns the active language code.
func CurrentLang() string {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// T translates key to the active language with optional fmt-style args.
// Fallback chain: current lang -> en -> "[missing: <key>]".
func T(key string, args ...interface{}) string {
	lazyInit()
	mu.RLock()
	defer mu.RUnlock()

	msg := lookup(key, current)
	if msg != "" {
		return format(msg, args)
	}

	// Fallback to en
	if current != "en" {
		msg = lookup(key, "en")
		if msg != "" {
			return format(msg, args)
		}
	}

	return "[missing: " + key + " ]"
}

// Tn translates with number-aware English pluralization.
// For n==1 the base key is used; for n!=1 the key+".plural" variant.
// Chinese always uses the base key (no grammatical plural).
//
// TODO: add .plural sample keys + TestTn_Singular/TestTn_Plural when
// the first real plural message is introduced. Current implementation
// has the selection logic but no test coverage with actual plural keys.
func Tn(key string, n int, args ...interface{}) string {
	lazyInit()
	mu.RLock()
	defer mu.RUnlock()

	lang := current
	if lang == "en" && n != 1 {
		if msg := lookup(key+".plural", "en"); msg != "" {
			return format(msg, append([]interface{}{n}, args...))
		}
	}
	if msg := lookup(key, lang); msg != "" {
		return format(msg, append([]interface{}{n}, args...))
	}
	if lang != "en" {
		if msg := lookup(key, "en"); msg != "" {
			return format(msg, append([]interface{}{n}, args...))
		}
	}
	return "[missing: " + key + " ]"
}

// lookup returns the message for key in lang, or "" if missing.
func lookup(key, lang string) string {
	m, ok := messages[lang]
	if !ok {
		return ""
	}
	return m[key]
}

// format applies fmt.Sprintf if args are provided.
func format(msg string, args []interface{}) string {
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

// KnownLangs returns the list of loaded language codes.
func KnownLangs() []string {
	mu.RLock()
	defer mu.RUnlock()
	langs := make([]string, 0, len(messages))
	for l := range messages {
		langs = append(langs, l)
	}
	return langs
}

// AllKeys returns every key registered in the en messages.
// Used by TestJSONIntegrity to verify key parity across languages.
func AllKeys() []string {
	mu.RLock()
	defer mu.RUnlock()
	keys := make([]string, 0, len(messages["en"]))
	for k := range messages["en"] {
		keys = append(keys, k)
	}
	return keys
}

// MessagesForLang returns all key-value pairs for a language.
// Used by TestNoForbiddenTerms to scan message values.
func MessagesForLang(lang string) map[string]string {
	mu.RLock()
	defer mu.RUnlock()
	m := messages[lang]
	if m == nil {
		return nil
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// Placeholders returns the set of fmt placeholders (%s, %d, %v, etc.)
// used in a message value. Used by TestJSONIntegrity to verify that
// both languages use the same placeholders for the same key.
func Placeholders(val string) []string {
	var out []string
	for i := 0; i < len(val)-1; i++ {
		if val[i] == '%' {
			next := val[i+1]
			if next == '%' {
				i++ // skip escaped %
				continue
			}
			if next == '(' {
				// %[1]s style
				j := i + 2
				for j < len(val) && val[j] != ')' {
					j++
				}
				if j < len(val) && j+1 < len(val) {
					out = append(out, "%[...]"+string(val[j+1]))
				}
				i = j + 1
				continue
			}
			out = append(out, "%"+string(next))
			i++
		}
	}
	return out
}

// PlaceholderString returns a sorted, deduplicated, comma-separated string
// of placeholders for easy comparison.
func PlaceholderString(val string) string {
	phs := Placeholders(val)
	if len(phs) == 0 {
		return ""
	}
	// Simple dedup
	seen := make(map[string]bool)
	var unique []string
	for _, p := range phs {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}
	// Sort
	for i := 0; i < len(unique); i++ {
		for j := i + 1; j < len(unique); j++ {
			if unique[i] > unique[j] {
				unique[i], unique[j] = unique[j], unique[i]
			}
		}
	}
	return strings.Join(unique, ",")
}
