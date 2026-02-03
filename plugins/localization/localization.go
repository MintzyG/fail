package localization

import (
	"sync"

	"github.com/MintzyG/fail/v3"
)

// Localizer holds all translations
type Localizer struct {
	mu   sync.RWMutex
	data map[string]map[string]string // locale -> id.String() -> template
}

// New creates a new localization registry
func New() *Localizer {
	return &Localizer{
		data: make(map[string]map[string]string),
	}
}

// Localize returns the localized template for the given error ID and locale
// Returns empty string if no translation exists
func (r *Localizer) Localize(id fail.ErrorID, locale string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.data == nil {
		return ""
	}

	if trMap, ok := r.data[locale]; ok {
		if msg, ok := trMap[id.String()]; ok {
			return msg
		}
	}
	return ""
}

// RegisterLocalization adds a single translation (for AddLocalization)
func (r *Localizer) RegisterLocalization(id fail.ErrorID, locale string, template string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.data == nil {
		r.data = make(map[string]map[string]string)
	}

	if _, exists := r.data[locale]; !exists {
		r.data[locale] = make(map[string]string)
	}

	if _, exists := r.data[locale][id.String()]; exists {
		return
	}

	r.data[locale][id.String()] = template
}

// RegisterLocalizations adds multiple translations for a locale (for bulk operations)
func (r *Localizer) RegisterLocalizations(locale string, translations map[fail.ErrorID]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.data == nil {
		r.data = make(map[string]map[string]string)
	}

	if _, exists := r.data[locale]; !exists {
		r.data[locale] = make(map[string]string)
	}

	for id, msg := range translations {
		if _, exists := r.data[locale][id.String()]; !exists {
			r.data[locale][id.String()] = msg
		}
	}
}
