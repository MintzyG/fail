package fail

import (
	"fmt"
	"strings"
	"sync"
)

// TranslationRegistry holds all translations
type TranslationRegistry struct {
	mu   sync.RWMutex
	data map[string]map[string]string // locale -> id.String() -> template
}

// RegisterTranslations adds translations for a locale on the global registry
func RegisterTranslations(locale string, msgs map[ErrorID]string) {
	global.mu.Lock()
	defer global.mu.Unlock()

	if global.localization.data == nil {
		global.localization.data = make(map[string]map[string]string)
	}

	if _, exists := global.localization.data[locale]; !exists {
		global.localization.data[locale] = make(map[string]string)
	}

	for id, msg := range msgs {
		global.localization.data[locale][id.String()] = msg
	}
}

// RegisterTranslations adds translations for a locale in a specific registry
func (r *Registry) RegisterTranslations(locale string, msgs map[ErrorID]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.localization.data == nil {
		r.localization.data = make(map[string]map[string]string)
	}

	if _, exists := r.localization.data[locale]; !exists {
		r.localization.data[locale] = make(map[string]string)
	}

	for id, msg := range msgs {
		r.localization.data[locale][id.String()] = msg
	}
}

// SetDefaultLocale sets the fallback locale for the global registry
func SetDefaultLocale(locale string) {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.defaultLocale = locale
}

// SetDefaultLocale sets the fallback locale for the specific registry
func (r *Registry) SetDefaultLocale(locale string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultLocale = locale
}

// Localize resolves the translated message template for this error's locale
// Stores result in e.Message, returns *Error for chaining
func (e *Error) Localize() *Error {
	locale := e.resolveLocale()
	e.Message = resolveTemplate(e, locale)
	return e
}

// Render formats the error's message template with its arguments
// Stores result in e.Message, returns *Error for chaining
func (e *Error) Render() *Error {
	// Ensure we have localized template first
	if e.Message == "" {
		_ = e.Localize()
	}

	template := e.Message

	// No formatting needed
	if !strings.Contains(template, "%") {
		return e
	}

	args := e.effectiveArgs()

	if len(args) == 0 {
		_ = e.AddMeta("fail.render_warning", "template has placeholders but no args provided")
		return e
	}

	e.Message = safeSprintf(e, template, args...)
	return e
}

// GetLocalized returns the localized message template as string (read-only, no modification)
func (e *Error) GetLocalized() string {
	locale := e.resolveLocale()
	return resolveTemplate(e, locale)
}

// GetRendered returns the fully rendered message as string (read-only, no modification)
func (e *Error) GetRendered() string {
	template := e.GetLocalized()

	if !strings.Contains(template, "%") {
		return template
	}

	args := e.effectiveArgs()
	if len(args) == 0 {
		return template
	}

	return safeSprintf(e, template, args...)
}

func (e *Error) resolveLocale() string {
	if e.Locale != "" {
		return e.Locale
	}

	reg := e.registry
	if reg == nil {
		reg = global
	}

	reg.mu.RLock()
	def := reg.defaultLocale
	reg.mu.RUnlock()

	if def != "" {
		return def
	}
	return "en-US"
}

func resolveTemplate(e *Error, locale string) string {
	// Try requested locale first
	e.registry.localization.mu.RLock()
	if trMap, ok := e.registry.localization.data[locale]; ok {
		if msg, ok := trMap[e.ID.String()]; ok {
			e.registry.localization.mu.RUnlock()
			return msg
		}
	}
	e.registry.localization.mu.RUnlock()

	// Fallback to error's default message
	return e.Message
}

func (e *Error) effectiveArgs() []any {
	if len(e.Args) > 0 {
		return e.Args
	}

	// Try to get default args from definition
	reg := e.registry
	if reg == nil {
		reg = global
	}

	reg.mu.RLock()
	def, exists := reg.definitions[e.ID]
	reg.mu.RUnlock()

	if exists && len(def.DefaultArgs) > 0 {
		return def.DefaultArgs
	}

	return nil
}

func safeSprintf(e *Error, format string, args ...any) (result string) {
	defer func() {
		if r := recover(); r != nil {
			_ = e.AddMeta("fail.render_error", fmt.Sprintf("panic during sprintf: %v", r))
			result = format
		}
	}()

	return fmt.Sprintf(format, args...)
}
